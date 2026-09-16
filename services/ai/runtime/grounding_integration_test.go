package runtime

import (
	"testing"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestGroundingIntegrationUsesSharedBrain(t *testing.T) {
	brain := knowledge.NewBrain()
	integration := NewGroundingIntegration(brain)

	vector := knowledge.NewNeuralVector([]float64{0.2, -0.1, 0.7})
	first, err := integration.GroundObservation(vector)
	if err != nil {
		t.Fatalf("first grounding failed: %v", err)
	}
	if first.NodeID == "" {
		t.Fatal("expected grounding to produce a node id")
	}

	second, err := integration.GroundObservation(vector)
	if err != nil {
		t.Fatalf("second grounding failed: %v", err)
	}
	if second.NodeID != first.NodeID {
		t.Fatalf("expected representation reuse: first=%q second=%q", first.NodeID, second.NodeID)
	}
}

func TestGroundingIntegrationRejectsEmptyVector(t *testing.T) {
	brain := knowledge.NewBrain()
	integration := NewGroundingIntegration(brain)

	if _, err := integration.GroundObservation(knowledge.NeuralVector{}); err == nil {
		t.Fatal("expected empty vector to be rejected")
	}
}
