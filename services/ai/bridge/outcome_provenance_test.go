package bridge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

func TestManagedOutcomeIngressPreservesProvenanceAcrossRecovery(t *testing.T) {
	at := time.Unix(100, 0).UTC()
	r := runtime.NewBrainRuntime(knowledge.NewBrain())
	r.SetClock(runtime.NewFixedClock(at))
	log, err := runtime.OpenEventLog(filepath.Join(t.TempDir(), "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	r.SetEventLog(log)
	g, err := r.GroundVectorAt(knowledge.NewNeuralVector([]float64{.2, .8}), at)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if err = r.RegisterActionBinding(runtime.ActionBinding{RequestID: fmt.Sprint(i), TargetNodeIDs: []knowledge.NodeID{g.NodeID}}); err != nil {
			t.Fatal(err)
		}
	}
	supervisor, err := runtime.NewRuntimeSupervisor(r)
	if err != nil {
		t.Fatal(err)
	}
	defer supervisor.Close()
	server, err := NewManaged(supervisor, Config{Token: "outcome-test"})
	if err != nil {
		t.Fatal(err)
	}
	post := func(i int, source, group, cause string) {
		t.Helper()
		// Use actual wire field names, including the optional snake_case lineage.
		body := fmt.Sprintf(`{"RequestID":%q,"Source":%q,"Success":true,"Observation":["seen"],"Reliability":0.9,"ObservedAt":"1970-01-01T00:01:40Z","episode_id":"scene","independence_group":%q,"causal_group":%q}`, fmt.Sprint(i), source, group, cause)
		req := httptest.NewRequest(http.MethodPost, "/v1/outcomes", bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer outcome-test")
		response := httptest.NewRecorder()
		server.Handler().ServeHTTP(response, req)
		if response.Code != http.StatusAccepted {
			t.Fatalf("HTTP status=%d: %s", response.Code, response.Body.String())
		}
	}
	snapshot := func() runtime.BrainSnapshot {
		t.Helper()
		path := filepath.Join(t.TempDir(), "checkpoint.json")
		if err = supervisor.With(func(active *runtime.BrainRuntime) error { return active.Checkpoint(path) }); err != nil {
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
		return s
	}
	post(0, "camera", "camera-declaration", "common-cause")
	post(1, "relay", "relay-declaration", "common-cause")
	first := snapshot()
	p := first.EvidenceLedger.Provenance[g.NodeID]
	if first.EvidenceLedger.ByNode[g.NodeID].IndependentSources != 1 || len(p.Records) != 2 || p.Records[0].EpisodeID != "scene" || p.Records[1].CausalGroup != "common-cause" {
		t.Fatal("HTTP ingress lost or inflated supplied provenance")
	}
	prefix := filepath.Join(t.TempDir(), "prefix.json")
	if err = supervisor.With(func(active *runtime.BrainRuntime) error { return active.Checkpoint(prefix) }); err != nil {
		t.Fatal(err)
	}
	if err = supervisor.RecoverFromCheckpoint(log, prefix); err != nil {
		t.Fatal(err)
	}
	post(2, "independent", "another-origin", "another-cause")
	second := snapshot()
	if second.EvidenceLedger.ByNode[g.NodeID].IndependentSources != 2 {
		t.Fatal("recovered HTTP ingress lost independent confirmation")
	}
	post(2, "independent", "another-origin", "another-cause")
	third := snapshot()
	if !reflect.DeepEqual(second.EvidenceLedger, third.EvidenceLedger) || !bytes.Equal(second.Brain, third.Brain) {
		t.Fatal("HTTP retry repeated evidence or neural promotion")
	}
}
