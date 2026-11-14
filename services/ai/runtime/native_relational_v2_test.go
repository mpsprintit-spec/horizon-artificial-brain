package runtime

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"

	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

func TestNativeV2SignReadoutAndCanonicalCompatibility(t *testing.T) {
	var readouts [2]*k.NativeReadout
	for sign := 0; sign < 2; sign++ {
		r, ids := nativeFixture(t)
		r.brain.Registry.GetByID(ids[0]).Synapses[ids[1]][0].Inhibitory = sign == 1
		cfg := nativeControlConfig(ids)
		cfg.Version, cfg.Encoder = k.NativeControlCausalVersion, k.NativeRelationalEncoderV2
		nativeApply(t, r, NativeControlCommand{ID: "init", Initialize: &cfg})
		nativeApply(t, r, NativeControlCommand{ID: "begin", Begin: &NativeBegin{EpisodeID: "probe", Observation: &NativeObservation{Source: "sign-probe", SampleID: "input", State: map[k.NodeID]float64{ids[0]: .8}}}})
		a := 0
		o := nativeApply(t, r, NativeControlCommand{ID: "step", Step: &NativeStep{EpisodeID: "probe", ActionOverride: &a}})
		readouts[sign] = o.Readout
		path := orderedSave(t, r)
		restored := orderedFixtureClone(t, path)
		gateGCompare(t, r, restored)
		raw, err := r.brain.CanonicalJSON()
		if err != nil {
			t.Fatal(err)
		}
		if err = k.ValidateNativeControlState(raw, true); err != nil {
			t.Fatal(err)
		}
		var malformed map[string]any
		if err = json.Unmarshal(raw, &malformed); err != nil {
			t.Fatal(err)
		}
		state := malformed["native_control"].(map[string]any)
		state["features"] = state["features"].([]any)[:4]
		bad, _ := json.Marshal(malformed)
		if k.ValidateNativeControlState(bad, true) == nil {
			t.Fatal("V2 accepted V1 feature dimensions")
		}
		cfg.Encoder = "local-relational-v999"
		if cfg.Validate() == nil {
			t.Fatal("unknown encoder did not fail closed")
		}
	}
	if k.NativeDigest(readouts[0]) == k.NativeDigest(readouts[1]) {
		t.Fatal("new encoder retained Phase III sign collision")
	}
}

func TestNativeV2MixedPendingReplayAndPolicyProtection(t *testing.T) {
	r, ids := nativeFixture(t)
	foundation := gateGFoundation(t, r)
	wal, err := OpenEventLog(filepath.Join(t.TempDir(), "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer wal.Close()
	r.SetEventLog(wal)
	gateGSensor(t, r, "production")
	cfg := nativeControlConfig(ids)
	cfg.Version, cfg.Encoder = k.NativeControlCausalVersion, k.NativeRelationalEncoderV2
	nativeApply(t, r, NativeControlCommand{ID: "init", Initialize: &cfg})
	before := continuationBytes(t, r)
	for _, id := range r.brain.NativeControlSnapshot().Features {
		if _, err = r.LearnExperience(learning.Experience{NeuralSequence: [][]k.NodeID{{ids[0]}, {id}}}, gateGTime(r)); err == nil || !bytes.Equal(before, continuationBytes(t, r)) {
			t.Fatal("input taught through a reserved V2 feature", err)
		}
	}
	state := map[k.NodeID]float64{ids[0]: .8}
	nativeApply(t, r, actionTrial(t, r, state, "a", "block", []int{0, 3}, .4))
	cp := orderedSave(t, r)
	second := actionTrial(t, r, state, "b", "block", []int{3}, 0)
	nativeApply(t, r, second)
	gateGText(t, r, "text")
	gateGInquiry(t, r, "inquiry")
	for i, build := range []func() (*BrainRuntime, error){
		func() (*BrainRuntime, error) { return RebuildRuntimeFromCommittedJournal(wal, foundation) },
		func() (*BrainRuntime, error) { return RebuildRuntimeFromCheckpointAndJournal(wal, cp) },
	} {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			got, err := build()
			if err != nil {
				t.Fatal(err)
			}
			gateGCompare(t, r, got)
			if !nativeApply(t, got, second).Duplicate {
				t.Fatal("replayed outcome learned twice")
			}
		})
	}
}
