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


func TestRepeatedCausalExperienceStrengthensPrediction(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	target := kb.Store("target")
	outcome := kb.Store("outcome")
	for i := 0; i < 5; i++ {
		kb.Patterns.LearnTrace(
			[]knowledge.PatternStep{
				{NodeID: target.ID, Position: 0, Activation: 1},
				{NodeID: outcome.ID, Position: 1, Activation: 1},
			},
			nil, outcome.ID, 0.9, 0.9,
		)
	}

	e := NewEngine(kb)
	e.ActivateWith(Request{
		StimulusTokens: []string{"target"},
		Cycles: 1,
		Now: time.Now().UTC(),
	})
	prediction := e.PredictionSnapshot()
	if prediction.State[outcome.ID] <= 0 {
		t.Fatal("repeated causal experience did not produce a prediction")
	}
	patterns := kb.Patterns.ResultsFor(outcome.ID)
	if len(patterns) != 1 {
		t.Fatalf("expected one consolidated causal trace, got %d", len(patterns))
	}
	if patterns[0].Frequency != 5 {
		t.Fatalf("expected repeated experience frequency 5, got %d", patterns[0].Frequency)
	}
}

func TestContradictoryCausalOutcomesRemainAvailable(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	target := kb.Store("target")
	outcomeA := kb.Store("outcome-a")
	outcomeB := kb.Store("outcome-b")

	kb.Patterns.LearnTrace(
		[]knowledge.PatternStep{
			{NodeID: target.ID, Position: 0, Activation: 1},
			{NodeID: outcomeA.ID, Position: 1, Activation: 1},
		},
		nil, outcomeA.ID, 0.9, 0.9,
	)
	kb.Patterns.LearnTrace(
		[]knowledge.PatternStep{
			{NodeID: target.ID, Position: 0, Activation: 1},
			{NodeID: outcomeB.ID, Position: 1, Activation: 1},
		},
		nil, outcomeB.ID, 0.9, 0.9,
	)

	e := NewEngine(kb)
	e.ActivateWith(Request{
		StimulusTokens: []string{"target"},
		Cycles: 1,
		Now: time.Now().UTC(),
	})
	prediction := e.PredictionSnapshot()
	if prediction.State[outcomeA.ID] <= 0 || prediction.State[outcomeB.ID] <= 0 {
		t.Fatalf("contradictory causal outcomes were not both retained in prediction: A=%v B=%v",
			prediction.State[outcomeA.ID], prediction.State[outcomeB.ID])
	}
	patternsA := kb.Patterns.ResultsFor(outcomeA.ID)
	patternsB := kb.Patterns.ResultsFor(outcomeB.ID)
	if len(patternsA) != 1 || len(patternsB) != 1 {
		t.Fatalf("expected both causal traces to remain: A=%d B=%d", len(patternsA), len(patternsB))
	}
}
