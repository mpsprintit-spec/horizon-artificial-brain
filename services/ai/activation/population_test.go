package activation

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestPopulationDynamicsContinueWithoutExternalStimulus(t *testing.T) {
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

	seeded, err := engine.ActivatePopulation(a, 1, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(seeded.Activations) == 0 {
		t.Fatal("population activation produced no internal state")
	}

	thought := engine.ThinkWithPrediction(1)
	if len(thought.Activations) == 0 {
		t.Fatal("autonomous thought produced no internal state")
	}

	foundC := false
	for _, unit := range c.Units {
		if thought.Activations[unit.NodeID] > 0 {
			foundC = true
			break
		}
	}
	if !foundC {
		t.Fatal("recurrent dynamics did not reach the next learned population without external stimulus")
	}
}

func TestPopulationTransitionReusesDynamicSynapses(t *testing.T) {
	brain := knowledge.NewBrain()
	engine := NewEngine(brain)
	now := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)

	a, err := brain.ProjectVectorPopulation(knowledge.NewNeuralVector([]float64{-0.8, -0.2, 0.6, 0.9}), 0.80, 4)
	if err != nil {
		t.Fatal(err)
	}
	b, err := brain.ProjectVectorPopulation(knowledge.NewNeuralVector([]float64{0.7, -0.6, 0.2, -0.9}), 0.80, 4)
	if err != nil {
		t.Fatal(err)
	}

	if err := engine.LearnPopulationTransition(a, b, 0.5, 1, now); err != nil {
		t.Fatal(err)
	}
	before := countDynamicConnections(brain, a, b)
	if before == 0 {
		t.Fatal("expected dynamic recurrent connections")
	}
	freqBefore := dynamicFrequency(brain, a.Units[0].NodeID, b.Units[0].NodeID)

	if err := engine.LearnPopulationTransition(a, b, 0.5, 1, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	after := countDynamicConnections(brain, a, b)
	if after != before {
		t.Fatalf("repeated transition created new synapses: before=%d after=%d", before, after)
	}
	freqAfter := dynamicFrequency(brain, a.Units[0].NodeID, b.Units[0].NodeID)
	if freqAfter <= freqBefore {
		t.Fatalf("repeated transition did not strengthen the existing synapse: before=%d after=%d", freqBefore, freqAfter)
	}
}

func countDynamicConnections(brain *knowledge.Brain, from, to knowledge.ProjectionPopulation) int {
	count := 0
	for _, sourceUnit := range from.Units {
		source := brain.Registry.GetByID(sourceUnit.NodeID)
		for _, targetUnit := range to.Units {
			if source == nil {
				continue
			}
			for _, synapse := range source.OutboundAll() {
				if synapse.TargetID == targetUnit.NodeID && synapse.IsDynamic() {
					count++
				}
			}
		}
	}
	return count
}

func dynamicFrequency(brain *knowledge.Brain, sourceID, targetID knowledge.NodeID) int64 {
	source := brain.Registry.GetByID(sourceID)
	if source == nil {
		return 0
	}
	for _, synapse := range source.OutboundAll() {
		if synapse.TargetID == targetID && synapse.IsDynamic() {
			return synapse.Dynamic.Frequency
		}
	}
	return 0
}
