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
	if err != nil {
		t.Fatal(err)
	}
	b, err := brain.ProjectVectorPopulation(knowledge.NewNeuralVector([]float64{-1, 1, -1, 1}), 0.90, 4)
	if err != nil {
		t.Fatal(err)
	}
	c, err := brain.ProjectVectorPopulation(knowledge.NewNeuralVector([]float64{1, -1, -1, 1}), 0.90, 4)
	if err != nil {
		t.Fatal(err)
	}

	if err := engine.LearnPopulationTransition(a, b, 0.8, 1, now); err != nil {
		t.Fatal(err)
	}
	if err := engine.LearnPopulationTransition(b, c, 0.8, 1, now); err != nil {
		t.Fatal(err)
	}

	if _, err := engine.ActivatePopulation(a, 1, now); err != nil {
		t.Fatal(err)
	}

	thoughts := engine.ThinkContinuously(4, 1)
	if len(thoughts) != 4 {
		t.Fatalf("expected four internally generated thought steps, got %d", len(thoughts))
	}
	for i, thought := range thoughts {
		if len(thought.Activations) == 0 {
			t.Fatalf("thought step %d produced no internal state", i)
		}
		if len(thought.Prediction.State) == 0 {
			t.Fatalf("thought step %d produced no next-state prediction", i)
		}
	}

	first := thoughts[0].Activations
	changed := false
	for i := 1; i < len(thoughts); i++ {
		if stateDifference(first, thoughts[i].Activations) > 0.0001 {
			changed = true
			break
		}
	}
	if !changed {
		t.Fatal("continuous thinking produced identical internal state at every step")
	}

	if len(c.Units) == 0 {
		t.Fatal("expected learned target population")
	}
	reachedC := false
	for _, thought := range thoughts {
		for _, unit := range c.Units {
			if thought.Activations[unit.NodeID] > 0 {
				reachedC = true
				break
			}
		}
		if reachedC {
			break
		}
	}
	if !reachedC {
		t.Fatal("continuous internal process did not propagate through the learned transition chain")
	}
}

func TestThinkContinuouslyFromFoundationalLanguageExperience(t *testing.T) {
	brain := knowledge.NewBrain()
	unit := learning.NewLearningUnit(brain)
	engine := NewEngine(brain)
	now := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)

	experiences := []learning.Experience{
		{Sequence: []string{"saya", "ingin", "belajar"}, Weight: 0.9, Confidence: 0.9},
		{Sequence: []string{"saya", "ingin", "belajar", "tentang", "air"}, Weight: 0.9, Confidence: 0.9},
		{Sequence: []string{"air", "itu", "dingin"}, Weight: 0.9, Confidence: 0.9},
		{Sequence: []string{"air", "menjadi", "panas"}, Weight: 0.9, Confidence: 0.9},
		{Sequence: []string{"es", "mencair", "ketika", "hangat"}, Weight: 0.9, Confidence: 0.9},
		{Sequence: []string{"setelah", "makan", "saya", "minum"}, Weight: 0.9, Confidence: 0.9},
		{Sequence: []string{"sebelum", "tidur", "saya", "membaca"}, Weight: 0.9, Confidence: 0.9},
		{Sequence: []string{"kemudian", "dia", "pulang"}, Weight: 0.9, Confidence: 0.9},
		{Sequence: []string{"kemarin", "saya", "pergi", "ke", "pasar"}, Weight: 0.9, Confidence: 0.9},
		{Sequence: []string{"besok", "saya", "akan", "bekerja"}, Weight: 0.9, Confidence: 0.9},
	}
	for _, experience := range experiences {
		unit.LearnExperience(experience, now)
	}

	before := len(brain.Registry.Nodes())
	if before < 12 {
		t.Fatalf("foundational experience produced too little learned structure: %d nodes", before)
	}
	if len(brain.Patterns.All()) < len(experiences) {
		t.Fatalf("expected temporal patterns for learned experiences: got %d", len(brain.Patterns.All()))
	}

	seed := brain.Fetch("saya")
	if seed == nil {
		t.Fatal("learned language seed was not stored")
	}
	state := map[knowledge.NodeID]float64{seed.ID: 1}
	confidence := map[knowledge.NodeID]float64{seed.ID: 1}
	state, confidence = engine.advance(state, confidence, now, 1)
	if len(state) == 0 {
		t.Fatal("language experience produced no recurrent state")
	}

	thoughts := engine.ThinkContinuously(8, 1)
	if len(thoughts) != 8 {
		t.Fatalf("expected eight internal thought steps from learned experience, got %d", len(thoughts))
	}

	seen := map[knowledge.NodeID]bool{}
	for _, thought := range thoughts {
		for id, level := range thought.Activations {
			if level > engine.Threshold {
				seen[id] = true
			}
		}
	}
	if len(seen) < 3 {
		t.Fatalf("internal process explored too little learned structure: %d active units", len(seen))
	}

	if brain.Fetch("air") == nil || brain.Fetch("belajar") == nil || brain.Fetch("pasar") == nil {
		t.Fatal("expected learned vocabulary to remain in the shared brain substrate")
	}
}
