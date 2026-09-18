package knowledge

import (
	"testing"
	"time"
)

func TestProjectVectorPopulationReusesDistributedUnits(t *testing.T) {
	brain := NewBrain()
	vector := NewNeuralVector([]float64{-0.7, 0.2, 0.8, -0.1})

	first, err := brain.ProjectVectorPopulation(vector, 0.80, 4)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Units) != 4 {
		t.Fatalf("expected sparse population of 4 units, got %d", len(first.Units))
	}

	second, err := brain.ProjectVectorPopulation(vector, 0.80, 4)
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Units) != 4 {
		t.Fatalf("expected reused population of 4 units, got %d", len(second.Units))
	}
	for _, unit := range first.Units {
		if !containsPopulationUnit(second.Units, unit.NodeID) {
			t.Fatalf("repeated vector failed to reuse node %d", unit.NodeID)
		}
	}
	if len(brain.Registry.Nodes()) != 4 {
		t.Fatalf("expected exactly 4 representation units, got %d", len(brain.Registry.Nodes()))
	}
}

func TestProjectVectorPopulationOverlapsSimilarExperience(t *testing.T) {
	brain := NewBrain()
	a := NewNeuralVector([]float64{-0.7, 0.2, 0.8, -0.1})
	b := NewNeuralVector([]float64{-0.65, 0.22, 0.76, -0.08})

	first, err := brain.ProjectVectorPopulation(a, 0.80, 4)
	if err != nil {
		t.Fatal(err)
	}
	second, err := brain.ProjectVectorPopulation(b, 0.80, 4)
	if err != nil {
		t.Fatal(err)
	}

	overlap := 0
	for _, left := range first.Units {
		if containsPopulationUnit(second.Units, left.NodeID) {
			overlap++
		}
	}
	if overlap == 0 {
		t.Fatal("similar experiences produced no overlapping neural population")
	}
}

func TestProjectVectorPopulationSeparatesOpposedExperience(t *testing.T) {
	brain := NewBrain()
	a := NewNeuralVector([]float64{-1, -1, 1, 1})
	b := NewNeuralVector([]float64{1, 1, -1, -1})

	first, err := brain.ProjectVectorPopulation(a, 0.90, 4)
	if err != nil {
		t.Fatal(err)
	}
	second, err := brain.ProjectVectorPopulation(b, 0.90, 4)
	if err != nil {
		t.Fatal(err)
	}

	overlap := 0
	for _, left := range first.Units {
		if containsPopulationUnit(second.Units, left.NodeID) {
			overlap++
		}
	}
	if overlap != 0 {
		t.Fatalf("opposed experiences unexpectedly shared %d neural units", overlap)
	}
}

func containsPopulationUnit(units []PopulationUnit, id NodeID) bool {
	for _, unit := range units {
		if unit.NodeID == id {
			return true
		}
	}
	return false
}

func TestProjectVectorPopulationAtUsesEventTime(t *testing.T) {
	brain := NewBrain()
	at := time.Unix(400, 0).UTC()
	vector := NewNeuralVector([]float64{0.1, -0.2, 0.3, 0.4})

	population, err := brain.ProjectVectorPopulationAt(vector, 0.80, 4, at)
	if err != nil {
		t.Fatal(err)
	}
	for _, unit := range population.Units {
		node := brain.Registry.GetByID(unit.NodeID)
		if node == nil {
			t.Fatalf("population references missing node %d", unit.NodeID)
		}
		if !node.LastActivation.Equal(at) {
			t.Fatalf("node %d used wall-clock time: got %v want %v", node.ID, node.LastActivation, at)
		}
	}
}
