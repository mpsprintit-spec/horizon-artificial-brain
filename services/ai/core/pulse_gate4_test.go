package core

import (
	"testing"
)

func TestPulseDefaultUsesCanonicalNeuralRuntime(t *testing.T) {
	h := NewHorizonEngine()
	result, err := h.Pulse("kamu siapa")
	if err != nil {
		t.Fatalf("Pulse: %v", err)
	}
	if result.Interpretation.Source != "neural" {
		t.Fatalf("interpretation source = %q, want neural", result.Interpretation.Source)
	}
	if result.Interpretation.Source == "legacy-compatibility" {
		t.Fatal("default pulse entered legacy interpretation")
	}
}

func TestPulseDoesNotFabricateActionRecommendation(t *testing.T) {
	h := NewHorizonEngine()
	result, err := h.Pulse("iya")
	if err != nil {
		t.Fatalf("Pulse: %v", err)
	}
	if result.Recommendation != nil {
		t.Fatal("Pulse must not fabricate an action recommendation")
	}
}
