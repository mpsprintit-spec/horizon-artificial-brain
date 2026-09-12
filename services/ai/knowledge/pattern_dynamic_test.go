package knowledge

import (
	"testing"
	"time"
)

func TestLearnTraceDistinguishesTemporalOrder(t *testing.T) {
	p := NewPatternIndex()
	result := NodeID(99)

	first := []PatternStep{
		{NodeID: 1, Delta: 0, Activation: 1},
		{NodeID: 2, Delta: 10 * time.Millisecond, Activation: 1},
	}
	second := []PatternStep{
		{NodeID: 2, Delta: 0, Activation: 1},
		{NodeID: 1, Delta: 10 * time.Millisecond, Activation: 1},
	}

	p.LearnTrace(first, nil, result, 0.8, 0.9)
	p.LearnTrace(second, nil, result, 0.8, 0.9)

	patterns := p.All()
	if len(patterns) != 2 {
		t.Fatalf("expected two temporal patterns, got %d", len(patterns))
	}
	if patterns[0].Frequency != 1 || patterns[1].Frequency != 1 {
		t.Fatalf("expected independent frequencies of 1, got %d and %d", patterns[0].Frequency, patterns[1].Frequency)
	}
}

func TestLearnTraceReinforcesSameTemporalContext(t *testing.T) {
	p := NewPatternIndex()
	result := NodeID(99)
	sequence := []PatternStep{
		{NodeID: 1, Delta: 0, Activation: 1},
		{NodeID: 2, Delta: 20 * time.Millisecond, Activation: 0.8},
	}
	context := []ContextFrame{{NodeID: 7, Activation: 0.9, Weight: 0.7}}

	pattern := p.LearnTrace(sequence, context, result, 0.6, 0.8)
	pattern = p.LearnTrace(sequence, context, result, 0.8, 0.9)

	if len(p.All()) != 1 {
		t.Fatalf("expected one reinforced pattern, got %d", len(p.All()))
	}
	if pattern.Frequency != 2 {
		t.Fatalf("expected frequency 2, got %d", pattern.Frequency)
	}
	if pattern.Weight <= 0.6 || pattern.Weight >= 0.8 {
		t.Fatalf("expected adaptive weight between observations, got %v", pattern.Weight)
	}
	if pattern.Confidence <= 0.8 {
		t.Fatalf("expected confidence reinforcement above 0.8, got %v", pattern.Confidence)
	}
}

func TestMatchTraceRanksTemporalAgreement(t *testing.T) {
	p := NewPatternIndex()
	result := NodeID(99)
	good := []PatternStep{
		{NodeID: 1, Delta: 0, Activation: 1},
		{NodeID: 2, Delta: 10 * time.Millisecond, Activation: 1},
	}
	bad := []PatternStep{
		{NodeID: 2, Delta: 0, Activation: 1},
		{NodeID: 1, Delta: 10 * time.Millisecond, Activation: 1},
	}
	p.LearnTrace(good, nil, result, 1, 1)
	p.LearnTrace(bad, nil, result, 1, 1)

	matches := p.MatchTrace(good, nil)
	if len(matches) != 2 {
		t.Fatalf("expected two matching population traces, got %d", len(matches))
	}
	if sequenceKey(matches[0].Sequence) != sequenceKey(good) {
		t.Fatalf("expected temporally matching trace first, got %v", matches[0].Sequence)
	}
}

func TestLegacyLearnStillDeduplicatesUnorderedMembers(t *testing.T) {
	p := NewPatternIndex()
	result := NodeID(99)
	p.Learn([]NodeID{1, 2, 3}, result, 0.5, 0.6)
	p.Learn([]NodeID{3, 1, 2}, result, 0.7, 0.8)

	patterns := p.All()
	if len(patterns) != 1 {
		t.Fatalf("expected legacy unordered patterns to deduplicate, got %d", len(patterns))
	}
	if patterns[0].Frequency != 2 {
		t.Fatalf("expected reinforced frequency 2, got %d", patterns[0].Frequency)
	}
	if len(patterns[0].Sequence) != 3 {
		t.Fatalf("expected compatibility sequence to be retained, got %d steps", len(patterns[0].Sequence))
	}
}
