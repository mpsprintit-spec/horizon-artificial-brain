package core

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/perception"
	runtime "github.com/project-horizon/horizon-core/services/ai/runtime"
)

type fixedPerception struct {
	signal perception.PerceptionSignal
}

func (p fixedPerception) Perceive(string) ([]perception.PerceptionSignal, error) {
	return []perception.PerceptionSignal{p.signal}, nil
}

func TestPulseUsesCanonicalCognitiveOrchestrator(t *testing.T) {
	at := time.Date(2026, 9, 20, 3, 24, 0, 0, time.UTC)
	h := NewHorizonEngine()
	h.Perception = fixedPerception{signal: perception.PerceptionSignal{
		Kind:       perception.PerceptionUserInput,
		Source:     "test-user",
		RawText:    "cahaya terang",
		Tokens:     []string{"cahaya", "terang"},
		Confidence: 1,
		ObservedAt: at,
	}}

	result, err := h.Pulse("ignored-by-fixed-perception")
	if err != nil {
		t.Fatalf("Pulse: %v", err)
	}
	if result.State.BrainIdentity != runtime.BrainIdentity {
		t.Fatalf("brain identity = %q, want %q", result.State.BrainIdentity, runtime.BrainIdentity)
	}
	if result.State.Sequence != 1 {
		t.Fatalf("sequence = %d, want 1", result.State.Sequence)
	}
	if !result.State.Timestamp.Equal(at) {
		t.Fatalf("timestamp = %v, want %v", result.State.Timestamp, at)
	}
	if result.Observation.Source != "test-user" {
		t.Fatalf("observation source = %q, want test-user", result.Observation.Source)
	}
	if len(result.Observation.Tokens) != 2 || result.Observation.Tokens[0] != "cahaya" || result.Observation.Tokens[1] != "terang" {
		t.Fatalf("observation tokens = %v", result.Observation.Tokens)
	}
	if result.Recommendation != nil {
		t.Fatal("Pulse must not bypass the explicit action-safety boundary")
	}
	if len(result.Interpretation.RankedNodeIDs) == 0 {
		t.Fatal("Pulse produced no neural interpretation")
	}
}
