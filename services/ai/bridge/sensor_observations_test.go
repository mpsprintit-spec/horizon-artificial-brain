package bridge

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

func TestSensorIngressRejectsMixedOrMalformedPayloadBeforeMutation(t *testing.T) {
	rt := runtime.NewBrainRuntime(nil)
	server, err := New(rt, Config{})
	if err != nil {
		t.Fatal(err)
	}
	valid := `{"modality":"vision","source":"test","sample_id":"bad-frame","timestamp":"2026-10-18T14:00:00Z","confidence":0.5,"fields":[{"field_id":"left:rgb","group_id":"left","kind":"light.rgb-v1","values":[0.2,0.4,0.6]}]}`
	for _, scenario := range []struct {
		name, body string
		status     int
	}{
		{"unknown-payload", strings.TrimSuffix(valid, "}") + `,"payload":{"label":"cat"}}`, http.StatusBadRequest},
		{"multiple-objects", valid + valid, http.StatusBadRequest},
		{"mixed-flat", strings.TrimSuffix(valid, "}") + `,"channel_states":[0.5]}`, http.StatusUnprocessableEntity},
		{"out-of-range", strings.Replace(valid, "[0.2,0.4,0.6]", "[2,0.4,0.6]", 1), http.StatusUnprocessableEntity},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			res := httptest.NewRecorder()
			server.Handler().ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/v1/sensor-observations", strings.NewReader(scenario.body)))
			if res.Code != scenario.status || rt.LastSequence() != 0 || rt.RecoveryRequired() {
				t.Fatalf("invalid request status=%d sequence=%d: %s", res.Code, rt.LastSequence(), res.Body.String())
			}
		})
	}
}

func TestManagedSensorIngressPersistsFieldsReplaysAndFollowsHandover(t *testing.T) {
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
	server, err := NewManaged(supervisor, Config{Token: "sensor-test", BrainMemoryPath: filepath.Join(t.TempDir(), "brain.json")})
	if err != nil {
		t.Fatal(err)
	}
	sensor := runtime.SensorObservation{Modality: "vision", Source: "browser-camera", SampleID: "numeric-frame-1", Timestamp: time.Date(2026, 10, 18, 14, 0, 0, 0, time.UTC), Confidence: .5,
		Fields: []runtime.SensorField{{FieldID: "left:rgb", GroupID: "left", Kind: "light.rgb-v1", Values: []float64{.8, -.4, .2}, Coordinates: []float64{-1, -1, 0, 1}}, {FieldID: "right:rgb", GroupID: "right", Kind: "light.rgb-v1", Values: []float64{.8, -.4, .2}, Coordinates: []float64{0, -1, 1, 1}}}}
	post := func(authorized bool, expected int) {
		t.Helper()
		body, err := json.Marshal(sensor)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodPost, "/v1/sensor-observations", bytes.NewReader(body))
		if authorized {
			req.Header.Set("Authorization", "Bearer sensor-test")
		}
		res := httptest.NewRecorder()
		server.Handler().ServeHTTP(res, req)
		if res.Code != expected {
			t.Fatalf("sensor ingress status=%d want=%d: %.300s", res.Code, expected, res.Body.String())
		}
		if expected == http.StatusAccepted {
			var response struct {
				Revision uint64 `json:"state_revision"`
				Bindings []any  `json:"sensor_bindings"`
			}
			if err := json.Unmarshal(res.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if len(response.Bindings) != 2 {
				t.Fatal("HTTP response lost local field bindings")
			}
		}
	}
	post(false, http.StatusUnauthorized)
	if original.LastSequence() != 0 {
		t.Fatal("unauthorized sensor request reached neural substrate")
	}
	post(true, http.StatusAccepted)
	checkpoint := filepath.Join(t.TempDir(), "checkpoint.json")
	if err := original.Checkpoint(checkpoint); err != nil {
		t.Fatal(err)
	}
	sensor.SampleID = "numeric-frame-2"
	sensor.Timestamp = sensor.Timestamp.Add(time.Second)
	post(true, http.StatusAccepted)
	before, err := original.InspectBrain()
	if err != nil {
		t.Fatal(err)
	}
	if err := supervisor.RecoverFromCheckpoint(journal, checkpoint); err != nil {
		t.Fatal(err)
	}
	if err := supervisor.With(func(active *runtime.BrainRuntime) error {
		after, err := active.InspectBrain()
		if err == nil && !bytes.Equal(before.Graph.CanonicalBytes(), after.Graph.CanonicalBytes()) {
			t.Fatal("typed HTTP sensor state changed across handover")
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	sensor.SampleID = "numeric-frame-3"
	sensor.Timestamp = sensor.Timestamp.Add(time.Second)
	post(true, http.StatusAccepted)
	if original.LastSequence() != 2 {
		t.Fatal("Bridge sent sensor input to retired runtime")
	}
	events, err := runtime.ReadEventLog(journalPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 3 {
		t.Fatalf("unexpected sensor journal length %d", len(events))
	}
	for _, event := range events {
		if event.Event == nil || event.Event.SensorObservation == nil || len(event.Event.SensorObservation.Fields) != 2 || len(event.Event.Stimulus) != 0 {
			t.Fatal("HTTP sensor observation was hashed or tokenized")
		}
	}
}
