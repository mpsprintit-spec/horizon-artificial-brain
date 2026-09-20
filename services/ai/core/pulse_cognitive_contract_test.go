package core

import (
	"testing"

	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

func TestPulseReturnsTypedCognitiveContract(t *testing.T) {
	h := NewHorizonEngine()
	h.Knowledge.Store("air")

	result, err := h.Pulse("air")
	if err != nil {
		t.Fatalf("Pulse: %v", err)
	}
	if result.Observation.Source == "" {
		t.Fatal("expected observation source")
	}
	if result.Observation.Modality != "text" {
		t.Fatalf("unexpected observation modality %q", result.Observation.Modality)
	}
	if len(result.Observation.Tokens) != 1 || result.Observation.Tokens[0] != "air" {
		t.Fatalf("unexpected stimulus tokens: %v", result.Observation.Tokens)
	}
	if result.State.BrainIdentity != runtime.BrainIdentity {
		t.Fatalf("unexpected brain identity %q", result.State.BrainIdentity)
	}
	if result.Recommendation != nil {
		t.Fatal("neural interpretation must not fabricate an action recommendation")
	}
}

func TestPulseUsesCanonicalOrchestratorState(t *testing.T) {
	h := NewHorizonEngine()
	result, err := h.Pulse("air room")
	if err != nil {
		t.Fatalf("Pulse: %v", err)
	}
	if result.Interpretation.BrainIdentity != runtime.BrainIdentity {
		t.Fatalf("unexpected interpretation brain identity %q", result.Interpretation.BrainIdentity)
	}
	if result.Interpretation.Sequence != result.State.Sequence {
		t.Fatalf("interpretation/state sequence mismatch: %d vs %d", result.Interpretation.Sequence, result.State.Sequence)
	}
	if result.Answer.Uncertainty.Level < 0 || result.Answer.Uncertainty.Level > 1 {
		t.Fatalf("uncertainty level out of range: %v", result.Answer.Uncertainty.Level)
	}
}
