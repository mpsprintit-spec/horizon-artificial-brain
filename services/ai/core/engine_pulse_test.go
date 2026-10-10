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


type multiPerception struct {
	signals []perception.PerceptionSignal
}

func (p multiPerception) Perceive(string) ([]perception.PerceptionSignal, error) {
	return append([]perception.PerceptionSignal(nil), p.signals...), nil
}

func TestPulseSignalsPreservesEachModality(t *testing.T) {
	at := time.Unix(100, 0).UTC()
	h := NewHorizonEngine()
	signals := []perception.PerceptionSignal{
		{
			Kind:       perception.PerceptionUserInput,
			Source:     "microphone",
			Modality:   "audio",
			Tokens:     []string{"suara"},
			Confidence: 0.9,
			ObservedAt: at,
		},
		{
			Kind:       perception.PerceptionWebSearch,
			Source:     "camera",
			Modality:   "vision",
			Tokens:     []string{"objek"},
			Confidence: 0.8,
			ObservedAt: at.Add(time.Millisecond),
		},
	}
	h.Perception = multiPerception{signals: signals}

	results, err := h.PulseSignals(signals)
	if err != nil {
		t.Fatalf("PulseSignals: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("result count = %d, want 2", len(results))
	}
	if results[0].Observation.Modality != "audio" {
		t.Fatalf("first modality = %q, want audio", results[0].Observation.Modality)
	}
	if results[1].Observation.Modality != "vision" {
		t.Fatalf("second modality = %q, want vision", results[1].Observation.Modality)
	}
}
