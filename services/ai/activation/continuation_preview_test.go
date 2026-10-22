package activation

import (
	"bytes"
	"reflect"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestContinuationHypothesisOrdersOpponentsWithoutChangingLegacyState(t *testing.T) {
	at := time.Unix(100, 0).UTC()
	brain := knowledge.NewBrain()
	p, err := brain.ProjectVectorPopulationAt(knowledge.NewNeuralVector([]float64{.2, .8}), 1, 4, at)
	if err != nil {
		t.Fatal(err)
	}
	a, b := brain.Registry.GetByID(p.Units[0].NodeID), brain.Registry.GetByID(p.Units[1].NodeID)
	brain.ConnectAt(a, b, .91, .73, true, at)
	brain.ConnectAt(a, b, .34, .82, false, at)
	b.RestingActivation = .13
	e := NewEngine(brain)
	state := map[knowledge.NodeID]float64{a.ID: .77, b.ID: .31}
	before, err := brain.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	want, wantConfidence := e.advanceHypotheticalWithSynapseOrder(state, state, at, 3, true)
	a.Synapses[b.ID][0], a.Synapses[b.ID][1] = a.Synapses[b.ID][1], a.Synapses[b.ID][0]
	got, gotConfidence := e.advanceHypotheticalWithSynapseOrder(state, state, at, 3, true)
	after, err := brain.CanonicalJSON()
	if err != nil || !reflect.DeepEqual(want, got) || !reflect.DeepEqual(wantConfidence, gotConfidence) || !bytes.Equal(before, after) {
		t.Fatal("new hypothetical projection depends on opponent list order or mutates canonical state")
	}
}
