package activation

import (
	"testing"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestThinkContinuesWithoutExternalStimulus(t *testing.T) {
	memory := knowledge.NewKnowledgeBase()
	a := memory.Store("a")
	b := memory.Store("b")
	c := memory.Store("c")
	memory.Connect(a, b, 0.9, 1, false)
	memory.Connect(b, c, 0.9, 1, false)

	engine := NewEngine(memory)
	first := engine.Activate([]string{"a"}, 1)
	if len(first.Activations) == 0 {
		t.Fatal("expected external experience to create internal activation state")
	}

	second := engine.Think(2)
	if len(second.Activations) == 0 {
		t.Fatal("expected internal thought to continue without a new stimulus")
	}
	if second.Resonance <= 0 {
		t.Fatal("expected internal thought to produce non-zero resonance")
	}
}

func TestThinkProducesRecurrentInternalStateChanges(t *testing.T) {
	memory := knowledge.NewKnowledgeBase()
	a := memory.Store("a")
	b := memory.Store("b")
	c := memory.Store("c")
	memory.Connect(a, b, 0.95, 1, false)
	memory.Connect(b, c, 0.95, 1, false)
	memory.Connect(c, a, 0.30, 1, false)

	engine := NewEngine(memory)
	engine.Activate([]string{"a"}, 1)

	first := engine.Think(1)
	second := engine.Think(1)
	third := engine.Think(1)

	if stateDifference(first.Activations, second.Activations) <= 0.001 {
		t.Fatal("expected the internal state to change between thought cycles")
	}
	if stateDifference(second.Activations, third.Activations) <= 0.001 {
		t.Fatal("expected recurrent dynamics to continue changing the internal state")
	}
}

func TestExternalExperienceCanViolatePredictionAndAdaptPath(t *testing.T) {
	memory := knowledge.NewKnowledgeBase()
	a := memory.Store("a")
	b := memory.Store("b")
	c := memory.Store("c")
	memory.Connect(a, b, 0.80, 0.80, false)
	memory.Connect(a, c, 0.20, 0.80, false)

	engine := NewEngine(memory)
	engine.Activate([]string{"a"}, 1)
	thought := engine.ThinkWithPrediction(1)
	if len(thought.Prediction.State) == 0 {
		t.Fatal("expected internal thought to produce a future prediction")
	}

	bSynapse := a.FindDynamicSynapse(b.ID, false)
	before := bSynapse.Dynamic.Weight

	// c is presented as the next actual experience. The previous prediction is
	// retained and therefore can register a mismatch instead of being erased.
	engine.Activate([]string{"c"}, 1)
	after := bSynapse.Dynamic.Weight
	if after >= before {
		t.Fatalf("expected violated prediction path to weaken: before=%v after=%v", before, after)
	}
}

func TestThinkDoesNotInventInitialStimulus(t *testing.T) {
	engine := NewEngine(knowledge.NewKnowledgeBase())
	result := engine.Think(2)
	if len(result.Activations) != 0 {
		t.Fatal("expected no thought before the brain has an internal state")
	}
}
