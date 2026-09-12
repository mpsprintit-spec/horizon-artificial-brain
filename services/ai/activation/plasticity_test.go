package activation

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestPredictionErrorPlasticityAdaptsExistingConnection(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	a := kb.Store("a")
	b := kb.Store("b")
	kb.Connect(a, b, 0.40, 0.50, false)

	synapse := a.Synapses[b.ID][0]
	before := synapse.Dynamic.Weight

	e := NewEngine(kb)
	e.ApplyPredictionErrorPlasticity(
		map[knowledge.NodeID]float64{a.ID: 0.2, b.ID: 0.1},
		map[knowledge.NodeID]float64{a.ID: 0.2, b.ID: 0.9},
		0.8,
		time.Now().UTC(),
	)

	if synapse.Dynamic.Weight <= before {
		t.Fatalf("expected surprising activation to strengthen existing path: before=%v after=%v", before, synapse.Dynamic.Weight)
	}
	if synapse.Dynamic.Weight > 0.95 || synapse.Dynamic.Weight < 0.05 {
		t.Fatalf("plasticity escaped bounded range: %v", synapse.Dynamic.Weight)
	}
}

func TestPredictionErrorPlasticityDoesNotCreateNewConnection(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	a := kb.Store("a")
	b := kb.Store("b")

	e := NewEngine(kb)
	e.ApplyPredictionErrorPlasticity(
		map[knowledge.NodeID]float64{a.ID: 0.2, b.ID: 0.0},
		map[knowledge.NodeID]float64{a.ID: 0.2, b.ID: 0.9},
		1.0,
		time.Now().UTC(),
	)

	if len(a.Synapses[b.ID]) != 0 {
		t.Fatal("prediction error must not invent a connection without structural growth evidence")
	}
}
