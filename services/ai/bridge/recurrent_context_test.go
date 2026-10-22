package bridge

import (
	"bytes"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

func TestManagedHTTPSessionsRetainAndSeparateNeuralContext(t *testing.T) {
	r := runtime.NewBrainRuntime(knowledge.NewBrain())
	at := time.Unix(100, 0).UTC()
	r.SetClock(runtime.NewFixedClock(at))
	for _, sequence := range [][]string{{"xa", "cue", "left"}, {"ya", "cue", "right"}} {
		if _, err := r.LearnExperience(learning.Experience{Sequence: sequence, Weight: .9, Confidence: .9}, at); err != nil {
			t.Fatal(err)
		}
	}
	graph, err := r.InspectBrain()
	if err != nil {
		t.Fatal(err)
	}
	a, _ := graph.Graph.NodeForSurface("left")
	b, _ := graph.Graph.NodeForSurface("right")
	prefix := filepath.Join(t.TempDir(), "trained.json")
	if err = r.Checkpoint(prefix); err != nil {
		t.Fatal(err)
	}
	for _, sameSession := range []bool{true, false} {
		clone := runtime.NewBrainRuntime(knowledge.NewBrain())
		if err = clone.RestoreCheckpoint(prefix); err != nil {
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
		for i, token := range []string{"xa", "cue"} {
			session := "one"
			if i == 1 && !sameSession {
				session = "two"
			}
			envelope := runtime.ObservationEnvelope{SchemaVersion: 1, BrainIdentity: runtime.BrainIdentity, EventID: token, SessionID: session, Timestamp: at.Add(time.Duration(i+1) * time.Second), Modality: "text", Source: "browser", Tokens: []string{token}, Provenance: runtime.ObservationProvenance{Source: "browser", Modality: "text"}}
			data, _ := json.Marshal(envelope)
			request := httptest.NewRequest(http.MethodPost, "/v1/observations", bytes.NewReader(data))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			server.Handler().ServeHTTP(response, request)
			if response.Code != http.StatusAccepted {
				t.Fatalf("HTTP status=%d body=%s", response.Code, response.Body.String())
			}
		}
		path := filepath.Join(t.TempDir(), "after.json")
		if err = owner.WithMutation(func(active *runtime.BrainRuntime) error { return active.Checkpoint(path) }); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var snapshot runtime.BrainSnapshot
		if err = json.Unmarshal(data, &snapshot); err != nil {
			t.Fatal(err)
		}
		margin := snapshot.Activation.LastPrediction[a] - snapshot.Activation.LastPrediction[b]
		if sameSession && margin < .2 || !sameSession && math.Abs(margin) > .03 {
			t.Fatalf("HTTP session context failed: same=%v margin=%g", sameSession, margin)
		}
	}
}
