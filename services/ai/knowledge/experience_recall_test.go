package knowledge_test

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/activation"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestExperienceTransitionRequiresLearnedConnection(t *testing.T) {
	brain := knowledge.NewBrain()

	previous := knowledge.NewNeuralVector([]float64{1, 0, 0, 0})
	current := knowledge.NewNeuralVector([]float64{0, 1, 0, 0})

	before, err := brain.ProjectVectorPopulation(previous, 0.90, 4)
	if err != nil {
		t.Fatalf("project previous experience: %v", err)
	}
	after, err := brain.ProjectVectorPopulation(current, 0.90, 4)
	if err != nil {
		t.Fatalf("project current experience: %v", err)
	}

	beforeIDs := make([]knowledge.NodeID, 0, len(before.Units))
	for _, unit := range before.Units {
		beforeIDs = append(beforeIDs, unit.NodeID)
	}

	engine := activation.NewEngine(brain)
	result := engine.ActivateWith(activation.Request{
		StimulusNodeIDs: beforeIDs,
		Cycles:          2,
		Now:             time.Unix(2, 0).UTC(),
	})

	var recalled float64
	for _, unit := range after.Units {
		if value := result.Activations[unit.NodeID]; value > recalled {
			recalled = value
		}
	}
	if recalled > 0 {
		t.Fatalf("unlearned transition unexpectedly reactivated subsequent population: %v", recalled)
	}
}

func TestExperienceTransitionCanBeReactivatedWithoutLanguageLabels(t *testing.T) {
	brain := knowledge.NewBrain()

	// Two distinct numeric experiences. No lexical token or semantic label is
	// supplied to the neural substrate.
	previous := knowledge.NewNeuralVector([]float64{1, 0, 0, 0})
	current := knowledge.NewNeuralVector([]float64{0, 1, 0, 0})

	before, err := brain.ProjectVectorPopulation(previous, 0.90, 4)
	if err != nil {
		t.Fatalf("project previous experience: %v", err)
	}
	after, err := brain.ProjectVectorPopulation(current, 0.90, 4)
	if err != nil {
		t.Fatalf("project current experience: %v", err)
	}

	if len(before.Units) == 0 || len(after.Units) == 0 {
		t.Fatal("expected both experiences to have neural populations")
	}

	beforeIDs := make([]knowledge.NodeID, 0, len(before.Units))
	for _, unit := range before.Units {
		beforeIDs = append(beforeIDs, unit.NodeID)
	}

	if err := brain.LearnVectorTransition(previous, current, 0.90, 4, time.Unix(1, 0).UTC()); err != nil {
		t.Fatalf("learn transition: %v", err)
	}

	engine := activation.NewEngine(brain)
	result := engine.ActivateWith(activation.Request{
		StimulusNodeIDs: beforeIDs,
		Cycles:          2,
		Now:             time.Unix(2, 0).UTC(),
	})

	var recalled float64
	for _, unit := range after.Units {
		if value := result.Activations[unit.NodeID]; value > recalled {
			recalled = value
		}
	}

	if recalled <= 0 {
		t.Fatalf("learned transition did not reactivate the subsequent population; activations=%v", result.Activations)
	}
}


func TestExperienceTransitionCanRecallAThroughBToC(t *testing.T) {
	brain := knowledge.NewBrain()

	a := knowledge.NewNeuralVector([]float64{1, 0, 0, 0})
	b := knowledge.NewNeuralVector([]float64{0, 1, 0, 0})
	c := knowledge.NewNeuralVector([]float64{0, 0, 1, 0})

	before, err := brain.ProjectVectorPopulation(a, 0.90, 4)
	if err != nil {
		t.Fatalf("project A: %v", err)
	}
	middle, err := brain.ProjectVectorPopulation(b, 0.90, 4)
	if err != nil {
		t.Fatalf("project B: %v", err)
	}
	after, err := brain.ProjectVectorPopulation(c, 0.90, 4)
	if err != nil {
		t.Fatalf("project C: %v", err)
	}

	beforeIDs := make([]knowledge.NodeID, 0, len(before.Units))
	for _, unit := range before.Units {
		beforeIDs = append(beforeIDs, unit.NodeID)
	}

	if err := brain.LearnVectorTransition(a, b, 0.90, 4, time.Unix(1, 0).UTC()); err != nil {
		t.Fatalf("learn A->B: %v", err)
	}
	if err := brain.LearnVectorTransition(b, c, 0.90, 4, time.Unix(2, 0).UTC()); err != nil {
		t.Fatalf("learn B->C: %v", err)
	}

	engine := activation.NewEngine(brain)
	result := engine.ActivateWith(activation.Request{
		StimulusNodeIDs: beforeIDs,
		Cycles:          3,
		Now:             time.Unix(3, 0).UTC(),
	})

	var recalledB, recalledC float64
	for _, unit := range middle.Units {
		if value := result.Activations[unit.NodeID]; value > recalledB {
			recalledB = value
		}
	}
	for _, unit := range after.Units {
		if value := result.Activations[unit.NodeID]; value > recalledC {
			recalledC = value
		}
	}

	if recalledB <= 0 {
		t.Fatalf("A did not recall B in learned chain; activations=%v", result.Activations)
	}
	if recalledC <= 0 {
		t.Fatalf("A did not propagate through B to recall C; activations=%v", result.Activations)
	}
}

func TestExperienceTransitionChainStopsWithoutSecondTransition(t *testing.T) {
	brain := knowledge.NewBrain()

	a := knowledge.NewNeuralVector([]float64{1, 0, 0, 0})
	b := knowledge.NewNeuralVector([]float64{0, 1, 0, 0})
	c := knowledge.NewNeuralVector([]float64{0, 0, 1, 0})

	before, err := brain.ProjectVectorPopulation(a, 0.90, 4)
	if err != nil {
		t.Fatalf("project A: %v", err)
	}
	middle, err := brain.ProjectVectorPopulation(b, 0.90, 4)
	if err != nil {
		t.Fatalf("project B: %v", err)
	}
	after, err := brain.ProjectVectorPopulation(c, 0.90, 4)
	if err != nil {
		t.Fatalf("project C: %v", err)
	}

	beforeIDs := make([]knowledge.NodeID, 0, len(before.Units))
	for _, unit := range before.Units {
		beforeIDs = append(beforeIDs, unit.NodeID)
	}

	if err := brain.LearnVectorTransition(a, b, 0.90, 4, time.Unix(1, 0).UTC()); err != nil {
		t.Fatalf("learn A->B: %v", err)
	}

	engine := activation.NewEngine(brain)
	result := engine.ActivateWith(activation.Request{
		StimulusNodeIDs: beforeIDs,
		Cycles:          3,
		Now:             time.Unix(3, 0).UTC(),
	})

	var recalledB, recalledC float64
	for _, unit := range middle.Units {
		if value := result.Activations[unit.NodeID]; value > recalledB {
			recalledB = value
		}
	}
	for _, unit := range after.Units {
		if value := result.Activations[unit.NodeID]; value > recalledC {
			recalledC = value
		}
	}

	if recalledB <= 0 {
		t.Fatalf("A did not recall learned B; activations=%v", result.Activations)
	}
	if recalledC > 0 {
		t.Fatalf("C was recalled without a learned B->C transition: %v; activations=%v", recalledC, result.Activations)
	}
}
