package activation

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

func TestRecurrentCompletionUsesDynamicSynapsesWithoutPatternIndex(t *testing.T) {
	brain := knowledge.NewBrain()
	learner := learning.NewLearningUnit(brain)
	engine := NewEngine(brain)
	now := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)

	for _, sequence := range [][]string{
		{"saya", "ingin", "belajar", "tentang", "air"},
		{"saya", "ingin", "belajar", "tentang", "air"},
	} {
		learner.LearnExperience(learning.Experience{Sequence: sequence, Weight: 1, Confidence: 1}, now)
	}

	cue := []knowledge.PatternStep{
		{NodeID: brain.Fetch("saya").ID, Position: 0, Activation: 1},
		{NodeID: brain.Fetch("ingin").ID, Position: 1, Activation: 1},
		{NodeID: brain.Fetch("belajar").ID, Position: 2, Activation: 1},
		{NodeID: brain.Fetch("tentang").ID, Position: 3, Activation: 1},
	}

	next := engine.RecurrentNext(cue, now)
	if len(next) == 0 {
		t.Fatal("expected recurrent continuation")
	}
	air := brain.Fetch("air")
	if next[0] != air.ID {
		t.Fatalf("expected strongest continuation air=%d, got %d", air.ID, next[0])
	}

	result := engine.RecurrentCompletion(cue, 1, now)
	if result.Activations[air.ID] <= 0 {
		t.Fatal("recurrent completion did not activate learned continuation")
	}
}

func TestRecurrentNextSeparatesLearnedBranchesBySynapticStrength(t *testing.T) {
	brain := knowledge.NewBrain()
	learner := learning.NewLearningUnit(brain)
	engine := NewEngine(brain)
	now := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)

	learner.LearnExperience(learning.Experience{Sequence: []string{"saya", "ingin", "belajar", "tentang", "air"}, Weight: 1, Confidence: 1}, now)
	learner.LearnExperience(learning.Experience{Sequence: []string{"saya", "ingin", "belajar", "tentang", "listrik"}, Weight: 0.4, Confidence: 0.8}, now)

	cue := []knowledge.PatternStep{
		{NodeID: brain.Fetch("saya").ID, Position: 0, Activation: 1},
		{NodeID: brain.Fetch("ingin").ID, Position: 1, Activation: 1},
		{NodeID: brain.Fetch("belajar").ID, Position: 2, Activation: 1},
		{NodeID: brain.Fetch("tentang").ID, Position: 3, Activation: 1},
	}
	next := engine.RecurrentNext(cue, now)
	if len(next) < 2 {
		t.Fatalf("expected both learned branches, got %d", len(next))
	}
	if next[0] != brain.Fetch("air").ID {
		t.Fatalf("stronger learned branch should rank first, got node %d", next[0])
	}
}
