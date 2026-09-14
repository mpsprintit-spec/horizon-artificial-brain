package knowledge

import "testing"

func TestLearnTraceReusesIdenticalStructuralPattern(t *testing.T) {
	p := NewPatternIndex()
	sequence := []PatternStep{{NodeID: 1, Position: 0, Activation: 1}, {NodeID: 2, Position: 1, Activation: 1}}

	first := p.LearnTraceWithEvidence(sequence, nil, 2, 0.4, 0.3, ExperienceEvidence{ExperienceID: "e1"})
	second := p.LearnTraceWithEvidence(sequence, nil, 2, 0.8, 0.7, ExperienceEvidence{ExperienceID: "e2"})

	if first == nil || second == nil {
		t.Fatal("learning must return a pattern")
	}
	if first.ID != second.ID {
		t.Fatalf("identical experience created duplicate patterns: first=%d second=%d", first.ID, second.ID)
	}
	if second.Frequency != 2 {
		t.Fatalf("expected one pattern reinforced twice, frequency=%d", second.Frequency)
	}
	if len(second.Evidence) != 2 {
		t.Fatalf("expected two independent evidence records, got %d", len(second.Evidence))
	}
	if len(p.All()) != 1 {
		t.Fatalf("expected exactly one stored structural pattern, got %d", len(p.All()))
	}
}

func TestLearnTraceKeepsDifferentTemporalPatternDistinct(t *testing.T) {
	p := NewPatternIndex()
	a := []PatternStep{{NodeID: 1, Position: 0, Delta: 0, Activation: 1}, {NodeID: 2, Position: 1, Delta: 1, Activation: 1}}
	b := []PatternStep{{NodeID: 1, Position: 0, Delta: 0, Activation: 1}, {NodeID: 2, Position: 1, Delta: 2, Activation: 1}}

	pa := p.LearnTrace(a, nil, 2, 0.5, 0.5)
	pb := p.LearnTrace(b, nil, 2, 0.5, 0.5)
	if pa.ID == pb.ID {
		t.Fatal("different temporal structures must not collapse into one pattern")
	}
	if len(p.All()) != 2 {
		t.Fatalf("expected two distinct temporal patterns, got %d", len(p.All()))
	}
}
