package runtime

import (
	"testing"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestGate4DefaultsToNeuralInterpreter(t *testing.T) {
	runtime := NewBrainRuntime(knowledge.NewBrain())
	output := CognitiveOutput{
		BrainIdentity: BrainIdentity,
		Sequence:      7,
		RankedNodeIDs: []knowledge.NodeID{3, 5},
		Activations:   map[knowledge.NodeID]float64{3: 0.9, 5: 0.7},
		Confidence:    map[knowledge.NodeID]float64{3: 0.8, 5: 0.6},
	}

	got, err := runtime.Interpret(output, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Source != "neural" {
		t.Fatalf("expected neural default, got %q", got.Source)
	}
	if got.BrainIdentity != BrainIdentity || got.Sequence != 7 {
		t.Fatal("neural interpretation did not preserve runtime identity/state")
	}
}

func TestGate4LegacyIsDisabledUnlessExplicitlyInjected(t *testing.T) {
	runtime := NewBrainRuntime(knowledge.NewBrain())
	output := CognitiveOutput{BrainIdentity: BrainIdentity, Sequence: 1}

	if _, err := runtime.Interpret(output, LegacyCompatibilityInterpreter{}); err == nil {
		t.Fatal("legacy interpreter must be disabled without an explicit delegate")
	}
}

func TestGate4LegacyCannotReplaceDefaultNeuralPath(t *testing.T) {
	runtime := NewBrainRuntime(knowledge.NewBrain())
	output := CognitiveOutput{BrainIdentity: BrainIdentity, Sequence: 2}
	legacy := LegacyCompatibilityInterpreter{Delegate: func(CognitiveOutput) (Interpretation, error) {
		return Interpretation{BrainIdentity: BrainIdentity, Sequence: 2}, nil
	}}

	legacyResult, err := runtime.Interpret(output, legacy)
	if err != nil {
		t.Fatal(err)
	}
	if legacyResult.Source != "legacy-compatibility" {
		t.Fatalf("expected explicit legacy source marker, got %q", legacyResult.Source)
	}

	neuralResult, err := runtime.Interpret(output, nil)
	if err != nil {
		t.Fatal(err)
	}
	if neuralResult.Source != "neural" {
		t.Fatalf("explicit legacy adapter changed default path: %q", neuralResult.Source)
	}
}
