package runtime

import (
	"testing"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestCognitiveInterpretationBoundsUncertaintyAndConfidence(t *testing.T) {
	result := ToCognitiveInterpretation(Observation{Source: "sensor", Modality: "vision"}, Interpretation{
		BrainIdentity: BrainIdentity,
		RankedNodeIDs: []knowledge.NodeID{1},
		Confidence: map[knowledge.NodeID]float64{1: 4.0},
		Resonance: 2.0,
		PredictionError: -3.0,
	})

	if result.Answer.Confidence != 1 {
		t.Fatalf("confidence must be bounded to [0,1], got %v", result.Answer.Confidence)
	}
	if result.Answer.Uncertainty.Level != 0 {
		t.Fatalf("uncertainty level must be bounded to [0,1], got %v", result.Answer.Uncertainty.Level)
	}
	if result.Answer.Uncertainty.PredictionError != 0 {
		t.Fatalf("prediction error must be bounded to [0,1], got %v", result.Answer.Uncertainty.PredictionError)
	}
	if result.Answer.Uncertainty.Reason != "within-neural-range" {
		t.Fatalf("unexpected uncertainty reason: %q", result.Answer.Uncertainty.Reason)
	}
}

func TestCognitiveInterpretationReportsHighPredictionErrorAndLowResonance(t *testing.T) {
	result := ToCognitiveInterpretation(Observation{}, Interpretation{
		BrainIdentity: BrainIdentity,
		Resonance: 0.2,
		PredictionError: 0.9,
	})

	if result.Answer.Uncertainty.Level != 0.8 {
		t.Fatalf("uncertainty level = %v, want 0.8", result.Answer.Uncertainty.Level)
	}
	if result.Answer.Uncertainty.Reason != "high-prediction-error-and-low-resonance" {
		t.Fatalf("unexpected uncertainty reason: %q", result.Answer.Uncertainty.Reason)
	}
}

func TestUncertaintyDoesNotCreateRecommendation(t *testing.T) {
	result := ToCognitiveInterpretation(Observation{}, Interpretation{
		BrainIdentity: BrainIdentity,
		Resonance: 0,
		PredictionError: 1,
	})

	if result.Recommendation != nil {
		t.Fatal("uncertainty must not manufacture an action recommendation")
	}
}
