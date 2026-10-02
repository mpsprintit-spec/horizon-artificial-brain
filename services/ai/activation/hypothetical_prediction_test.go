package activation

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestPredictOutcomeFromNodesDoesNotMutateActivationState(t *testing.T) {
	brain := knowledge.NewBrain()
	source := brain.Store("source")
	target := brain.Store("target")
	engine := NewEngine(brain)
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)

	result := engine.ActivateWith(Request{
		StimulusNodeIDs: []knowledge.NodeID{source.ID},
		Cycles:          1,
		Now:             now,
	})
	before := cloneState(result.Activations)

	prediction := engine.PredictOutcomeFromNodes([]knowledge.NodeID{target.ID}, now.Add(time.Second), 1)
	if len(prediction.State) == 0 {
		t.Fatal("expected hypothetical prediction")
	}

	after := engine.CurrentState()
	if len(after) != len(before) {
		t.Fatalf("hypothetical prediction mutated state size: got %d want %d", len(after), len(before))
	}
	for id, want := range before {
		if got := after[id]; got != want {
			t.Fatalf("hypothetical prediction mutated node %d: got %v want %v", id, got, want)
		}
	}
}
