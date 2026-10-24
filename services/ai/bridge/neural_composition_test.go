package bridge

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/core"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

func TestManagedCompositionTeachingRecoveryAndObservation(t *testing.T) {
	h := core.NewHorizonEngine()
	dir := t.TempDir()
	log, err := runtime.OpenEventLog(filepath.Join(dir, "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	if err := h.Supervisor().AttachEventLog(log); err != nil {
		t.Fatal(err)
	}
	at := time.Now().UTC().Add(-time.Second)
	if err := h.InitializeNeuralComposition(knowledge.DefaultCompositionConfig(), filepath.Join(dir, "rollback.json"), at); err != nil {
		t.Fatal(err)
	}
	checkpoint := filepath.Join(dir, "composition.json")
	if err := h.Supervisor().WithMutation(func(r *runtime.BrainRuntime) error { return r.Checkpoint(checkpoint) }); err != nil {
		t.Fatal(err)
	}
	server, err := NewManaged(h.Supervisor(), Config{Token: "teaching-test", AllowCompositionTraining: true, EventLogPath: filepath.Join(dir, "events.jsonl")})
	if err != nil {
		t.Fatal(err)
	}
	scene := knowledge.CompositionScene{ObservedAt: at, Point: []float64{0, 0, -1}, Patches: []knowledge.CompositionPatch{{Values: []float64{.8, -.8, .8, -1, -1}, XY: []float64{0, 0}}}}
	after := scene
	after.ObservedAt = at.Add(time.Millisecond)
	e := knowledge.CompositionExposure{ID: "one-lesson", Source: "teacher", EpisodeID: "episode", CausalGroup: "contact", IndependenceGroup: "teacher-1", Reliability: 1, Input: knowledge.CompositionInput{Words: [][]float64{knowledge.EncodeObservation("sample", "language").Values}, Scene: scene}, Motor: []float64{0, 0, -1, 1}, After: after}
	post := func(path string, value any) *httptest.ResponseRecorder {
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		request := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(data))
		request.Header.Set("Authorization", "Bearer teaching-test")
		out := httptest.NewRecorder()
		server.Handler().ServeHTTP(out, request)
		return out
	}
	response := post("/v1/composition/exposures", e)
	if response.Code != http.StatusAccepted {
		t.Fatalf("teaching: %d %s", response.Code, response.Body)
	}
	if err := h.Supervisor().RecoverFromCheckpoint(log, checkpoint); err != nil {
		t.Fatal(err)
	}
	duplicate := post("/v1/composition/exposures", e)
	if duplicate.Code != http.StatusUnprocessableEntity {
		t.Fatal("recovered teaching ID applied twice")
	}
	request := runtime.ObservationEnvelope{EventID: "written-query", Source: "user", Modality: "language", Timestamp: time.Now().UTC(), Tokens: []string{"sample"}}
	response = post("/v1/observations", request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("observation: %d %s", response.Code, response.Body)
	}
	var body struct {
		Composition *knowledge.CompositionReadout `json:"composition"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body.Composition == nil {
		t.Fatalf("ordinary observation response omitted native composition: %v", err)
	}
	bad := map[string]any{"teacher_target": "forbidden-label"}
	if response := post("/v1/composition/exposures", bad); response.Code != http.StatusBadRequest {
		t.Fatal("teaching accepted a foreign semantic field")
	}
	closed, err := NewManaged(h.Supervisor(), Config{Token: "teaching-test"})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/composition/exposures", bytes.NewBufferString("{}"))
	req.Header.Set("Authorization", "Bearer teaching-test")
	out := httptest.NewRecorder()
	closed.Handler().ServeHTTP(out, req)
	if out.Code != http.StatusForbidden {
		t.Fatal("teaching was enabled by default")
	}
}
