package knowledge

import "testing"

func TestReconsolidateFromStateTargetsFailedPredictedResult(t *testing.T) {
	brain := NewBrain()
	target := brain.Store("target")
	outcome := brain.Store("outcome")
	otherCue := brain.Store("other-cue")
	otherOutcome := brain.Store("other-outcome")

	first := brain.Patterns.LearnTrace(
		[]PatternStep{
			{NodeID: target.ID, Position: 0, Activation: 1},
			{NodeID: outcome.ID, Position: 1, Activation: 1},
		},
		nil, outcome.ID, 1, 1,
	)
	second := brain.Patterns.LearnTrace(
		[]PatternStep{
			{NodeID: otherCue.ID, Position: 0, Activation: 1},
			{NodeID: otherOutcome.ID, Position: 1, Activation: 1},
		},
		nil, otherOutcome.ID, 1, 1,
	)
	beforeFirst := first.Weight
	beforeSecond := second.Weight

	brain.Patterns.ReconsolidateFromState(
		map[NodeID]float64{
			target.ID:      0.8,
			outcome.ID:     0.8,
			otherCue.ID:    0,
			otherOutcome.ID: 0,
		},
		map[NodeID]float64{
			target.ID: 0.8,
			// outcome is absent: the predicted result failed.
			otherCue.ID: 0.9,
		},
		0.8,
	)

	if first.Weight >= beforeFirst {
		t.Fatalf("failed predicted outcome was not weakened: before=%v after=%v", beforeFirst, first.Weight)
	}
	if second.Weight != beforeSecond {
		t.Fatalf("unrelated trace was weakened: before=%v after=%v", beforeSecond, second.Weight)
	}
}

func TestReconsolidateFromStateDoesNotWeakenObservedResult(t *testing.T) {
	brain := NewBrain()
	target := brain.Store("target")
	outcome := brain.Store("outcome")
	pattern := brain.Patterns.LearnTrace(
		[]PatternStep{
			{NodeID: target.ID, Position: 0, Activation: 1},
			{NodeID: outcome.ID, Position: 1, Activation: 1},
		},
		nil, outcome.ID, 1, 1,
	)
	before := pattern.Weight

	brain.Patterns.ReconsolidateFromState(
		map[NodeID]float64{target.ID: 0.8, outcome.ID: 0.8},
		map[NodeID]float64{target.ID: 0.8, outcome.ID: 0.7},
		0.8,
	)

	if pattern.Weight != before {
		t.Fatalf("observed result was incorrectly weakened: before=%v after=%v", before, pattern.Weight)
	}
}
