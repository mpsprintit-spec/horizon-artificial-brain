package knowledge

import (
	"math"
	"testing"
	"time"
)

func TestAdaptPopulationRepresentationConvergesSharedStructureWithoutMerging(t *testing.T) {
	brain := NewKnowledgeBase()
	at := time.Date(2026, 10, 2, 10, 0, 0, 0, time.UTC)

	a := NewNeuralVector([]float64{1, 0, 0, 0})
	b := NewNeuralVector([]float64{0.8, 0.2, 0, 0})

	popA, err := brain.ProjectVectorPopulationAt(a, 0.75, 4, at)
	if err != nil {
		t.Fatalf("project A: %v", err)
	}
	popB, err := brain.ProjectVectorPopulationAt(b, 0.75, 4, at.Add(time.Second))
	if err != nil {
		t.Fatalf("project B: %v", err)
	}

	if PopulationEquivalent(popA, popB) {
		t.Fatal("similar experiences must retain distinct distributed populations")
	}

	shared := NodeID(0)
	for _, ua := range popA.Units {
		for _, ub := range popB.Units {
			if ua.NodeID == ub.NodeID {
				shared = ua.NodeID
			}
		}
	}
	if shared == 0 {
		t.Fatal("expected similar experiences to acquire a shared substrate unit")
	}

	node := brain.Registry.GetByID(shared)
	if node == nil {
		t.Fatalf("shared unit %d not found", shared)
	}
	initialB := NewNeuralVector(node.Representation).Similarity(b)

	for i := 0; i < 20; i++ {
		if _, err := brain.AdaptPopulationRepresentationAt(popA, a, 0.25, at.Add(time.Duration(i+2)*time.Second)); err != nil {
			t.Fatalf("adapt A[%d]: %v", i, err)
		}
		if _, err := brain.AdaptPopulationRepresentationAt(popB, b, 0.25, at.Add(time.Duration(i+2)*time.Second)); err != nil {
			t.Fatalf("adapt B[%d]: %v", i, err)
		}
	}

	finalA := NewNeuralVector(node.Representation).Similarity(a)
	finalB := NewNeuralVector(node.Representation).Similarity(b)
	if finalB <= initialB {
		t.Fatalf("shared representation did not become more compatible with the second experience: initial B=%.6f final B=%.6f", initialB, finalB)
	}
	if finalA < 0.90 {
		t.Fatalf("shared representation drifted too far from the first experience: final A=%.6f", finalA)
	}
	if math.Abs(finalA-finalB) > 0.08 {
		t.Fatalf("shared representation did not converge toward both experiences: final A/B=%.6f/%.6f", finalA, finalB)
	}

	storedA := brain.ProjectionPopulations[0]
	storedB := brain.ProjectionPopulations[1]
	if len(storedA.Prototype) != len(a.Values) || len(storedB.Prototype) != len(b.Values) {
		t.Fatal("population developmental prototypes were not persisted")
	}
	if PopulationEquivalent(storedA, storedB) {
		t.Fatal("plasticity must not collapse distinct populations into one identity")
	}
}
