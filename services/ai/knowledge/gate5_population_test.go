package knowledge

import "testing"

func TestGate5PopulationSurfaceIsDistributed(t *testing.T) {
	brain := NewKnowledgeBase()
	vector := NewNeuralVector([]float64{0.12, 0.31, 0.77, -0.18, 0.44, 0.63})

	population, err := brain.ProjectVectorPopulation(vector, 0.995, 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(population.Units) < 2 {
		t.Fatalf("distributed representation collapsed to %d unit(s)", len(population.Units))
	}

	seen := make(map[NodeID]struct{}, len(population.Units))
	for _, unit := range population.Units {
		if _, duplicate := seen[unit.NodeID]; duplicate {
			t.Fatalf("population contains duplicate unit %d", unit.NodeID)
		}
		seen[unit.NodeID] = struct{}{}
	}
}

func TestGate5NodeAblationPreservesPopulationPattern(t *testing.T) {
	brain := NewKnowledgeBase()
	vector := NewNeuralVector([]float64{0.21, 0.42, 0.64, 0.86})
	population, err := brain.ProjectVectorPopulation(vector, 0.995, 4)
	if err != nil {
		t.Fatal(err)
	}
	if len(population.Units) < 3 {
		t.Fatalf("ablation experiment requires at least 3 units, got %d", len(population.Units))
	}

	members := make([]NodeID, 0, len(population.Units))
	for _, unit := range population.Units {
		members = append(members, unit.NodeID)
	}
	pattern := brain.Patterns.Learn(members, members[0], 0.9, 0.8)
	if pattern == nil {
		t.Fatal("failed to learn distributed population pattern")
	}

	ablated := append([]NodeID(nil), members[1:]...)
	matches := brain.Patterns.MatchPopulation(ablated, 0.75)
	if len(matches) == 0 {
		t.Fatal("ablating one population unit destroyed the learned pattern")
	}
	if matches[0].Pattern.ID != pattern.ID {
		t.Fatalf("ablation matched a different pattern: got %d want %d", matches[0].Pattern.ID, pattern.ID)
	}
	if matches[0].Coverage < 0.75 {
		t.Fatalf("invalid population coverage %.3f", matches[0].Coverage)
	}
}

func TestGate5PartialCueCompletesLearnedPattern(t *testing.T) {
	patterns := NewPatternIndex()
	full := []PatternStep{
		{NodeID: 11, Position: 0, Activation: 1},
		{NodeID: 22, Position: 1, Activation: 1},
		{NodeID: 33, Position: 2, Activation: 1},
		{NodeID: 44, Position: 3, Activation: 1},
	}
	learned := patterns.LearnTrace(full, nil, 44, 0.9, 0.8)
	if learned == nil {
		t.Fatal("failed to learn full temporal pattern")
	}

	cue := full[:2]
	matches := patterns.CompleteTrace(cue, nil)
	if len(matches) == 0 {
		t.Fatal("partial cue failed to retrieve learned pattern")
	}
	if matches[0].ID != learned.ID {
		t.Fatalf("partial cue retrieved pattern %d, want %d", matches[0].ID, learned.ID)
	}
	if matches[0].Result != 44 {
		t.Fatalf("completion returned result %d, want 44", matches[0].Result)
	}
}

func TestGate5RepeatedPopulationExperienceReusesSubstrate(t *testing.T) {
	brain := NewKnowledgeBase()
	vector := NewNeuralVector([]float64{0.18, 0.36, 0.54, 0.72})

	first, err := brain.ProjectVectorPopulation(vector, 0.995, 6)
	if err != nil {
		t.Fatal(err)
	}
	initialNodes := len(brain.Registry.Nodes())
	second, err := brain.ProjectVectorPopulation(vector, 0.995, 6)
	if err != nil {
		t.Fatal(err)
	}
	if !PopulationEquivalent(first, second) {
		t.Fatal("repeated population experience did not reuse the same distributed substrate")
	}
	if got := len(brain.Registry.Nodes()); got != initialNodes {
		t.Fatalf("repeated population experience created duplicate units: before=%d after=%d", initialNodes, got)
	}
}
