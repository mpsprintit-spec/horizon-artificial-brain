package runtime

import (
	"testing"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestCognitiveStateDiffTracksNeuralStateChanges(t *testing.T) {
	const (
		addedID   knowledge.NodeID = 3
		removedID knowledge.NodeID = 1
		retained  knowledge.NodeID = 2
	)
	previous := CognitiveState{
		Sequence:       10,
		ActiveNodeIDs:  []knowledge.NodeID{removedID, retained},
		Activations:    map[knowledge.NodeID]float64{removedID: 0.4, retained: 0.2},
		Confidence:     map[knowledge.NodeID]float64{removedID: 0.5, retained: 0.6},
		Resonance:      0.3,
		PredictionError: 0.7,
	}
	current := CognitiveState{
		Sequence:       11,
		ActiveNodeIDs:  []knowledge.NodeID{retained, addedID},
		Activations:    map[knowledge.NodeID]float64{retained: 0.7, addedID: 0.8},
		Confidence:     map[knowledge.NodeID]float64{retained: 0.9, addedID: 0.4},
		Resonance:      0.8,
		PredictionError: 0.2,
	}

	delta := current.Diff(previous)
	if delta.FromSequence != 10 || delta.ToSequence != 11 {
		t.Fatalf("sequence range = %d -> %d, want 10 -> 11", delta.FromSequence, delta.ToSequence)
	}
	if len(delta.AddedNodeIDs) != 1 || delta.AddedNodeIDs[0] != addedID {
		t.Fatalf("added nodes = %v, want [%d]", delta.AddedNodeIDs, addedID)
	}
	if len(delta.RemovedNodeIDs) != 1 || delta.RemovedNodeIDs[0] != removedID {
		t.Fatalf("removed nodes = %v, want [%d]", delta.RemovedNodeIDs, removedID)
	}
	if delta.ActivationDelta[retained] != 0.5 || delta.ActivationDelta[addedID] != 0.8 || delta.ActivationDelta[removedID] != -0.4 {
		t.Fatalf("activation deltas = %v", delta.ActivationDelta)
	}
	if delta.ConfidenceDelta[retained] != 0.3 || delta.ConfidenceDelta[addedID] != 0.4 || delta.ConfidenceDelta[removedID] != -0.5 {
		t.Fatalf("confidence deltas = %v", delta.ConfidenceDelta)
	}
	if delta.ResonanceDelta != 0.5 || delta.PredictionErrorDelta != -0.5 {
		t.Fatalf("scalar deltas = resonance %v, prediction error %v", delta.ResonanceDelta, delta.PredictionErrorDelta)
	}
}
