package activation

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestThinkContinuouslyMaintainsInternalProcess(t *testing.T) {
	brain := knowledge.NewBrain()
	engine := NewEngine(brain)
	now := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)

	a, err := brain.ProjectVectorPopulation(knowledge.NewNeuralVector([]float64{-1, -1, 1, 1}), 0.90, 4)
	if err != nil {
		t.Fatal(err)
	}
	b, err := brain.ProjectVectorPopulation(knowledge.NewNeuralVector([]float64{-1, 1, -1, 1}), 0.90, 4)
	if err != nil {
		t.Fatal(err)
	}
	c, err := brain.ProjectVectorPopulation(knowledge.NewNeuralVector([]float64{1, -1, -1, 1}), 0.90, 4)
	if err != nil {
		t.Fatal(err)
	}

	if err := engine.LearnPopulationTransition(a, b, 0.8, 1, now); err != nil {
		t.Fatal(err)
	}
	if err := engine.LearnPopulationTransition(b, c, 0.8, 1, now); err != nil {
		t.Fatal(err)
	}

	if _, err := engine.ActivatePopulation(a, 1, now); err != nil {
		t.Fatal(err)
	}

	thoughts := engine.ThinkContinuously(4, 1)
	if len(thoughts) != 4 {
		t.Fatalf("expected four internally generated thought steps, got %d", len(thoughts))
	}
	for i, thought := range thoughts {
		if len(thought.Activations) == 0 {
			t.Fatalf("thought step %d produced no internal state", i)
		}
		if len(thought.Prediction.State) == 0 {
			t.Fatalf("thought step %d produced no next-state prediction", i)
		}
	}

	first := thoughts[0].Activations
	changed := false
	for i := 1; i < len(thoughts); i++ {
		if stateDifference(first, thoughts[i].Activations) > 0.0001 {
			changed = true
			break
		}
	}
	if !changed {
		t.Fatal("continuous thinking produced identical internal state at every step")
	}

	if len(c.Units) == 0 {
		t.Fatal("expected learned target population")
	}
	reachedC := false
	for _, thought := range thoughts {
		for _, unit := range c.Units {
			if thought.Activations[unit.NodeID] > 0 {
				reachedC = true
				break
			}
		}
		if reachedC {
			break
		}
	}
	if !reachedC {
		t.Fatal("continuous internal process did not propagate through the learned transition chain")
	}
}
