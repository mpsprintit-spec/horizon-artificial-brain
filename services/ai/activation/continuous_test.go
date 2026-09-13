package activation

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

func TestThinkContinuouslyMaintainsInternalProcess(t *testing.T) {
	brain := knowledge.NewBrain()
	engine := NewEngine(brain)
	now := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)

	a, err := brain.ProjectVectorPopulation(knowledge.NewNeuralVector([]float64{-1, -1, 1, 1}), 0.90, 4)
	if err != nil { t.Fatal(err) }
	b, err := brain.ProjectVectorPopulation(knowledge.NewNeuralVector([]float64{-1, 1, -1, 1}), 0.90, 4)
	if err != nil { t.Fatal(err) }
	c, err := brain.ProjectVectorPopulation(knowledge.NewNeuralVector([]float64{1, -1, -1, 1}), 0.90, 4)
	if err != nil { t.Fatal(err) }
	if err := engine.LearnPopulationTransition(a, b, 0.8, 1, now); err != nil { t.Fatal(err) }
	if err := engine.LearnPopulationTransition(b, c, 0.8, 1, now); err != nil { t.Fatal(err) }
	if _, err := engine.ActivatePopulation(a, 1, now); err != nil { t.Fatal(err) }

	thoughts := engine.ThinkContinuously(4, 1)
	if len(thoughts) != 4 { t.Fatalf("expected four thought steps, got %d", len(thoughts)) }
	first := thoughts[0].Activations
	changed := false
	for i, thought := range thoughts {
		if len(thought.Activations) == 0 || len(thought.Prediction.State) == 0 { t.Fatalf("thought step %d lost internal state", i) }
		if i > 0 && stateDifference(first, thought.Activations) > 0.0001 { changed = true }
	}
	if !changed { t.Fatal("continuous thinking produced identical internal state") }
}

func TestThinkContinuouslyFromLearnedExperience(t *testing.T) {
	brain := knowledge.NewBrain()
	unit := learning.NewLearningUnit(brain)
	engine := NewEngine(brain)
	now := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)
	experiences := []learning.Experience{
		{Sequence: []string{"saya", "ingin", "belajar"}, Weight: .9, Confidence: .9},
		{Sequence: []string{"saya", "ingin", "belajar", "tentang", "air"}, Weight: .9, Confidence: .9},
		{Sequence: []string{"air", "itu", "dingin"}, Weight: .9, Confidence: .9},
		{Sequence: []string{"air", "menjadi", "panas"}, Weight: .9, Confidence: .9},
		{Sequence: []string{"es", "mencair", "ketika", "hangat"}, Weight: .9, Confidence: .9},
		{Sequence: []string{"setelah", "makan", "saya", "minum"}, Weight: .9, Confidence: .9},
		{Sequence: []string{"sebelum", "tidur", "saya", "membaca"}, Weight: .9, Confidence: .9},
		{Sequence: []string{"kemudian", "dia", "pulang"}, Weight: .9, Confidence: .9},
		{Sequence: []string{"kemarin", "saya", "pergi", "ke", "pasar"}, Weight: .9, Confidence: .9},
		{Sequence: []string{"besok", "saya", "akan", "bekerja"}, Weight: .9, Confidence: .9},
	}
	for _, experience := range experiences { unit.LearnExperience(experience, now) }
	if len(brain.Registry.Nodes()) < 12 { t.Fatalf("too little learned structure: %d nodes", len(brain.Registry.Nodes())) }
	if len(brain.Patterns.All()) < len(experiences) { t.Fatal("learned experiences did not form temporal patterns") }

	seed := brain.Fetch("saya")
	if seed == nil { t.Fatal("learned seed missing") }
	population := knowledge.ProjectionPopulation{Units: []knowledge.PopulationUnit{{NodeID: seed.ID, Activation: 1}}}
	if _, err := engine.ActivatePopulation(population, 1, now); err != nil { t.Fatal(err) }
	thoughts := engine.ThinkContinuously(12, 1)
	if len(thoughts) != 12 { t.Fatalf("expected 12 internal steps, got %d", len(thoughts)) }

	seen := map[knowledge.NodeID]bool{}
	for i, thought := range thoughts {
		if len(thought.Activations) == 0 || len(thought.Prediction.State) == 0 { t.Fatalf("thought step %d lost continuity", i) }
		for id, level := range thought.Activations { if level > engine.Threshold { seen[id] = true } }
	}
	if len(seen) < 3 { t.Fatalf("internal process explored too little learned structure: %d units", len(seen)) }
}
