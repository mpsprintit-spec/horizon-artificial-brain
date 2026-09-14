package knowledge

import "testing"

func TestCompetingPatternsCompareDifferentContinuations(t *testing.T) {
	p := NewPatternIndex()
	cue := []PatternStep{{NodeID: 1, Position: 0, Activation: 1}, {NodeID: 2, Position: 1, Activation: 1}}
	a := append(append([]PatternStep{}, cue...), PatternStep{NodeID: 3, Position: 2, Activation: 1})
	b := append(append([]PatternStep{}, cue...), PatternStep{NodeID: 4, Position: 2, Activation: 1})

	pa := p.LearnTrace(a, nil, 3, 0.9, 0.9)
	pb := p.LearnTrace(b, nil, 4, 0.4, 0.4)

	matches := p.CompetingPatterns(cue, nil)
	if len(matches) != 2 {
		t.Fatalf("expected two competing continuations, got %d", len(matches))
	}
	for _, match := range matches {
		if match.DivergenceAt != len(cue) {
			t.Fatalf("unexpected divergence point: got %d", match.DivergenceAt)
		}
		if len(match.OpponentIDs) != 1 {
			t.Fatalf("expected one opponent, got %d", len(match.OpponentIDs))
		}
		if match.CompetitiveScore <= 0 {
			t.Fatal("expected positive competitive score")
		}
	}
	if matches[0].PatternID != pa.ID || matches[1].PatternID != pb.ID {
		t.Fatalf("expected learned strength to rank A before B: %#v", matches)
	}
}

func TestCompetingPatternsRankingIsIndependentOfInsertionOrder(t *testing.T) {
	cue := []PatternStep{{NodeID: 10, Position: 0, Activation: 1}}
	a := append(append([]PatternStep{}, cue...), PatternStep{NodeID: 20, Position: 1, Activation: 1})
	b := append(append([]PatternStep{}, cue...), PatternStep{NodeID: 30, Position: 1, Activation: 1})

	first := NewPatternIndex()
	pa1 := first.LearnTrace(a, nil, 20, 0.9, 0.9)
	pb1 := first.LearnTrace(b, nil, 30, 0.4, 0.4)
	firstResult := first.CompetingPatterns(cue, nil)

	second := NewPatternIndex()
	pb2 := second.LearnTrace(b, nil, 30, 0.4, 0.4)
	pa2 := second.LearnTrace(a, nil, 20, 0.9, 0.9)
	secondResult := second.CompetingPatterns(cue, nil)

	if len(firstResult) != 2 || len(secondResult) != 2 {
		t.Fatalf("expected two competitors in both indexes: %d and %d", len(firstResult), len(secondResult))
	}
	if firstResult[0].PatternID != pa1.ID || firstResult[1].PatternID != pb1.ID {
		t.Fatalf("unexpected first ranking: %#v", firstResult)
	}
	if secondResult[0].PatternID != pa2.ID || secondResult[1].PatternID != pb2.ID {
		t.Fatalf("unexpected reversed-insertion ranking: %#v", secondResult)
	}
	if firstResult[0].CompetitiveScore != secondResult[0].CompetitiveScore ||
		firstResult[1].CompetitiveScore != secondResult[1].CompetitiveScore {
		t.Fatalf("ranking score changed with insertion order: %#v vs %#v", firstResult, secondResult)
	}
}

func TestCompetingPatternsDoNotTreatSameContinuationAsContradiction(t *testing.T) {
	p := NewPatternIndex()
	cue := []PatternStep{{NodeID: 1, Position: 0, Activation: 1}}
	traceA := append(append([]PatternStep{}, cue...), PatternStep{NodeID: 2, Position: 1, Activation: 1})
	traceB := append(append([]PatternStep{}, cue...), PatternStep{NodeID: 2, Position: 1, Activation: 1})
	p.LearnTrace(traceA, nil, 2, 0.7, 0.7)
	p.LearnTrace(traceB, nil, 2, 0.6, 0.6)

	if got := p.CompetingPatterns(cue, nil); len(got) != 0 {
		t.Fatalf("same learned continuation was incorrectly marked as competition: %#v", got)
	}
}

func TestCompetingPatternsPreserveContradictoryEvidence(t *testing.T) {
	p := NewPatternIndex()
	cue := []PatternStep{{NodeID: 1, Position: 0, Activation: 1}}
	a := append(append([]PatternStep{}, cue...), PatternStep{NodeID: 2, Position: 1, Activation: 1})
	b := append(append([]PatternStep{}, cue...), PatternStep{NodeID: 3, Position: 1, Activation: 1})
	p.LearnTraceWithEvidence(a, nil, 2, 0.8, 0.8, ExperienceEvidence{ExperienceID: "a", Reliability: 0.9, IndependenceGroup: "g1", ContradictionSet: "c1"})
	p.LearnTraceWithEvidence(b, nil, 3, 0.8, 0.8, ExperienceEvidence{ExperienceID: "b", Reliability: 0.9, IndependenceGroup: "g2", ContradictionSet: "c1"})

	got := p.CompetingPatterns(cue, nil)
	if len(got) != 2 {
		t.Fatalf("expected contradictory alternatives to remain represented, got %d", len(got))
	}
	for _, match := range got {
		if match.Contradiction <= 0 {
			t.Fatal("expected contradiction metadata to affect competition without deleting the trace")
		}
	}
}
