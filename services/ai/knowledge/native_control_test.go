package knowledge

import (
	"encoding/json"
	"math"
	"testing"
	"time"
)

func nativeControlFixture(t *testing.T, lambda float64) (*Brain, NativeControlConfig, time.Time) {
	t.Helper()
	b := NewBrain()
	at := time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC)
	ids := []NodeID{}
	for i := 0; i < 4; i++ {
		ids = append(ids, b.createRepresentationPrototypeAt(NewNeuralVector([]float64{float64(i) / 5, .9}), at).ID)
	}
	c := NativeControlConfig{Version: NativeControlVersion, Sources: ids, Operations: []NativeOperation{{Kind: "propagate", Gain: 1, Cost: .1}, {Kind: "stop"}}, Seed: 19, LearningRate: .2, Gamma: 1, Lambda: lambda, Temperature: .2, MaxSteps: 4, MaxActiveNodes: 100, MaxSynapseReads: 1000}
	if e := b.InitializeNativeControl(c, at); e != nil {
		t.Fatal(e)
	}
	return b, c, at
}
func TestNativeQReadoutDerivatives(t *testing.T) {
	b, c, _ := nativeControlFixture(t, 1)
	state := map[NodeID]float64{c.Sources[0]: .8, c.Sources[1]: .3}
	s := b.NativeControlSnapshot()
	eps := 1e-6
	for _, from := range append(s.Config.Sources, s.Bias) {
		for action, group := range s.Outputs {
			for _, target := range group {
				syn := b.Registry.GetByID(from).Synapses[target][0]
				original := syn.Weight
				syn.Weight = original + eps
				plus, _ := b.NativePolicy(state, true, 0)
				syn.Weight = original - eps
				minus, _ := b.NativePolicy(state, true, 0)
				syn.Weight = original
				want := state[from]
				if from == s.Bias {
					want = 1
				}
				want *= nativeReadoutScale / 4
				if syn.Inhibitory {
					want = -want
				}
				got := (plus.Values[action] - minus.Values[action]) / (2 * eps)
				if math.Abs(got-want) > 1e-9 {
					t.Fatal("readout derivative", got, want)
				}
			}
		}
	}
}
func TestNativeLambdaDelayedRewardAndProtection(t *testing.T) {
	for _, reward := range []float64{1, -1} {
		for _, lambda := range []float64{0, 1} {
			b, c, at := nativeControlFixture(t, lambda)
			_ = b.BeginNativeEpisode("e", true, at)
			states := []map[NodeID]float64{{c.Sources[0]: 1}, {c.Sources[1]: 1}}
			s := b.NativeControlSnapshot()
			protected := b.Registry.GetByID(c.Sources[3]).Synapses[s.Outputs[0][0]][0].Dynamic
			var reads []NativeReadout
			for i, x := range states {
				p, _ := b.NativePolicy(x, false, .01)
				reads = append(reads, p)
				d := NativeDecision{State: x, Values: p.Values, Probabilities: p.Probabilities, Action: i, Cost: c.Operations[i].Cost, Before: NativeStateDigest(x), After: NativeStateDigest(x), At: at}
				if e := b.AppendNativeDecision(d); e != nil {
					t.Fatal(e)
				}
			}
			syn := b.Registry.GetByID(c.Sources[0]).Synapses[s.Outputs[0][0]][0]
			old := nativeWeight(syn)
			target := -.1 + ((1-lambda)*reads[1].Values[1] + lambda*reward)
			want := old + c.LearningRate*(target-reads[0].Values[0])/2
			result, e := b.LearnNativeOutcome(reward, true, at)
			if e != nil || result.UpdatedSynapses == 0 || math.Abs(nativeWeight(syn)-want) > 1e-13 {
				t.Fatal("lambda-return update", nativeWeight(syn), want, e)
			}
			if b.Registry.GetByID(c.Sources[3]).Synapses[s.Outputs[0][0]][0].Dynamic != protected {
				t.Fatal("unrelated synapse changed")
			}
		}
	}
}
func TestNativeCanonicalValidationRejectsTampering(t *testing.T) {
	b, _, _ := nativeControlFixture(t, 1)
	raw, e := b.CanonicalJSON()
	if e != nil {
		t.Fatal(e)
	}
	if e = ValidateNativeControlState(raw, true); e != nil {
		t.Fatal(e)
	}
	for _, fault := range []string{"schema", "source", "output", "weight", "foreign"} {
		var x map[string]any
		_ = json.Unmarshal(raw, &x)
		state := x["native_control"].(map[string]any)
		switch fault {
		case "schema":
			x["schema_version"] = 6
		case "source":
			state["config"].(map[string]any)["sources"] = []int{999}
		case "output":
			state["outputs"] = [][]int{{1, 2, 3, 4}, {1, 2, 3, 4}}
		case "weight":
			x["synapses"].([]any)[0].(map[string]any)["weight"] = 2
		case "foreign":
			x["cognitive_model"] = map[string]any{"config": nil}
		}
		data, _ := json.Marshal(x)
		if ValidateNativeControlState(data, true) == nil {
			t.Fatal("accepted", fault)
		}
	}
}
