package runtime

import "testing"

func TestAssessConsequenceValenceIsBoundedAndDirectional(t *testing.T) {
	success := AssessConsequence(ConsequenceInput{
		Success: true, Reliability: 0.9, Reversible: true, PredictionError: 0.2,
	})
	if success.Valence <= 0 || success.Valence > 1 {
		t.Fatalf("expected bounded positive valence, got %v", success.Valence)
	}

	failure := AssessConsequence(ConsequenceInput{
		Success: false, Reliability: 0.9, Reversible: true, PredictionError: 0.2,
	})
	if failure.Valence >= 0 || failure.Valence < -1 {
		t.Fatalf("expected bounded negative valence, got %v", failure.Valence)
	}
}

func TestAssessConsequencePredictionErrorDoesNotReplaceOutcomeDirection(t *testing.T) {
	success := AssessConsequence(ConsequenceInput{
		Success: true, Reliability: 1, Cost: 1, Risk: 1, PredictionError: 1,
	})
	if success.Valence <= 0 {
		t.Fatalf("prediction error incorrectly reversed successful outcome: %v", success.Valence)
	}

	failure := AssessConsequence(ConsequenceInput{
		Success: false, Reliability: 1, Cost: 0, Risk: 0, PredictionError: 1,
	})
	if failure.Valence >= 0 {
		t.Fatalf("prediction error incorrectly reversed failed outcome: %v", failure.Valence)
	}
}


func TestAssessConsequencePreservesInformationGainIndependently(t *testing.T) {
	assessment := AssessConsequence(ConsequenceInput{
		Success: true, Reliability: 0.8, InformationGain: 0.75,
	})
	if assessment.InformationGain != 0.75 {
		t.Fatalf("information gain was not preserved: %v", assessment.InformationGain)
	}
	if assessment.Valence <= 0 {
		t.Fatalf("successful consequence should remain positive: %v", assessment.Valence)
	}
}
