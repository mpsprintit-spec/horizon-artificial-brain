package knowledge

import (
	"math"
	"testing"
	"time"
)

func TestNativeRelationalPermutationAndGrowth(t *testing.T) {
	b := NewBrain()
	at := time.Date(2026, 11, 7, 0, 0, 0, 0, time.UTC)
	var ids []NodeID
	for i := 0; i < 8; i++ {
		ids = append(ids, b.createRepresentationPrototypeAt(NewNeuralVector([]float64{math.Sin(float64(i)), math.Cos(float64(i))}), at).ID)
	}
	for i := 0; i < 4; i++ {
		from, to := ids[i*2], ids[i*2+1]
		b.Registry.GetByID(from).Synapses[to] = SynapseList{&Synapse{TargetID: to, Weight: 1, Confidence: 1}}
	}
	c := NativeControlConfig{Version: NativeControlRelationalVersion, Encoder: "local-relational-v1", Sources: ids[:2], Operations: []NativeOperation{{Kind: "propagate", Gain: 1, Cost: .1}, {Kind: "stop"}}, Seed: 3, LearningRate: .1, Gamma: 1, Lambda: 1, Temperature: .2, MaxSteps: 8, MaxActiveNodes: 100, MaxSynapseReads: 100}
	if err := b.InitializeNativeControl(c, at); err != nil {
		t.Fatal(err)
	}
	a, err := b.NativePolicy(map[NodeID]float64{ids[0]: .8}, true, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []NodeID{ids[2], ids[4], ids[6]} {
		p, err := b.NativePolicy(map[NodeID]float64{id: .8}, true, 0)
		if err != nil {
			t.Fatal(err)
		}
		if NativeDigest(a) != NativeDigest(p) {
			t.Fatal("readout depends on equivalent neuron identity")
		}
	}
	// New members after controller creation use the same shared parameters.
	n := b.createRepresentationPrototypeAt(NewNeuralVector([]float64{.41, .21, .9}), at)
	n.Synapses[ids[7]] = SynapseList{&Synapse{TargetID: ids[7], Weight: 1, Confidence: 1}}
	p, err := b.NativePolicy(map[NodeID]float64{n.ID: .8}, true, 0)
	if err != nil || NativeDigest(a) != NativeDigest(p) {
		t.Fatal("growth transfer", err)
	}
	raw, err := b.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if err = ValidateNativeControlState(raw, true); err != nil {
		t.Fatal(err)
	}
}

func TestNativeContrastCancelsActiveDistractorAndLearnsBothSigns(t *testing.T) {
	for _, reward := range []float64{.9, -.9} {
		b, c, at := nativeControlFixture(t, 1)
		b.BrainState.NativeControl.Config.Version = NativeControlRelationalVersion
		s := b.NativeControlSnapshot()
		protected := b.Registry.GetByID(c.Sources[1]).Synapses[s.Outputs[0][0]][0].Dynamic
		original := nativeWeight(b.Registry.GetByID(c.Sources[0]).Synapses[s.Outputs[0][0]][0])
		for trial, x := range []float64{1, 0} {
			id := []string{"a", "b"}[trial]
			state := map[NodeID]float64{c.Sources[0]: x, c.Sources[1]: .8}
			if err := b.BeginNativeEpisode(id, true, at, true); err != nil {
				t.Fatal(err)
			}
			for _, action := range []int{0, 1} {
				p, err := b.NativePolicy(state, false, .1)
				if err != nil {
					t.Fatal(err)
				}
				d := NativeDecision{State: state, Values: p.Values, Probabilities: p.Probabilities, Action: action, Cost: c.Operations[action].Cost, Before: NativeStateDigest(state), After: NativeStateDigest(state), At: at}
				if err = b.AppendNativeDecision(d); err != nil {
					t.Fatal(err)
				}
			}
			result, err := b.LearnNativeObservedOutcome(reward*x, true, "pair", "observed-test", id, at)
			if err != nil {
				t.Fatal(err)
			}
			if trial == 0 {
				if !result.ContrastPending || result.UpdatedSynapses != 0 {
					t.Fatal("premature credit")
				}
				raw, _ := b.CanonicalJSON()
				if err = ValidateNativeControlState(raw, true); err != nil {
					t.Fatal(err)
				}
			} else if result.ContrastPending || result.UpdatedSynapses != 8 {
				t.Fatal("unexpected paired credit", result)
			}
		}
		if b.Registry.GetByID(c.Sources[1]).Synapses[s.Outputs[0][0]][0].Dynamic != protected {
			t.Fatal("common active distractor changed")
		}
		delta := nativeWeight(b.Registry.GetByID(c.Sources[0]).Synapses[s.Outputs[0][0]][0]) - original
		if delta*reward <= 0 {
			t.Fatal("credit sign", delta, reward)
		}
	}
}

func TestNativeSnapshotTypedCloneMatchesJSONAndIsIsolated(t *testing.T) {
	b, c, at := nativeControlFixture(t, 1)
	b.BrainState.NativeControl.Config.Version = NativeControlRelationalVersion
	b.BeginNativeEpisode("clone", true, at, true)
	state := map[NodeID]float64{c.Sources[0]: .7}
	p, _ := b.NativePolicy(state, true, 0)
	b.AppendNativeDecision(NativeDecision{State: state, Values: p.Values, Probabilities: p.Probabilities, Action: 1, Before: NativeStateDigest(state), After: NativeStateDigest(state), At: at})
	if _, err := b.LearnNativeObservedOutcome(.5, true, "pair", "environment", "observation", at); err != nil {
		t.Fatal(err)
	}
	snapshot := b.NativeControlSnapshot()
	if NativeDigest(snapshot) != NativeDigest(b.BrainState.NativeControl) {
		t.Fatal("typed clone changes representation")
	}
	snapshot.PendingContrast.Episode.Decisions[0].State[c.Sources[0]] = 99
	snapshot.PendingContrast.Episode.Decisions[0].Values[0] = 99
	snapshot.Config.Sources[0] = 999
	snapshot.Outputs[0][0] = 999
	if b.BrainState.NativeControl.PendingContrast.Episode.Decisions[0].State[c.Sources[0]] != .7 || b.BrainState.NativeControl.PendingContrast.Episode.Decisions[0].Values[0] == 99 || b.BrainState.NativeControl.Config.Sources[0] == 999 || b.BrainState.NativeControl.Outputs[0][0] == 999 {
		t.Fatal("snapshot aliases canonical state")
	}
}

func TestNativeRelationalMultichannelOrdering(t *testing.T) {
	b := NewBrain()
	at := time.Date(2026, 11, 7, 0, 0, 0, 0, time.UTC)
	a := b.createRepresentationPrototypeAt(NewNeuralVector([]float64{1, 0}), at)
	z := b.createRepresentationPrototypeAt(NewNeuralVector([]float64{0, 1}), at)
	list := SynapseList{&Synapse{TargetID: z.ID, Weight: 1e-17, Confidence: 1}, &Synapse{TargetID: z.ID, Weight: .8, Confidence: 1}, &Synapse{TargetID: z.ID, Weight: .17, Confidence: 1}}
	a.Synapses[z.ID] = list
	y := b.createRepresentationPrototypeAt(NewNeuralVector([]float64{.31, .27, .82}), at)
	a.Synapses[y.ID] = SynapseList{&Synapse{TargetID: y.ID, Weight: .37, Confidence: 1}}
	z.Synapses[a.ID] = SynapseList{&Synapse{TargetID: a.ID, Weight: 1, Confidence: 1}}
	c := NativeControlConfig{Version: NativeControlRelationalVersion, Encoder: "local-relational-v1", Sources: []NodeID{a.ID, z.ID}, Operations: []NativeOperation{{Kind: "propagate", Gain: 1, Cost: .1}, {Kind: "stop"}}, Seed: 7, LearningRate: .2, Gamma: 1, Lambda: 1, Temperature: .2, MaxSteps: 8, MaxActiveNodes: 100, MaxSynapseReads: 100}
	if err := b.InitializeNativeControl(c, at); err != nil {
		t.Fatal(err)
	}
	x := map[NodeID]float64{a.ID: .8}
	before, err := b.NativePolicy(x, true, 0)
	if err != nil {
		t.Fatal(err)
	}
	list[0], list[2] = list[2], list[0]
	after, err := b.NativePolicy(x, true, 0)
	if err != nil || NativeDigest(before) != NativeDigest(after) {
		t.Fatal("channel insertion order changes readout", err)
	}
}

func TestNativeContrastMeanCalibrationPreservesCommonFeatures(t *testing.T) {
	b, c, at := nativeControlFixture(t, 1)
	s := b.BrainState.NativeControl
	s.Config.Version = NativeControlRelationalVersion
	s.Config.ContrastMean = true
	target := s.Outputs[1][0]
	bias := b.Registry.GetByID(s.Bias).Synapses[target][0]
	old := nativeWeight(bias)
	common := b.Registry.GetByID(c.Sources[1]).Synapses[target][0].Dynamic
	predictions := []float64{}
	for i, v := range []float64{.8, .2} {
		id := []string{"a", "b"}[i]
		state := map[NodeID]float64{c.Sources[0]: v, c.Sources[1]: .7}
		if err := b.BeginNativeEpisode(id, true, at, true); err != nil {
			t.Fatal(err)
		}
		p, err := b.NativePolicy(state, true, 0)
		if err != nil {
			t.Fatal(err)
		}
		predictions = append(predictions, p.Values[1])
		if err = b.AppendNativeDecision(NativeDecision{State: state, Values: p.Values, Probabilities: p.Probabilities, Action: 1, Before: NativeStateDigest(state), After: NativeStateDigest(state), At: at}); err != nil {
			t.Fatal(err)
		}
		if _, err = b.LearnNativeObservedOutcome(v, true, "pair", "environment", id, at); err != nil {
			t.Fatal(err)
		}
	}
	want := old + c.LearningRate*(.5-(predictions[0]+predictions[1])/2)
	if math.Abs(nativeWeight(bias)-want) > 1e-14 {
		t.Fatal("mean calibration derivative", nativeWeight(bias), want)
	}
	if b.Registry.GetByID(c.Sources[1]).Synapses[target][0].Dynamic != common {
		t.Fatal("calibration changed common input weight")
	}
}
