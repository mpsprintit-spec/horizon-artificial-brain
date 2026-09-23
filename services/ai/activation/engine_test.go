package activation

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

func TestActivationEmergesFromLearnedGraph(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	learning.NewLearningUnit(kb).Assimilate("api panas", 0.9, 0.8)
	result := NewEngine(kb).Activate([]string{"api"}, 3)
	if !result.Converged {
		t.Fatal("expected activation to converge")
	}
	panas := kb.Fetch("panas")
	if result.Activations[panas.ID] <= 0 {
		t.Fatal("expected activation to spread to co-activated node")
	}
}


func TestLearnedCausalTraceInfluencesPrediction(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	target := kb.Store("target")
	outcome := kb.Store("outcome")
	pattern := kb.Patterns.LearnTrace(
		[]knowledge.PatternStep{
			{NodeID: target.ID, Position: 0, Activation: 1},
			{NodeID: outcome.ID, Position: 1, Activation: 1},
		},
		nil, outcome.ID, 0.9, 0.9,
	)
	if pattern == nil {
		t.Fatal("expected causal pattern")
	}
	e := NewEngine(kb)
	e.ActivateWith(Request{
		StimulusTokens: []string{"target"},
		Cycles: 1,
		Now: time.Now().UTC(),
	})
	prediction := e.PredictionSnapshot()
	if prediction.State[outcome.ID] <= 0 {
		t.Fatal("learned causal trace did not influence next prediction")
	}
}
