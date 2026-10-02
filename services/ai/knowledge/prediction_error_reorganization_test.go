package knowledge

import (
	"testing"
	"time"
)

func TestPredictionErrorDrivesDistributedRepresentationPlasticity(t *testing.T) {
	brain := NewKnowledgeBase()
	now := time.Date(2026, 10, 2, 20, 0, 0, 0, time.UTC)

	experience := NewNeuralVector([]float64{0.9, 0.1, 0, 0})
	population, err := brain.ProjectVectorPopulationAt(experience, 0.70, 4, now)
	if err != nil {
		t.Fatalf("project population: %v", err)
	}
	if len(population.Units) < 2 {
		t.Fatal("expected a distributed population with multiple units")
	}

	actualID := population.Units[0].NodeID
	otherID := population.Units[1].NodeID
	actualNode := brain.Registry.GetByID(actualID)
	otherNode := brain.Registry.GetByID(otherID)
	if actualNode == nil || otherNode == nil {
		t.Fatal("population units were not materialized")
	}

	before := append([]float64(nil), otherNode.Representation...)
	target := append([]float64(nil), actualNode.Representation...)
	predicted := map[NodeID]float64{otherID: 1}
	actual := map[NodeID]float64{actualID: 1}

	brain.ApplyPredictionErrorRepresentationPlasticity(predicted, actual, 1, 0.40, now.Add(time.Second))

	after := brain.Registry.GetByID(otherID)
	if after == nil {
		t.Fatal("experience-specific unit disappeared")
	}
	movement := 0.0
	for i := range before {
		movement += abs(before[i] - after.Representation[i])
	}
	if movement <= 0 {
		t.Fatalf("prediction error did not reorganize the distributed representation: movement=%v", movement)
	}

	targetSimilarity := NewNeuralVector(after.Representation).Similarity(NewNeuralVector(target))
	beforeSimilarity := NewNeuralVector(before).Similarity(NewNeuralVector(target))
	if targetSimilarity <= beforeSimilarity {
		t.Fatalf("representation did not move toward the observed state: before=%v after=%v", beforeSimilarity, targetSimilarity)
	}
}

func TestPredictionErrorRepresentationPlasticityIsNoOpWithoutError(t *testing.T) {
	brain := NewKnowledgeBase()
	now := time.Date(2026, 10, 2, 21, 0, 0, 0, time.UTC)

	experience := NewNeuralVector([]float64{0.7, 0.3, 0, 0})
	population, err := brain.ProjectVectorPopulationAt(experience, 0.70, 4, now)
	if err != nil {
		t.Fatalf("project population: %v", err)
	}
	unit := brain.Registry.GetByID(population.Units[1].NodeID)
	if unit == nil {
		t.Fatal("population unit missing")
	}
	before := append([]float64(nil), unit.Representation...)

	brain.ApplyPredictionErrorRepresentationPlasticity(
		map[NodeID]float64{population.Units[1].NodeID: 1},
		map[NodeID]float64{population.Units[0].NodeID: 1},
		0,
		0.40,
		now.Add(time.Second),
	)

	after := brain.Registry.GetByID(population.Units[1].NodeID)
	for i := range before {
		if before[i] != after.Representation[i] {
			t.Fatalf("zero prediction error changed representation at dimension %d: before=%v after=%v", i, before, after.Representation)
		}
	}
}
