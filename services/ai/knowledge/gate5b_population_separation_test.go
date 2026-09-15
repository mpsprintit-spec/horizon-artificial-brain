package knowledge

import "testing"

func TestGate5BPopulationIdentityIgnoresRankingOrder(t *testing.T) {
	a := ProjectionPopulation{Units: []PopulationUnit{
		{NodeID: 3, Activation: 0.9},
		{NodeID: 7, Activation: 0.8},
		{NodeID: 11, Activation: 0.7},
	}}
	b := ProjectionPopulation{Units: []PopulationUnit{
		{NodeID: 11, Activation: 0.95},
		{NodeID: 3, Activation: 0.6},
		{NodeID: 7, Activation: 0.5},
	}}
	if !PopulationEquivalent(a, b) {
		t.Fatal("same population membership was treated as a different population because ranking changed")
	}
}

func TestGate5BDistinctExperiencesRemainSeparated(t *testing.T) {
	brain := NewKnowledgeBase()
	a := NewNeuralVector([]float64{0.20, 0.40, 0.60, 0.80})
	b := NewNeuralVector([]float64{0.20, 0.40, 0.60, 0.20})

	pa, err := brain.ProjectVectorPopulation(a, 0.995, 8)
	if err != nil {
		t.Fatal(err)
	}
	beforeB := len(brain.Registry.Nodes())
	pb, err := brain.ProjectVectorPopulation(b, 0.995, 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(pa.Units) == 0 || len(pb.Units) == 0 {
		t.Fatal("one of the distinct experiences produced no population")
	}
	if PopulationEquivalent(pa, pb) {
		t.Fatal("distinct experiences collapsed into the same population")
	}
	if len(brain.Registry.Nodes()) <= beforeB {
		t.Fatal("distinct experience did not create any new substrate when no compatible structure existed")
	}
}

func TestGate5BSimilarExperiencesMayShareButNotCollapse(t *testing.T) {
	brain := NewKnowledgeBase()
	a := NewNeuralVector([]float64{0.20, 0.40, 0.60, 0.80})
	b := NewNeuralVector([]float64{0.20, 0.40, 0.60, 0.79})

	pa, err := brain.ProjectVectorPopulation(a, 0.995, 8)
	if err != nil {
		t.Fatal(err)
	}
	pb, err := brain.ProjectVectorPopulation(b, 0.995, 8)
	if err != nil {
		t.Fatal(err)
	}

	shared := 0
	for _, ua := range pa.Units {
		for _, ub := range pb.Units {
			if ua.NodeID == ub.NodeID {
				shared++
			}
		}
	}
	if shared == 0 {
		t.Fatal("similar experiences failed to share compatible substrate")
	}
	if PopulationEquivalent(pa, pb) {
		t.Fatal("similar but distinct experiences collapsed into identical populations")
	}
}

func TestGate5BRepeatedDistinctSequenceDoesNotForcePopulationMerge(t *testing.T) {
	brain := NewKnowledgeBase()
	a := NewNeuralVector([]float64{0.20, 0.40, 0.60, 0.80})
	b := NewNeuralVector([]float64{0.20, 0.40, 0.60, 0.20})

	firstA, err := brain.ProjectVectorPopulation(a, 0.995, 6)
	if err != nil {
		t.Fatal(err)
	}
	firstB, err := brain.ProjectVectorPopulation(b, 0.995, 6)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		nextA, err := brain.ProjectVectorPopulation(a, 0.995, 6)
		if err != nil {
			t.Fatal(err)
		}
		nextB, err := brain.ProjectVectorPopulation(b, 0.995, 6)
		if err != nil {
			t.Fatal(err)
		}
		if !PopulationEquivalent(firstA, nextA) {
			t.Fatal("repeated A experience changed its population")
		}
		if !PopulationEquivalent(firstB, nextB) {
			t.Fatal("repeated B experience changed its population")
		}
		if PopulationEquivalent(nextA, nextB) {
			t.Fatal("repeated distinct experiences converged to one population")
		}
	}
}
