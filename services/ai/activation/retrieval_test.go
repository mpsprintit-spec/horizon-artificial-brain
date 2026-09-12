package activation

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// TestInternalRecallWithoutExternalStimulus demonstrates the next cognitive
// substrate property: an experience can remain in the internal state and be
// reactivated without presenting the original external token again.
func TestInternalRecallWithoutExternalStimulus(t *testing.T) {
	memory := knowledge.NewKnowledgeBase()
	a := memory.Store("alpha")
	b := memory.Store("beta")
	memory.Connect(a, b, 0.9, 0.9, false)

	engine := NewEngine(memory)
	engine.Activate([]string{"alpha"}, 2)

	// Internal thought receives no new stimulus. It must operate from the
	// persisted internal state rather than looking up a token again.
	thought := engine.ThinkWithPrediction(2)
	if len(thought.Activations) == 0 {
		t.Fatal("internal thought produced no activation")
	}
	if thought.Activations[a.ID] <= 0 && thought.Activations[b.ID] <= 0 {
		t.Fatal("internal thought lost the learned internal state")
	}
}

func TestExperienceReinforcesExistingStructure(t *testing.T) {
	memory := knowledge.NewKnowledgeBase()
	a := memory.Store("alpha")
	b := memory.Store("beta")
	memory.Connect(a, b, 0.4, 0.5, false)

	before := a.OutboundAll()[0].Dynamic.Weight
	engine := NewEngine(memory)

	for i := 0; i < 3; i++ {
		engine.ActivateWith(Request{
			StimulusTokens: []string{"alpha", "beta"},
			Cycles:         1,
			Now:            time.Now().UTC(),
		})
	}

	after := a.OutboundAll()[0].Dynamic.Weight
	if after < before {
		t.Fatalf("expected repeated experience not to weaken connection: before=%v after=%v", before, after)
	}
}
