package bridge

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

func managedBridgeSensor(id string, at time.Time) runtime.SensorObservation {
	return runtime.SensorObservation{
		SampleID: id,
		Source: "test-camera",
		Modality: "vision",
		Timestamp: at,
		ChannelStates: []float64{0.2, -0.4, 0.8},
		Confidence: 0.9,
	}
}

func TestManagedBridgeWritesOnlyToRecoveredRuntime(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	checkpoint := filepath.Join(t.TempDir(), "checkpoint.json")
	log, err := runtime.OpenEventLog(path)
	if err != nil { t.Fatal(err) }
	defer log.Close()
	old := runtime.NewBrainRuntime(knowledge.NewBootstrapBrain())
	old.SetEventLog(log)
	owner, err := runtime.NewRuntimeOwner(old, log)
	if err != nil { t.Fatal(err) }
	server, err := NewManaged(owner, Config{EventLogPath: path})
	if err != nil { t.Fatal(err) }
	at := time.Date(2026, 10, 12, 8, 0, 0, 0, time.UTC)
	for _, tc := range []struct{ id string; stamp time.Time }{
		{"sensor-before-checkpoint", at},
		{"sensor-after-checkpoint", at.Add(time.Second)},
	} {
		if err := owner.WithRuntime(func(active *runtime.BrainRuntime) error {
			_, err := runtime.NewCognitiveOrchestrator(active).ProcessSensorObservation(managedBridgeSensor(tc.id, tc.stamp))
			return err
		}); err != nil { t.Fatal(err) }
		if tc.id == "sensor-before-checkpoint" {
			if err := owner.WithRuntime(func(active *runtime.BrainRuntime) error { return active.Checkpoint(checkpoint) }); err != nil { t.Fatal(err) }
		}
	}
	if err := owner.RecoverFromCheckpoint(checkpoint); err != nil { t.Fatalf("checkpoint handover: %v", err) }
	if !old.Retired() { t.Fatal("bridge old runtime is not retired") }

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), "\"state_revision\":2") {
		t.Fatalf("health did not select recovered runtime: status=%d body=%s", res.Code, res.Body.String())
	}

	body := strings.NewReader(`{"schema_version":1,"brain_identity":"horizon-primary-brain","event_id":"bridge-after-handover","timestamp":"2026-10-12T08:00:03Z","modality":"vision","source":"test-camera","tokens":["motion"]}`)
	req = httptest.NewRequest(http.MethodPost, "/v1/observations", body)
	res = httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusAccepted {
		t.Fatalf("managed bridge failed to process post-handover event: status=%d body=%s", res.Code, res.Body.String())
	}
	if got := old.LastSequence(); got != 2 {
		t.Fatalf("retired runtime advanced through bridge: %d", got)
	}
	if err := owner.WithRuntime(func(active *runtime.BrainRuntime) error {
		if active == old || active.LastSequence() != 3 {
			t.Fatalf("bridge did not route event to new runtime: old=%t seq=%d", active==old, active.LastSequence())
		}
		return nil
	}); err != nil { t.Fatal(err) }
	events, err := runtime.ReadEventLog(path)
	if err != nil || len(events)!=3 {
		t.Fatalf("bridge journal missing post-handover event: err=%v count=%d",err,len(events))
	}
}

func TestManagedBridgeReportsRecoveryRequiredViaHealth(t *testing.T) {
	// A closed log simulates failed persistence; the raw runtime is the
	// instance owned by the bridge but is not exposed as a write fallback.
	path := filepath.Join(t.TempDir(), "events.jsonl")
	log, err := runtime.OpenEventLog(path)
	if err != nil { t.Fatal(err) }
	r := runtime.NewBrainRuntime(knowledge.NewBootstrapBrain())
	r.SetEventLog(log)
	owner, err := runtime.NewRuntimeOwner(r, log)
	if err != nil { t.Fatal(err) }
	server, err := NewManaged(owner, Config{})
	if err != nil { t.Fatal(err) }
	if err := log.Close(); err != nil { t.Fatal(err) }
	sensor := managedBridgeSensor("failed-intent", time.Date(2026, 10, 12, 8, 0, 0, 0, time.UTC))
	if err := owner.WithRuntime(func(active *runtime.BrainRuntime) error {
		_, err := runtime.NewCognitiveOrchestrator(active).ProcessSensorObservation(sensor)
		return err
	}); !errors.Is(err, runtime.ErrRuntimeRecoveryRequired) {
		t.Fatalf("closed journal did not poison sensor runtime: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("health advertised poisoned runtime as healthy: %d %s", res.Code, res.Body.String())
	}
	req = httptest.NewRequest(http.MethodPost,"/v1/observations",
		strings.NewReader(`{"event_id":"after-poison","timestamp":"2026-10-12T08:00:02Z","modality":"vision","source":"test-camera","tokens":["blocked"]}`))
	res = httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("bridge accepted poisoned runtime: status=%d body=%s", res.Code,res.Body.String())
	}
}
