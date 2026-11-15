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
