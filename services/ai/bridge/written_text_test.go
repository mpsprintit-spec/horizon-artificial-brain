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
	testManagedBridgeWrittenContract(t, false)
}

func TestManagedBridgeNumericWrittenContractWithoutLabelsAndAfterRestart(t *testing.T) {
	testManagedBridgeWrittenContract(t, true)
}

func testManagedBridgeWrittenContract(t *testing.T, neural bool) {
	brain := knowledge.NewBrain()
	r := runtime.NewBrainRuntime(brain)
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
		if neural {
			plan, err := active.PlanNeuralWrittenGrounding()
			if err != nil {
				return err
			}
			if err := active.ApplyNeuralWrittenGrounding(plan, filepath.Join(t.TempDir(), "pre-migration.json"), at); err != nil {
				return err
			}
		}
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
		if neural {
			// Controlled ablation on this test Brain, inside its owner lease.
			if err := brain.RestoreSurfaceAnnotations(nil); err != nil {
				return err
			}
		}
		return active.Checkpoint(cp)
	}); err != nil {
		t.Fatal(err)
	}
	config := Config{}
	if neural {
		config.Token = "numeric-written-diagnostic"
	}
	server, err := NewManaged(owner, config)
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
		request := httptest.NewRequest(http.MethodPost, "/v1/observations", bytes.NewReader(body))
		if neural {
			request.Header.Set("Authorization", "Bearer "+config.Token)
		}
		server.Handler().ServeHTTP(response, request)
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
