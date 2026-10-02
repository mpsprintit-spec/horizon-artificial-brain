package knowledge

import "testing"

func TestEvaluateInquiryOutcomeReportsPredictionMismatch(t *testing.T) {
	candidate := InquiryCandidate{
		ActionID: "probe",
		PredictedStates: []map[NodeID]float64{
			{1: 0.9, 2: 0.1},
			{1: 0.8, 2: 0.2},
		},
		OutcomeProbabilities: []float64{0.5, 0.5},
	}
	gain, err := EvaluateInquiryOutcome(candidate, InquiryOutcome{
		State: map[NodeID]float64{1: 0.1, 2: 0.9},
	})
	if gain <= 0 || err <= 0 {
		t.Fatalf("expected mismatch signals, gain=%v error=%v", gain, err)
	}
}

func TestEvaluateInquiryOutcomeIsDeterministic(t *testing.T) {
	candidate := InquiryCandidate{
		ActionID: "probe",
		PredictedStates: []map[NodeID]float64{
			{2: 0.2, 1: 0.8},
			{1: 0.6, 2: 0.4},
		},
		OutcomeProbabilities: []float64{0.25, 0.75},
	}
	actual := InquiryOutcome{State: map[NodeID]float64{1: 0.3, 2: 0.7}}
	gainA, errA := EvaluateInquiryOutcome(candidate, actual)
	gainB, errB := EvaluateInquiryOutcome(candidate, actual)
	if gainA != gainB || errA != errB {
		t.Fatalf("non-deterministic outcome evaluation: (%v,%v) != (%v,%v)", gainA, errA, gainB, errB)
	}
}
