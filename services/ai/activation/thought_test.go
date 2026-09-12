package activation

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestThinkProducesInternalPrediction(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	a := kb.Store("a")
	b := kb.Store("b")
	c := kb.Store("c")
	kb.Connect(a, b, 0.9, 1, false)
	kb.Connect(b, c, 0.9, 1, false)

	e := NewEngine(kb)
	e.ActivateWith(Request{StimulusTokens: []string{"a"}, Cycles: 1, Now: time.Unix(1000, 0).UTC()})
	thought := e.ThinkWithPrediction(1)

	if len(thought.Prediction.State) == 0 {
		t.Fatal("expected internal prediction state")
	}
	if len(thought.Result.Activations) == 0 {
		t.Fatal("expected actual internal state")
	}
	if thought.PredictionError < 0 || thought.PredictionError > 1 {
		t.Fatalf("prediction error out of range: %v", thought.PredictionError)
	}
}

func TestThinkDoesNotRequireExternalStimulus(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	a := kb.Store("a")
	b := kb.Store("b")
	kb.Connect(a, b, 0.8, 1, false)

	e := NewEngine(kb)
	e.Activate([]string{"a"}, 1)
	first := e.Think(1)
	second := e.Think(1)

	if len(first.Activations) == 0 || len(second.Activations) == 0 {
		t.Fatal("expected thought to continue from internal state")
	}
}

func TestEmptyBrainDoesNotInventThought(t *testing.T) {
	e := NewEngine(knowledge.NewKnowledgeBase())
	result := e.Think(3)
	if len(result.Activations) != 0 {
		t.Fatal("empty brain should not invent an internal stimulus")
	}
}
