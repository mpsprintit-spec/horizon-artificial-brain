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


func TestPredictionErrorChangesLearnedTraceStrength(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	target := kb.Store("target")
	outcome := kb.Store("outcome")
	unexpected := kb.Store("unexpected")
	pattern := kb.Patterns.LearnTrace(
		[]knowledge.PatternStep{
			{NodeID: target.ID, Position: 0, Activation: 1},
			{NodeID: outcome.ID, Position: 1, Activation: 1},
		},
		nil, outcome.ID, 0.9, 0.9,
	)
	beforeWeight := pattern.Weight
	beforeConfidence := pattern.Confidence

	e := NewEngine(kb)
	e.ActivateWith(Request{
		StimulusTokens: []string{"target"},
		Cycles: 1,
		Now: time.Now().UTC(),
	})
	e.ApplyPredictionErrorPlasticity(
		map[knowledge.NodeID]float64{target.ID: 0.8, outcome.ID: 0.8},
		map[knowledge.NodeID]float64{target.ID: 0.8, outcome.ID: 0.0, unexpected.ID: 0.9},
		0.8,
		time.Now().UTC().Add(time.Second),
	)

	patterns := kb.Patterns.ResultsFor(outcome.ID)
	if len(patterns) != 1 {
		t.Fatalf("expected one causal trace after reconsolidation, got %d", len(patterns))
	}
	if patterns[0].Weight >= beforeWeight {
		t.Fatalf("expected prediction error to weaken learned trace: before=%v after=%v", beforeWeight, patterns[0].Weight)
	}
	if patterns[0].Confidence >= beforeConfidence {
		t.Fatalf("expected prediction error to reduce trace confidence: before=%v after=%v", beforeConfidence, patterns[0].Confidence)
	}
}


func TestKnownLanguageStimulusReactivatesCanonicalUnitWithoutPopulationGrowth(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	a := kb.Store("saya")
	b := kb.Store("ingin")
	c := kb.Store("belajar")
	e := NewEngine(kb)
	now := time.Unix(100, 0).UTC()

	first := e.ActivateWith(Request{
		StimulusTokens: []string{"saya", "ingin", "belajar"},
		Cycles:        2,
		Now:           now,
	})
	if !first.Converged {
		t.Fatal("expected first activation to converge")
	}
	if got := len(kb.Registry.Nodes()); got != 3 {
		t.Fatalf("expected canonical substrate to remain at 3 nodes, got %d", got)
	}
	if got := len(kb.ProjectionPopulations); got != 0 {
		t.Fatalf("known language stimulus must not create projection populations, got %d", got)
	}
	if first.Activations[a.ID] <= 0 || first.Activations[b.ID] <= 0 || first.Activations[c.ID] <= 0 {
		t.Fatalf("expected canonical units to activate: %v", first.Activations)
	}

	second := e.ActivateWith(Request{
		StimulusTokens: []string{"saya", "ingin", "belajar"},
		Cycles:        2,
		Now:           now.Add(time.Second),
	})
	if !second.Converged {
		t.Fatal("expected repeated activation to converge")
	}
	if got := len(kb.Registry.Nodes()); got != 3 {
		t.Fatalf("repeated language activation created new substrate nodes: got %d", got)
	}
	if got := len(kb.ProjectionPopulations); got != 0 {
		t.Fatalf("repeated known language activation created projection populations: got %d", got)
	}
	if second.Activations[a.ID] <= 0 || second.Activations[b.ID] <= 0 || second.Activations[c.ID] <= 0 {
		t.Fatalf("expected repeated activation to reuse canonical units: %v", second.Activations)
	}
}

func TestUnknownLanguageStimulusStillUsesDistributedProjectionPath(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	e := NewEngine(kb)

	result := e.ActivateWith(Request{
		StimulusTokens: []string{"air"},
		Cycles:        1,
		Now:           time.Unix(200, 0).UTC(),
	})
	if !result.Converged {
		t.Fatal("expected unknown stimulus activation to converge")
	}
	if got := len(kb.Registry.Nodes()); got != 4 {
		t.Fatalf("expected unknown experience to create one distributed population of 4 nodes, got %d", got)
	}
	if got := len(kb.ProjectionPopulations); got != 1 {
		t.Fatalf("expected one distributed projection population, got %d", got)
	}
}


func TestPlasticityMutatesDynamicsWithoutCreatingNeuralTopology(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	a := kb.Store("a")
	b := kb.Store("b")
	kb.Connect(a, b, 0.6, 0.8, false)

	e := NewEngine(kb)
	now := time.Unix(300, 0).UTC()
	beforeNodes := len(kb.Registry.Nodes())
	beforePopulations := len(kb.ProjectionPopulations)
	beforeSynapses := len(a.OutboundAll())

	e.ApplyActivityPlasticity(
		map[knowledge.NodeID]float64{a.ID: 1},
		map[knowledge.NodeID]float64{a.ID: 0.8, b.ID: 0.9},
		now,
	)
	e.ApplyPredictionErrorPlasticity(
		map[knowledge.NodeID]float64{a.ID: 0.8, b.ID: 0.8},
		map[knowledge.NodeID]float64{a.ID: 0.8, b.ID: 0.1},
		0.9,
		now.Add(time.Millisecond),
	)

	if got := len(kb.Registry.Nodes()); got != beforeNodes {
		t.Fatalf("plasticity created neural nodes: before=%d after=%d", beforeNodes, got)
	}
	if got := len(kb.ProjectionPopulations); got != beforePopulations {
		t.Fatalf("plasticity created projection populations: before=%d after=%d", beforePopulations, got)
	}
	if got := len(a.OutboundAll()); got != beforeSynapses {
		t.Fatalf("plasticity changed synapse topology: before=%d after=%d", beforeSynapses, got)
	}
}

func TestStimulusNodeActivationDoesNotImplyCertainty(t *testing.T) {
	brain := knowledge.NewBrain()
	node := brain.Store("uji")
	engine := NewEngine(brain)
	result := engine.ActivateWith(Request{
		StimulusNodeIDs: []knowledge.NodeID{node.ID},
		Cycles: 1,
	})
	if result.Activations[node.ID] <= 0 {
		t.Fatalf("expected stimulus node to activate, got %.3f", result.Activations[node.ID])
	}
	if result.Confidence[node.ID] >= 1 {
		t.Fatalf("stimulus activation must not imply certainty, got %.3f", result.Confidence[node.ID])
	}
}
