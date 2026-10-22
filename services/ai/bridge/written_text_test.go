package bridge

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

func TestManagedBridgeTextRetrievesTrainedTargetBeforeAndAfterRestart(t *testing.T) {
	r := runtime.NewBrainRuntime(knowledge.NewBrain())
	owner, err := runtime.NewRuntimeSupervisor(r)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	log, err := runtime.OpenEventLog(filepath.Join(t.TempDir(), "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	if err := owner.AttachEventLog(log); err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 10, 19, 10, 0, 0, 0, time.UTC)
	cp := filepath.Join(t.TempDir(), "trained.json")
	var target knowledge.NodeID
	if err := owner.WithMutation(func(active *runtime.BrainRuntime) error {
		for i := 0; i < 3; i++ {
			if _, err := active.LearnExperience(learning.Experience{Sequence: []string{"naku", "melo", "tiva"}, Weight: .8, Confidence: .8, Source: "same-teacher", Modality: "text", IndependenceGroup: "same-teacher"}, at); err != nil {
				return err
			}
		}
		inspection, err := active.InspectBrain()
		if err != nil {
			return err
		}
		var ok bool
		target, ok = inspection.Graph.NodeForSurface("tiva")
		if !ok {
			t.Fatal("trained target has no detached label")
		}
		return active.Checkpoint(cp)
	}); err != nil {
		t.Fatal(err)
	}
	server, err := NewManaged(owner, Config{})
	if err != nil {
		t.Fatal(err)
	}
	post := func(id string, offset time.Duration) {
		t.Helper()
		body, err := json.Marshal(runtime.ObservationEnvelope{EventID: id, Source: "another-keyboard", Modality: "text", Timestamp: at.Add(offset), Tokens: []string{"NAKU", "MeLo"}})
		if err != nil {
			t.Fatal(err)
		}
		response := httptest.NewRecorder()
		server.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/observations", bytes.NewReader(body)))
		if response.Code != http.StatusAccepted {
			t.Fatalf("Bridge text failed: %d %s", response.Code, response.Body.String())
		}
		var result struct {
			Active []knowledge.NodeID `json:"active_node_ids"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if !slices.Contains(result.Active, target) {
			t.Fatal("held-out target missing from production endpoint's neural readout")
		}
		if err := owner.With(func(active *runtime.BrainRuntime) error {
			inspection, err := active.InspectBrain()
			if err != nil {
				return err
			}
			nodes, _, _ := inspection.Graph.Counts()
			if nodes != 3 {
				t.Fatalf("text endpoint created %d nodes instead of reusing the three trained units", nodes)
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	post("before-restart", time.Second)
	if err := owner.RecoverFromCheckpoint(log, cp); err != nil {
		t.Fatal(err)
	}
	post("after-restart", 2*time.Second)
}
