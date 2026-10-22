package core

import (
	"bytes"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
	"github.com/project-horizon/horizon-core/services/ai/perception"
	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

func TestManagedFacadeUsesNumericWrittenRoutesAfterMigrationAndRecovery(t *testing.T) {
	brain := knowledge.NewBrain()
	r := runtime.NewBrainRuntime(brain)
	at := time.Date(2026, 10, 21, 3, 0, 0, 0, time.UTC)
	r.SetClock(runtime.NewFixedClock(at))
	owner, err := runtime.NewRuntimeSupervisor(r)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = owner.Close() })
	log, err := runtime.OpenEventLog(filepath.Join(t.TempDir(), "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = log.Close() })
	if err := owner.AttachEventLog(log); err != nil {
		t.Fatal(err)
	}
	backup := filepath.Join(t.TempDir(), "pre-migration.json")
	err = owner.WithMutation(func(rt *runtime.BrainRuntime) error {
		plan, err := rt.PlanNeuralWrittenGrounding()
		if err != nil {
			return err
		}
		if err := rt.ApplyNeuralWrittenGrounding(plan, backup, at); err != nil {
			return err
		}
		for i := 0; i < 4; i++ {
			if _, err := rt.LearnExperience(learning.Experience{Sequence: []string{"naku", "melo", "tiva"}, Weight: .8, Confidence: .8}, at); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	h := &HorizonEngine{supervisor: owner, Perception: fixedPerception{signal: perception.PerceptionSignal{Source: "test", Modality: "text", Tokens: []string{"naku", "melo"}, ObservedAt: at.Add(time.Second)}}}
	checkpoint := filepath.Join(t.TempDir(), "numeric.json")
	if err := owner.WithMutation(func(rt *runtime.BrainRuntime) error { return rt.Checkpoint(checkpoint) }); err != nil {
		t.Fatal(err)
	}
	// Controlled test fixture ablation only; no live mutable callback is exposed.
	if err := brain.RestoreSurfaceAnnotations(nil); err != nil {
		t.Fatal(err)
	}
	out, err := h.Pulse("naku melo")
	if err != nil {
		t.Fatal(err)
	}
	if len(out.GroundedRepresentations) != 2 || len(brain.Registry.Nodes()) != 3 {
		t.Fatal("facade did not use the numeric written populations")
	}
	wantCanonical := inspectForTest(t, h).Graph.CanonicalBytes()
	// Recover the same pre-query state, then repeat through the managed facade.
	// The prefix checkpoint is replayed with the query suffix in its journal.
	if err := owner.RecoverFromCheckpoint(log, checkpoint); err != nil {
		t.Fatal(err)
	}
	if err := owner.WithMutation(func(rt *runtime.BrainRuntime) error {
		contract, err := rt.WrittenGroundingContract()
		if err != nil {
			return err
		}
		if contract != knowledge.NeuralWrittenEncoding {
			t.Fatal("owner handover lost numeric serving")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	after := inspectForTest(t, h)
	if !bytes.Equal(wantCanonical, after.Graph.CanonicalBytes()) {
		t.Fatal("managed recovery changed the label-ablated neural continuation")
	}
	if after.Revision != out.State.Sequence {
		t.Fatal("query suffix was not recovered exactly once")
	}
	var ids []knowledge.NodeID
	for _, unit := range after.Graph.NeuralUnits() {
		ids = append(ids, unit.ID)
	}
	if !reflect.DeepEqual(ids, []knowledge.NodeID{1, 2, 3}) {
		t.Fatal("recovery duplicated neural identities")
	}
}
