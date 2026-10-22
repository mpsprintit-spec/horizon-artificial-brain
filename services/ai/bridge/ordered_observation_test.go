package bridge

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

func TestManagedHTTPObservationRetainsOrderedContextForLearnedPrediction(t *testing.T) {
	r := runtime.NewBrainRuntime(knowledge.NewBrain())
	at := time.Unix(100, 0).UTC()
	r.SetClock(runtime.NewFixedClock(at))
	for i, context := range []string{"naku melo", "melo naku"} {
		_, err := r.LearnExperience(learning.Experience{ContextSequence: strings.Fields(context), Sequence: []string{"cue", []string{"tiva", "luma"}[i]}, Weight: .9, Confidence: .9}, at)
		if err != nil {
			t.Fatal(err)
		}
	}
	graph, err := r.InspectBrain()
	if err != nil {
		t.Fatal(err)
	}
	a, _ := graph.Graph.NodeForSurface("tiva")
	b, _ := graph.Graph.NodeForSurface("luma")
	checkpoint := filepath.Join(t.TempDir(), "trained.json")
	if err = r.Checkpoint(checkpoint); err != nil {
		t.Fatal(err)
	}
	for side, context := range []string{"naku melo", "melo naku"} {
		clone := runtime.NewBrainRuntime(knowledge.NewBrain())
		if err = clone.RestoreCheckpoint(checkpoint); err != nil {
			t.Fatal(err)
		}
		owner, err := runtime.NewRuntimeSupervisor(clone)
		if err != nil {
			t.Fatal(err)
		}
		defer owner.Close()
		server, err := NewManaged(owner, Config{})
		if err != nil {
			t.Fatal(err)
		}
		envelope := runtime.ObservationEnvelope{SchemaVersion: 1, BrainIdentity: runtime.BrainIdentity, EventID: "http-ordered", Timestamp: at.Add(time.Second), Modality: "text", Source: "browser", Tokens: []string{"cue"}, ContextTokens: strings.Fields(context), Provenance: runtime.ObservationProvenance{Source: "browser", Modality: "text", Adapter: "test"}}
		body, _ := json.Marshal(envelope)
		request := httptest.NewRequest(http.MethodPost, "/v1/observations", bytes.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		server.Handler().ServeHTTP(response, request)
		if response.Code != http.StatusAccepted {
			t.Fatalf("HTTP observation status=%d body=%s", response.Code, response.Body.String())
		}
		path := filepath.Join(t.TempDir(), "after-http.json")
		if err = owner.WithMutation(func(active *runtime.BrainRuntime) error { return active.Checkpoint(path) }); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var s runtime.BrainSnapshot
		if err = json.Unmarshal(data, &s); err != nil {
			t.Fatal(err)
		}
		margin := s.Activation.LastPrediction[a] - s.Activation.LastPrediction[b]
		if side == 0 && margin < .2 || side == 1 && margin > -.2 {
			t.Fatalf("HTTP boundary collapsed context %q: margin=%g", context, margin)
		}
	}
}
