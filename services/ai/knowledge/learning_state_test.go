package knowledge_test

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestPredictionErrorUpdatesPersistentLearningState(t *testing.T) {
	brain := knowledge.NewBrain()
	now := time.Unix(123, 0).UTC()

	brain.RecordLearningSignal(0.75, now)

	if got := brain.BrainState.PredictionState["last_error"]; got != 0.75 {
		t.Fatalf("prediction error not persisted: %v", got)
	}
	if got := brain.BrainState.PlasticityState["error_drive"]; got != 0.75 {
		t.Fatalf("plasticity error drive not updated: %v", got)
	}
	if got := brain.BrainState.PlasticityState["current"]; got <= brain.BrainState.PlasticityState["baseline"] {
		t.Fatalf("plasticity did not increase after prediction error: %v", got)
	}
	if got := brain.BrainState.MemoryState["last_priority"]; got <= brain.BrainState.MemoryState["retention"] {
		t.Fatalf("memory priority did not increase after prediction error: %v", got)
	}
	if got := brain.BrainState.PredictionState["last_update_unix"]; got != float64(now.UnixNano()) {
		t.Fatalf("prediction timestamp not persisted: %v", got)
	}
}


func TestMemoryDynamicsDecaysOldSynapseWithoutDeletingIt(t *testing.T) {
	brain := knowledge.NewBrain()
	a := brain.Store("a")
	b := brain.Store("b")
	if a == nil || b == nil {
		t.Fatal("failed to create neural units")
	}
	brain.Connect(a, b, 0.8, 0.8, false)
	before := a.Synapses[b.ID][0].Dynamic.Weight
	if before <= 0 {
		t.Fatalf("expected learned synapse weight, got %v", before)
	}

	old := time.Now().UTC().Add(-72 * time.Hour)
	a.Synapses[b.ID][0].Dynamic.LastModification = old
	a.Synapses[b.ID][0].Dynamic.LastActivation = old
	brain.BrainState.MemoryState["retention"] = 0.2

	brain.ApplyMemoryDynamics(time.Now().UTC())

	synapse := a.Synapses[b.ID][0]
	if synapse == nil {
		t.Fatal("synapse was deleted")
	}
	if synapse.Dynamic.Weight >= before {
		t.Fatalf("old synapse did not decay: before=%v after=%v", before, synapse.Dynamic.Weight)
	}
	if synapse.Dynamic.Weight <= 0 {
		t.Fatalf("synapse decayed to invalid zero state")
	}
}
