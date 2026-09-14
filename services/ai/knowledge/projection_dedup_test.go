package knowledge

import "testing"

func TestProjectVectorPopulationReusesSameSubstrate(t *testing.T) {
	brain := NewKnowledgeBase()
	vector := NewNeuralVector([]float64{0.15, 0.35, 0.75, -0.2})

	first, err := brain.ProjectVectorPopulation(vector, 0.995, 4)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Units) == 0 {
		t.Fatal("first projection created no population")
	}
	initialCount := len(brain.Registry.Nodes())

	second, err := brain.ProjectVectorPopulation(vector, 0.995, 4)
	if err != nil {
		t.Fatal(err)
	}
	if !PopulationEquivalent(first, second) {
		t.Fatalf("same experience recruited a different population: first=%v second=%v", first, second)
	}
	if got := len(brain.Registry.Nodes()); got != initialCount {
		t.Fatalf("repeated experience grew duplicate substrate: before=%d after=%d", initialCount, got)
	}
}

func TestProjectVectorPopulationSharesCompatibleUnits(t *testing.T) {
	brain := NewKnowledgeBase()
	base := NewNeuralVector([]float64{0.2, 0.4, 0.6, 0.8})
	near := NewNeuralVector([]float64{0.2, 0.4, 0.6, 0.79})

	first, err := brain.ProjectVectorPopulation(base, 0.99, 4)
	if err != nil {
		t.Fatal(err)
	}
	before := len(brain.Registry.Nodes())
	second, err := brain.ProjectVectorPopulation(near, 0.99, 4)
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Units) == 0 {
		t.Fatal("nearby experience recruited no population")
	}
	shared := 0
	for _, a := range first.Units {
		for _, b := range second.Units {
			if a.NodeID == b.NodeID {
				shared++
			}
		}
	}
	if shared == 0 {
		t.Fatal("compatible experiences did not share any substrate units")
	}
	if len(brain.Registry.Nodes()) < before {
		t.Fatal("substrate node count moved backwards")
	}
}
