package core

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/perception"
	runtime "github.com/project-horizon/horizon-core/services/ai/runtime"
)

func TestPulseUsesConfiguredPerceptionLayer(t *testing.T) {
	h := NewHorizonEngine()
	h.Perception = fixedPerception{signal: perception.PerceptionSignal{
		Kind: perception.PerceptionUserInput,
		Source: "synthetic-sensor",
		RawText: "air",
		Tokens: []string{"air"},
		Confidence: 0.9,
		ObservedAt: time.Unix(100, 0).UTC(),
	}}

	result, err := h.Pulse("input-yang-harus-diabaikan")
	if err != nil {
		t.Fatalf("Pulse: %v", err)
	}
	if result.State.BrainIdentity != runtime.BrainIdentity {
		t.Fatalf("brain identity = %q, want %q", result.State.BrainIdentity, runtime.BrainIdentity)
	}
	if result.Observation.Tokens[0] != "air" {
		t.Fatalf("expected perception-produced token to reach canonical pipeline, got %v", result.Observation.Tokens)
	}
	if !result.State.Timestamp.Equal(time.Unix(100, 0).UTC()) {
		t.Fatalf("timestamp = %v, want event timestamp", result.State.Timestamp)
	}
	if result.Recommendation != nil {
		t.Fatal("Pulse must not bypass the explicit action-safety boundary")
	}
}

func TestPulseUsesCanonicalNeuralRuntimeByDefault(t *testing.T) {
	h := NewHorizonEngine()
	result, err := h.Pulse("bagaimana air?")
	if err != nil {
		t.Fatalf("Pulse: %v", err)
	}
	if result.Interpretation.Source != "neural" {
		t.Fatalf("interpretation source = %q, want neural", result.Interpretation.Source)
	}
	if result.State.Sequence != 1 {
		t.Fatalf("sequence = %d, want 1", result.State.Sequence)
	}
	if len(result.Interpretation.RankedNodeIDs) == 0 {
		t.Fatal("expected neural interpretation to contain ranked nodes")
	}
}

func TestPulseDoesNotRequireLegacyThinkingEngine(t *testing.T) {
	h := NewHorizonEngine()
	h.Thinking = nil
	result, err := h.Pulse("uji jalur neural")
	if err != nil {
		t.Fatalf("Pulse should not depend on legacy ThinkingEngine: %v", err)
	}
	if result.Interpretation.Source != "neural" {
		t.Fatalf("interpretation source = %q, want neural", result.Interpretation.Source)
	}
}
