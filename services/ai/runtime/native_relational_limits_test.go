package runtime

import (
	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
	"testing"
)

// A counterexample, not a capability test: the v2 encoder discards edge sign.
// Opposite transitions therefore have identical features and action values for
// every possible setting of its readout weights. More readout training cannot
// remove this collision; a future encoder needs the missing observable input.
func TestNativeRelationalInhibitoryCollisionDocumentsLimit(t *testing.T) {
	var values, next [2]map[k.NodeID]float64
	var readouts [2]*k.NativeReadout
	for i := 0; i < 2; i++ {
		rt, ids := nativeFixture(t)
		rt.brain.Registry.GetByID(ids[0]).Synapses[ids[1]][0].Inhibitory = i == 1
		cfg := nativeControlConfig(ids)
		cfg.Version = k.NativeControlRelationalVersion
		cfg.Encoder = "local-relational-v1"
		nativeApply(t, rt, NativeControlCommand{ID: "init", Initialize: &cfg})
		values[i] = map[k.NodeID]float64{ids[0]: .8}
		nativeApply(t, rt, NativeControlCommand{ID: "begin", Begin: &NativeBegin{EpisodeID: "probe", Observation: &NativeObservation{Source: "collision-probe", SampleID: "input", State: values[i]}}})
		action := 0
		out := nativeApply(t, rt, NativeControlCommand{ID: "step", Step: &NativeStep{EpisodeID: "probe", ActionOverride: &action}})
		readouts[i], next[i] = out.Readout, out.State
	}
	if k.NativeDigest(readouts[0]) != k.NativeDigest(readouts[1]) {
		t.Fatal("expected documented sign collision")
	}
	if k.NativeStateDigest(next[0]) == k.NativeStateDigest(next[1]) {
		t.Fatal("probe failed to create distinct real transitions")
	}
}
