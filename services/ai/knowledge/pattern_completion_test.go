package knowledge

import "testing"

func TestCompleteTraceReconstructsLearnedTemporalPatternFromPartialCue(t *testing.T) {
	brain := NewBrain()
	samples := [][]string{
		{"saya", "ingin", "belajar", "tentang", "air"},
		{"saya", "ingin", "belajar", "tentang", "listrik"},
		{"saya", "ingin", "belajar", "tentang", "air"},
		{"saya", "ingin", "belajar", "tentang", "listrik"},
	}
	for _, sample := range samples {
		steps := make([]PatternStep, len(sample))
		for i, token := range sample {
			node := brain.Store(token)
			steps[i] = PatternStep{NodeID: node.ID, Position: i, Activation: 1}
		}
		brain.Patterns.LearnTrace(steps, nil, 0, 1, 1)
	}

	saya, ingin, belajar := brain.Fetch("saya"), brain.Fetch("ingin"), brain.Fetch("belajar")
	cue := []PatternStep{
		{NodeID: saya.ID, Position: 0, Activation: 1},
		{NodeID: ingin.ID, Position: 1, Activation: 1},
		{NodeID: belajar.ID, Position: 2, Activation: 1},
	}
	matches := brain.Patterns.CompleteTrace(cue, nil)
	if len(matches) != 2 {
		t.Fatalf("expected two distinct learned continuations, got %d", len(matches))
	}

	seen := map[NodeID]bool{}
	for _, pattern := range matches {
		if len(pattern.Sequence) != 5 {
			t.Fatalf("expected complete learned sequence, got %d steps", len(pattern.Sequence))
		}
		seen[pattern.Sequence[4].NodeID] = true
		if pattern.Frequency != 2 {
			t.Fatalf("expected repeated experience frequency 2, got %d", pattern.Frequency)
		}
	}
	if !seen[brain.Fetch("air").ID] || !seen[brain.Fetch("listrik").ID] {
		t.Fatal("partial cue failed to reconstruct both learned branches")
	}
}

func TestCompleteTraceDoesNotInventUnseenContinuation(t *testing.T) {
	brain := NewBrain()
	sample := []string{"saya", "ingin", "belajar", "tentang", "air"}
	steps := make([]PatternStep, len(sample))
	for i, token := range sample {
		node := brain.Store(token)
		steps[i] = PatternStep{NodeID: node.ID, Position: i, Activation: 1}
	}
	brain.Patterns.LearnTrace(steps, nil, 0, 1, 1)

	saya, ingin, belajar := brain.Fetch("saya"), brain.Fetch("ingin"), brain.Fetch("belajar")
	fake := brain.Store("listrik")
	matches := brain.Patterns.CompleteTrace([]PatternStep{
		{NodeID: saya.ID, Position: 0, Activation: 1},
		{NodeID: ingin.ID, Position: 1, Activation: 1},
		{NodeID: belajar.ID, Position: 2, Activation: 1},
		{NodeID: fake.ID, Position: 3, Activation: 1},
	}, nil)
	if len(matches) != 1 {
		t.Fatalf("expected one matching learned pattern, got %d", len(matches))
	}
	if matches[0].Sequence[4].NodeID != brain.Fetch("air").ID {
		t.Fatal("completion returned a continuation that was never learned")
	}
}
