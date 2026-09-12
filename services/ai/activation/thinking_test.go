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

func TestThinkDoesNotInventInitialStimulus(t *testing.T) {
	engine := NewEngine(knowledge.NewKnowledgeBase())
	result := engine.Think(2)
	if len(result.Activations) != 0 {
		t.Fatal("expected no thought before the brain has an internal state")
	}
}
