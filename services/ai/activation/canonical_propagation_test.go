package activation

import (
	"bytes"
	"reflect"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func canonicalOpponentEngine(t *testing.T, reverse bool) (*Engine, map[knowledge.NodeID]float64) {
	t.Helper()
	at := time.Unix(100, 0).UTC()
	b := knowledge.NewBrain()
	p, err := b.ProjectVectorPopulationAt(knowledge.NewNeuralVector([]float64{.2, .8}), 1, 4, at)
	if err != nil {
		t.Fatal(err)
	}
	a, target := b.Registry.GetByID(p.Units[0].NodeID), b.Registry.GetByID(p.Units[1].NodeID)
	b.ConnectAt(a, target, .91, .73, true, at)
	b.ConnectAt(a, target, .34, .82, false, at)
	if reverse {
		a.Synapses[target.ID][0], a.Synapses[target.ID][1] = a.Synapses[target.ID][1], a.Synapses[target.ID][0]
	}
	target.RestingActivation = .13
	return NewEngine(b), map[knowledge.NodeID]float64{a.ID: .77, target.ID: .31}
}

func TestCanonicalPropagationOpponentsRealAndHypothetical(t *testing.T) {
	at := time.Unix(100, 0).UTC()
	a, state := canonicalOpponentEngine(t, false)
	b, _ := canonicalOpponentEngine(t, true)
	want, wantConfidence := a.advanceWithDynamics(state, state, at, 3, knowledge.CanonicalPropagationDynamics)
	got, gotConfidence := b.advanceWithDynamics(state, state, at, 3, knowledge.CanonicalPropagationDynamics)
	if !reflect.DeepEqual(want, got) || !reflect.DeepEqual(wantConfidence, gotConfidence) {
		t.Fatal("native propagation depends on opponent insertion order")
	}
	if !bytes.Equal(canonicalPropagationBrainBytes(t, a.Memory), canonicalPropagationBrainBytes(t, b.Memory)) {
		t.Fatal("native propagation changed canonical history across orders")
	}
	before := canonicalPropagationBrainBytes(t, a.Memory)
	hypothesis, confidence := a.advanceHypotheticalWithSynapseOrder(state, state, at, 3, true)
	if !reflect.DeepEqual(want, hypothesis) || !reflect.DeepEqual(wantConfidence, confidence) || !bytes.Equal(before, canonicalPropagationBrainBytes(t, a.Memory)) {
		t.Fatal("hypothetical/native arithmetic differs or hypothesis mutates the Brain")
	}
}

func TestCanonicalPropagationRoutesObservationThoughtAndInquiryForecast(t *testing.T) {
	at := time.Unix(100, 0).UTC()
	a, state := canonicalOpponentEngine(t, false)
	b, _ := canonicalOpponentEngine(t, true)
	dynamics := knowledge.CanonicalPropagationDynamics
	for _, engine := range []*Engine{a, b} {
		engine.ActivateWith(Request{DynamicsEncoding: &dynamics, NeuralTrace: &knowledge.ObservationTrace{Stimulus: [][]knowledge.NodeID{sortedNodeIDs(state)}}, Now: at, Cycles: 3})
	}
	if !reflect.DeepEqual(a.SnapshotState(), b.SnapshotState()) || !bytes.Equal(canonicalPropagationBrainBytes(t, a.Memory), canonicalPropagationBrainBytes(t, b.Memory)) {
		t.Fatal("observation or next forecast bypasses canonical ordering")
	}
	x := a.ThinkWithContextDynamicsAt(3, nil, at.Add(time.Second), dynamics)
	y := b.ThinkWithContextDynamicsAt(3, nil, at.Add(time.Second), dynamics)
	if (!reflect.DeepEqual(x.Activations, y.Activations) || !reflect.DeepEqual(x.Confidence, y.Confidence) || !reflect.DeepEqual(x.Prediction, y.Prediction) || x.Resonance != y.Resonance || x.PredictionError != y.PredictionError) || !reflect.DeepEqual(a.SnapshotState(), b.SnapshotState()) || !bytes.Equal(canonicalPropagationBrainBytes(t, a.Memory), canonicalPropagationBrainBytes(t, b.Memory)) {
		t.Fatal("internal thought bypasses canonical ordering")
	}
	before := canonicalPropagationBrainBytes(t, a.Memory)
	xp := a.PredictOutcomeFromNodesWithDynamics(sortedNodeIDs(state), at.Add(time.Second), 3, dynamics)
	yp := b.PredictOutcomeFromNodesWithDynamics(sortedNodeIDs(state), at.Add(time.Second), 3, dynamics)
	if !reflect.DeepEqual(xp, yp) || !bytes.Equal(before, canonicalPropagationBrainBytes(t, a.Memory)) {
		t.Fatal("inquiry prediction depends on order or writes state")
	}
}

func canonicalPropagationBrainBytes(t *testing.T, brain *knowledge.Brain) []byte {
	t.Helper()
	data, err := brain.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	return data
}
