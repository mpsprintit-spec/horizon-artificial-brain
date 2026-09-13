package core

import (
	"context"
	"testing"
)

func TestPulseUsesNeuralRuntimeByDefault(t *testing.T) {
	previous := LegacyCognitionEnabled
	LegacyCognitionEnabled = false
	defer func() { LegacyCognitionEnabled = previous }()

	h := NewHorizonEngine()
	h.Knowledge.Store("air")
	result := h.Pulse(context.Background(), TaskPulse{Stimulus: "bagaimana air?"})
	if result.Path != "neural_runtime" {
		t.Fatalf("expected neural runtime path, got %q", result.Path)
	}
	if !result.Success {
		t.Fatal("expected neural runtime pulse to succeed")
	}
	if h.Runtime.LastSequence() != 1 {
		t.Fatalf("expected one runtime transition, got %d", h.Runtime.LastSequence())
	}
}

func TestNeuralPulseDoesNotRequireLegacyThinkingEngine(t *testing.T) {
	previous := LegacyCognitionEnabled
	LegacyCognitionEnabled = false
	defer func() { LegacyCognitionEnabled = previous }()

	h := NewHorizonEngine()
	h.Knowledge.Store("air")
	// Deliberately remove the legacy ThinkingEngine. The neural runtime must
	// remain independently usable when legacy cognition is disabled.
	h.Thinking = nil

	result := h.Pulse(context.Background(), TaskPulse{Stimulus: "bagaimana air?"})
	if result.Path != "neural_runtime" {
		t.Fatalf("expected neural runtime path without legacy thinking, got %q", result.Path)
	}
	if !result.Success {
		t.Fatal("expected neural runtime to remain operational without legacy thinking")
	}
}

func TestPulseLegacyPathRequiresExplicitOptIn(t *testing.T) {
	previous := LegacyCognitionEnabled
	LegacyCognitionEnabled = true
	defer func() { LegacyCognitionEnabled = previous }()

	h := NewHorizonEngine()
	result := h.Pulse(context.Background(), TaskPulse{Stimulus: "saya ingin belajar"})
	if result.Path != "legacy_compatibility" {
		t.Fatalf("expected explicit legacy path, got %q", result.Path)
	}
}
