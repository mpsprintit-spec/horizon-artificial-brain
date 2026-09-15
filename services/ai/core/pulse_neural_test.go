package core

import (
	"context"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/perception"
)

type fixedPerception struct{}

func (fixedPerception) Perceive(string) ([]perception.PerceptionSignal, error) {
	return []perception.PerceptionSignal{{
		Kind: perception.PerceptionUserInput,
		Source: "synthetic-sensor",
		RawText: "air",
		Tokens: []string{"air"},
		Confidence: 0.9,
		ObservedAt: time.Unix(100, 0).UTC(),
	}}, nil
}

func TestPulseUsesConfiguredPerceptionLayer(t *testing.T) {
	previous := LegacyCognitionEnabled
	LegacyCognitionEnabled = false
	defer func() { LegacyCognitionEnabled = previous }()

	h := NewHorizonEngine()
	h.Perception = fixedPerception{}
	h.Knowledge.Store("air")

	result := h.Pulse(context.Background(), TaskPulse{Stimulus: "input-yang-harus-diabaikan"})
	if result.Path != "neural_runtime" { t.Fatalf("expected neural runtime path, got %q", result.Path) }
	if !result.Success { t.Fatal("expected neural pulse to succeed") }
	found := false
	for _, concept := range result.Concepts {
		if concept == "air" { found = true; break }
	}
	if !found { t.Fatalf("expected perception-produced token to reach neural interpretation, concepts=%v", result.Concepts) }
}

func TestPulseUsesNeuralRuntimeByDefault(t *testing.T) {
	previous := LegacyCognitionEnabled
	LegacyCognitionEnabled = false
	defer func() { LegacyCognitionEnabled = previous }()

	h := NewHorizonEngine()
	h.Knowledge.Store("air")
	result := h.Pulse(context.Background(), TaskPulse{Stimulus: "bagaimana air?"})
	if result.Path != "neural_runtime" { t.Fatalf("expected neural runtime path, got %q", result.Path) }
	if !result.Success { t.Fatal("expected neural runtime pulse to succeed") }
	if !result.Learned { t.Fatal("expected default neural pulse to record the interaction as experience") }
	if h.Runtime.LastSequence() != 2 { t.Fatalf("expected process plus learning transition, got %d", h.Runtime.LastSequence()) }
}

func TestNeuralPulseDoesNotRequireLegacyThinkingEngine(t *testing.T) {
	previous := LegacyCognitionEnabled
	LegacyCognitionEnabled = false
	defer func() { LegacyCognitionEnabled = previous }()

	h := NewHorizonEngine()
	h.Knowledge.Store("air")
	h.Thinking = nil
	result := h.Pulse(context.Background(), TaskPulse{Stimulus: "bagaimana air?"})
	if result.Path != "neural_runtime" { t.Fatalf("expected neural runtime path without legacy thinking, got %q", result.Path) }
	if !result.Success { t.Fatal("expected neural runtime to remain operational without legacy thinking") }
}

func TestPulseLegacyPathRequiresExplicitOptIn(t *testing.T) {
	previous := LegacyCognitionEnabled
	LegacyCognitionEnabled = true
	defer func() { LegacyCognitionEnabled = previous }()

	h := NewHorizonEngine()
	result := h.Pulse(context.Background(), TaskPulse{Stimulus: "saya ingin belajar"})
	if result.Path != "legacy_compatibility" { t.Fatalf("expected explicit legacy path, got %q", result.Path) }
}

func TestNeuralPulseCarriesContextAndDataIntoCognitiveBoundary(t *testing.T) {
	previous := LegacyCognitionEnabled
	LegacyCognitionEnabled = false
	defer func() { LegacyCognitionEnabled = previous }()

	h := NewHorizonEngine()
	h.Perception = fixedPerception{}
	h.Knowledge.Store("air")
	contextNode := h.Knowledge.Store("air-ruang")
	dataNode := h.Knowledge.Store("sensor-42")

	result := h.Pulse(context.Background(), TaskPulse{
		Stimulus: "input-diabaikan",
		Context:  "air-ruang",
		Data:     "sensor-42",
	})
	if !result.Success || result.Cognitive == nil {
		t.Fatalf("expected typed cognitive result, success=%v cognitive=%v", result.Success, result.Cognitive != nil)
	}
	observation := result.Cognitive.Observation
	if len(observation.ContextTokens) != 1 || observation.ContextTokens[0] != "air-ruang" {
		t.Fatalf("context was not preserved at cognitive boundary: %#v", observation.ContextTokens)
	}
	if len(observation.DataTokens) != 1 || observation.DataTokens[0] != "sensor-42" {
		t.Fatalf("data was not preserved at cognitive boundary: %#v", observation.DataTokens)
	}
	if activation := result.Cognitive.State.Activations[contextNode.ID]; activation <= 0 {
		t.Fatalf("context node did not influence neural activation: %v", activation)
	}
	if activation := result.Cognitive.State.Activations[dataNode.ID]; activation <= 0 {
		t.Fatalf("data node did not influence neural activation: %v", activation)
	}
}
