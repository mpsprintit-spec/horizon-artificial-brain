package activation

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

func TestCompleteFromCueReactivatesLearnedBranches(t *testing.T) {
	brain := knowledge.NewBrain()
	learner := learning.NewLearningUnit(brain)
	engine := NewEngine(brain)
	now := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)

	for _, sequence := range [][]string{
		{"saya", "ingin", "belajar", "tentang", "air"},
		{"saya", "ingin", "belajar", "tentang", "listrik"},
		{"saya", "ingin", "belajar", "tentang", "air"},
		{"saya", "ingin", "belajar", "tentang", "listrik"},
	} {
		learner.LearnExperience(learning.Experience{Sequence: sequence, Weight: 1, Confidence: 1}, now)
	}

	cue := []knowledge.PatternStep{
		{NodeID: brain.Fetch("saya").ID, Position: 0, Activation: 1},
		{NodeID: brain.Fetch("ingin").ID, Position: 1, Activation: 1},
		{NodeID: brain.Fetch("belajar").ID, Position: 2, Activation: 1},
	}
	matches, result, err := engine.CompleteFromCue(cue, nil, 1, now)
	if err != nil { t.Fatal(err) }
	if len(matches) != 2 { t.Fatalf("expected two learned branches, got %d", len(matches)) }
	if len(result.Activations) == 0 { t.Fatal("completion produced no recurrent activation state") }

	air, listrik := brain.Fetch("air"), brain.Fetch("listrik")
	if result.Activations[air.ID] <= 0 || result.Activations[listrik.ID] <= 0 {
		t.Fatal("completion did not reactivate both learned branch endpoints")
	}
}

func TestCompleteFromCueRejectsUnlearnedContinuation(t *testing.T) {
	brain := knowledge.NewBrain()
	learner := learning.NewLearningUnit(brain)
	engine := NewEngine(brain)
	now := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)
	learner.LearnExperience(learning.Experience{Sequence: []string{"saya", "ingin", "belajar", "tentang", "air"}, Weight: 1, Confidence: 1}, now)

	cue := []knowledge.PatternStep{
		{NodeID: brain.Fetch("saya").ID, Position: 0, Activation: 1},
		{NodeID: brain.Fetch("ingin").ID, Position: 1, Activation: 1},
		{NodeID: brain.Fetch("belajar").ID, Position: 2, Activation: 1},
		{NodeID: brain.Fetch("listrik").ID, Position: 3, Activation: 1},
	}
	matches, _, err := engine.CompleteFromCue(cue, nil, 1, now)
	if err != nil { t.Fatal(err) }
	if len(matches) != 0 { t.Fatalf("unlearned continuation was incorrectly reconstructed: %d matches", len(matches)) }
}
