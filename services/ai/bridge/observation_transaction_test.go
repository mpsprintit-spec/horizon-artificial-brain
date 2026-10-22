package bridge

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

func TestManagedObservationDuplicateAndRecoveryUseWholeCommand(t *testing.T) {
	journalPath := filepath.Join(t.TempDir(), "events.jsonl")
	journal, err := runtime.OpenEventLog(journalPath)
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
	original := runtime.NewBrainRuntime(nil)
	original.SetEventLog(journal)
	supervisor, err := runtime.NewRuntimeSupervisor(original)
	if err != nil {
		t.Fatal(err)
	}
	defer supervisor.Close()
	server, err := NewManaged(supervisor, Config{})
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 10, 19, 9, 0, 0, 0, time.UTC)
	post := func(id, token string, expected int) {
		t.Helper()
		body, err := json.Marshal(runtime.ObservationEnvelope{EventID: id, Source: "local", Modality: "text", Timestamp: at, Tokens: []string{token}})
		if err != nil {
			t.Fatal(err)
		}
		response := httptest.NewRecorder()
		server.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/observations", bytes.NewReader(body)))
		if response.Code != expected {
			t.Fatalf("status=%d want=%d: %s", response.Code, expected, response.Body.String())
		}
	}
	post("first", "known", http.StatusAccepted)
	before, err := original.InspectBrain()
	if err != nil {
		t.Fatal(err)
	}
	post("first", "must-not-ground", http.StatusUnprocessableEntity)
	after, err := original.InspectBrain()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before.Graph.CanonicalBytes(), after.Graph.CanonicalBytes()) || original.LastSequence() != 1 {
		t.Fatal("HTTP duplicate changed canonical state")
	}
	checkpoint := filepath.Join(t.TempDir(), "checkpoint.json")
	if err := original.Checkpoint(checkpoint); err != nil {
		t.Fatal(err)
	}
	post("suffix", "changes", http.StatusAccepted)
	want, err := original.InspectBrain()
	if err != nil {
		t.Fatal(err)
	}
	if err := supervisor.RecoverFromCheckpoint(journal, checkpoint); err != nil {
		t.Fatal(err)
	}
	if err := supervisor.With(func(active *runtime.BrainRuntime) error {
		got, err := active.InspectBrain()
		if err == nil && !bytes.Equal(want.Graph.CanonicalBytes(), got.Graph.CanonicalBytes()) {
			t.Fatal("HTTP observation lost cognition on recovery")
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	post("first", "another-duplicate", http.StatusUnprocessableEntity)
	post("next", "continuation", http.StatusAccepted)
	if original.LastSequence() != 2 {
		t.Fatal("HTTP ingress reused retired runtime")
	}
	events, err := runtime.ReadEventLog(journalPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 3 {
		t.Fatal("duplicates reached journal")
	}
	for _, event := range events {
		if event.Type != runtime.EventTypeObservation || event.ObservationCommand == nil || event.Event.Observation == nil {
			t.Fatal("HTTP boundary journaled only a downstream fragment")
		}
	}
}

func TestHTTPObservationClosedWALReturnsUnavailableWithoutGrounding(t *testing.T) {
	brain := knowledge.NewBootstrapBrain()
	r := runtime.NewBrainRuntime(brain)
	journal, err := runtime.OpenEventLog(filepath.Join(t.TempDir(), "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	r.SetEventLog(journal)
	if err := journal.Close(); err != nil {
		t.Fatal(err)
	}
	server, err := New(r, Config{})
	if err != nil {
		t.Fatal(err)
	}
	before, err := brain.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"event_id":"failure","source":"local","modality":"text","timestamp":"2026-10-19T09:00:00Z","tokens":["must-not-ground"]}`)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/observations", bytes.NewReader(body)))
	if response.Code != http.StatusServiceUnavailable || !r.RecoveryRequired() {
		t.Fatalf("uncertain WAL did not close HTTP ingress: %d %s", response.Code, response.Body.String())
	}
	after, err := brain.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) || r.LastSequence() != 0 {
		t.Fatal("failed HTTP request grounded before WAL")
	}
}
