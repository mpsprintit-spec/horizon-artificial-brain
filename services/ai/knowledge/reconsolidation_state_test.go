package knowledge

import "testing"

func TestReconsolidateFromStateWeakensExistingTraceWithoutDeletingIt(t *testing.T) {
	index := NewPatternIndex()
	sequence := []PatternStep{
		{NodeID: 1, Position: 0, Activation: 1},
		{NodeID: 2, Position: 1, Activation: 1},
	}
	pattern := index.LearnTrace(sequence, nil, 2, 0.80, 0.90)
	beforeWeight := pattern.Weight
	beforeConfidence := pattern.Confidence
	beforeCount := len(index.All())

	index.ReconsolidateFromState(
		map[NodeID]float64{1: 0.8, 2: 0},
		map[NodeID]float64{1: 0.8, 2: 0.9},
		0.8,
	)

	if len(index.All()) != beforeCount {
		t.Fatalf("reconsolidation deleted or created a pattern: before=%d after=%d", beforeCount, len(index.All()))
	}
	if pattern.Weight >= beforeWeight {
		t.Fatalf("expected unexpected state to weaken trace weight: before=%v after=%v", beforeWeight, pattern.Weight)
	}
	if pattern.Confidence >= beforeConfidence {
		t.Fatalf("expected unexpected state to reduce trace confidence: before=%v after=%v", beforeConfidence, pattern.Confidence)
	}
}

func TestReconsolidateFromStateNoOpWithoutPredictionError(t *testing.T) {
	index := NewPatternIndex()
	sequence := []PatternStep{
		{NodeID: 1, Position: 0, Activation: 1},
		{NodeID: 2, Position: 1, Activation: 1},
	}
	pattern := index.LearnTrace(sequence, nil, 2, 0.80, 0.90)
	beforeWeight := pattern.Weight
	beforeConfidence := pattern.Confidence

	index.ReconsolidateFromState(
		map[NodeID]float64{1: 0.8, 2: 0},
		map[NodeID]float64{1: 0.8, 2: 0.9},
		0,
	)

	if pattern.Weight != beforeWeight || pattern.Confidence != beforeConfidence {
		t.Fatalf("zero prediction error changed trace: weight=%v/%v confidence=%v/%v",
			pattern.Weight, beforeWeight, pattern.Confidence, beforeConfidence)
	}
}
