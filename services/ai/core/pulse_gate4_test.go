package core

import (
	"context"
	"testing"

	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

func TestPulseDefaultUsesNeuralRuntime(t *testing.T) {
	old := LegacyCognitionEnabled
	LegacyCognitionEnabled = false
	defer func() { LegacyCognitionEnabled = old }()

	h := NewHorizonEngine()
	result := h.Pulse(context.Background(), TaskPulse{Stimulus: "kamu siapa"})

	if result.Path != "neural_runtime" {
		t.Fatalf("default pulse path = %q, want neural_runtime", result.Path)
	}
	if result.Path == "legacy_compatibility" || result.Path == "legacy_control" {
		t.Fatalf("default pulse entered legacy path: %q", result.Path)
	}
	// A neural interaction now records both the accepted process event and the
	// resulting learning transition on the same runtime sequence.
	if h.Runtime.LastSequence() != 2 {
		t.Fatalf("expected process+learning runtime sequence 2, got %d", h.Runtime.LastSequence())
	}
}

func TestPulseDefaultUsesNeuralInterpreter(t *testing.T) {
	old := LegacyCognitionEnabled
	LegacyCognitionEnabled = false
	defer func() { LegacyCognitionEnabled = old }()

	h := NewHorizonEngine()
	result := h.Pulse(context.Background(), TaskPulse{Stimulus: "uji jalur neural"})

	if !result.Success {
		t.Fatalf("neural pulse failed: path=%q", result.Path)
	}
	if result.Path != "neural_runtime" {
		t.Fatalf("pulse path = %q, want neural_runtime", result.Path)
	}
	if result.InterpretationSource != "neural" {
		t.Fatalf("interpretation source = %q, want neural", result.InterpretationSource)
	}
}

func TestPulseDoesNotUseLegacyIntentControlWhenDisabled(t *testing.T) {
	old := LegacyCognitionEnabled
	LegacyCognitionEnabled = false
	defer func() { LegacyCognitionEnabled = old }()

	h := NewHorizonEngine()
	result := h.Pulse(context.Background(), TaskPulse{Stimulus: "iya"})

	if result.Path != "neural_runtime" {
		t.Fatalf("control-like input was routed to %q", result.Path)
	}
	if result.Confidence >= 1 {
		t.Fatalf("neural path unexpectedly returned hard-coded certainty: %.3f", result.Confidence)
	}
}

func TestLegacyCompatibilityIsExplicit(t *testing.T) {
	interpreter := runtime.LegacyCompatibilityInterpreter{}
	_, err := interpreter.Interpret(runtime.CognitiveOutput{BrainIdentity: runtime.BrainIdentity})
	if err == nil {
		t.Fatal("legacy interpreter without delegate must remain disabled")
	}
}
