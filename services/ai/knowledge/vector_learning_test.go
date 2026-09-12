package knowledge

import (
	"testing"
	"time"
)

func TestLearnVectorTransitionReusesPopulationAndStrengthensTransition(t *testing.T) {
	brain := NewBrain()
	a := NewNeuralVector([]float64{0.9, 0.1, -0.4, 0.7})
	b := NewNeuralVector([]float64{-0.6, 0.8, 0.2, -0.1})
	now := time.Now().UTC()
	if err := brain.LearnVectorTransition(a, b, 0.9, 4, now); err != nil {
		t.Fatal(err)
	}
	beforeNodes := len(brain.Registry.Nodes())
	if err := brain.LearnVectorTransition(a, b, 0.9, 4, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if len(brain.Registry.Nodes()) != beforeNodes {
		t.Fatalf("repeated transition created new neural units: before=%d after=%d", beforeNodes, len(brain.Registry.Nodes()))
	}

	var reinforced int
	for _, source := range brain.Registry.Nodes() {
		for _, synapses := range source.Synapses {
			for _, synapse := range synapses {
				if synapse != nil && synapse.Kind == "" && synapse.Frequency >= 2 {
					reinforced++
				}
			}
		}
	}
	if reinforced == 0 {
		t.Fatal("expected repeated population transition to reinforce dynamic synapses")
	}
}
