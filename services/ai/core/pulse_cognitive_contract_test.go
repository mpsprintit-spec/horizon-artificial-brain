package core

import (
	"context"
	"testing"

	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

func TestPulseReturnsTypedCognitiveContractWithAllInputChannels(t *testing.T) {
	previous := LegacyCognitionEnabled
	LegacyCognitionEnabled = false
	defer func() { LegacyCognitionEnabled = previous }()

	h := NewHorizonEngine()
	h.Knowledge.Store("air")
	h.Knowledge.Store("room")
	h.Knowledge.Store("temperature")

	result := h.Pulse(context.Background(), TaskPulse{
		Stimulus: "air",
		Context:  "room",
		Data:     "temperature",
	})
	if !result.Success {
		t.Fatalf("expected neural pulse to succeed: %+v", result)
	}
	if result.Cognitive == nil {
		t.Fatal("expected typed cognitive result")
	}
	cognitive := result.Cognitive
	if cognitive.Observation.Source != "pulse" {
		t.Fatalf("unexpected observation source %q", cognitive.Observation.Source)
	}
	if cognitive.Observation.Modality != "user-input" {
		t.Fatalf("unexpected observation modality %q", cognitive.Observation.Modality)
	}
	if len(cognitive.Observation.Tokens) != 1 || cognitive.Observation.Tokens[0] != "air" {
		t.Fatalf("unexpected stimulus tokens: %v", cognitive.Observation.Tokens)
	}
	if len(cognitive.Observation.ContextTokens) != 1 || cognitive.Observation.ContextTokens[0] != "room" {
		t.Fatalf("unexpected context tokens: %v", cognitive.Observation.ContextTokens)
	}
	if len(cognitive.Observation.DataTokens) != 1 || cognitive.Observation.DataTokens[0] != "temperature" {
		t.Fatalf("unexpected data tokens: %v", cognitive.Observation.DataTokens)
	}
	if cognitive.State.BrainIdentity != runtime.BrainIdentity {
		t.Fatalf("unexpected brain identity %q", cognitive.State.BrainIdentity)
	}
	if cognitive.State.Sequence != 1 {
		t.Fatalf("expected cognitive state to represent process sequence 1, got %d", cognitive.State.Sequence)
	}
	if cognitive.Recommendation != nil {
		t.Fatal("neural interpretation must not fabricate an action recommendation")
	}
}

func TestPulseContextAndDataInfluenceNeuralActivation(t *testing.T) {
	previous := LegacyCognitionEnabled
	LegacyCognitionEnabled = false
	defer func() { LegacyCognitionEnabled = previous }()

	withContext := NewHorizonEngine()
	withContext.Knowledge.Store("air")
	withContext.Knowledge.Store("room")
	withContext.Knowledge.Store("temperature")
	withResult := withContext.Pulse(context.Background(), TaskPulse{
		Stimulus: "air",
		Context:  "room",
		Data:     "temperature",
	})
	if !withResult.Success || withResult.Cognitive == nil {
		t.Fatalf("context/data pulse failed: %+v", withResult)
	}

	withoutContext := NewHorizonEngine()
	withoutContext.Knowledge.Store("air")
	withoutContext.Knowledge.Store("room")
	withoutContext.Knowledge.Store("temperature")
	withoutResult := withoutContext.Pulse(context.Background(), TaskPulse{Stimulus: "air"})
	if !withoutResult.Success || withoutResult.Cognitive == nil {
		t.Fatalf("baseline pulse failed: %+v", withoutResult)
	}

	roomID := withContext.Knowledge.Registry.Get("room").ID
	temperatureID := withContext.Knowledge.Registry.Get("temperature").ID
	withRoom := withResult.Cognitive.State.Activations[roomID]
	withoutRoom := withoutResult.Cognitive.State.Activations[withoutContext.Knowledge.Registry.Get("room").ID]
	withTemperature := withResult.Cognitive.State.Activations[temperatureID]
	withoutTemperature := withoutResult.Cognitive.State.Activations[withoutContext.Knowledge.Registry.Get("temperature").ID]

	if withRoom <= withoutRoom {
		t.Fatalf("context did not increase neural activation: with=%v without=%v", withRoom, withoutRoom)
	}
	if withTemperature <= withoutTemperature {
		t.Fatalf("data did not increase neural activation: with=%v without=%v", withTemperature, withoutTemperature)
	}
}
