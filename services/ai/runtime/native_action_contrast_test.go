package runtime

import (
	"bytes"
	"fmt"
	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
	"math"
	"path/filepath"
	"testing"
	"time"
)

func actionTrial(t *testing.T, rt *BrainRuntime, state map[k.NodeID]float64, id, block string, actions []int, reward float64) NativeControlCommand {
	t.Helper()
	nativeApply(t, rt, NativeControlCommand{ID: id, Begin: &NativeBegin{EpisodeID: id, Training: true, Intervention: true, MatchKey: block, Observation: &NativeObservation{Source: "controlled-action-world", SampleID: id, State: state}}})
	var out NativeControlResult
	for i, a := range actions {
		action := a
		out = nativeApply(t, rt, NativeControlCommand{ID: fmt.Sprintf("%s-%d", id, i), Step: &NativeStep{EpisodeID: id, ExperimentAction: &action}})
	}
	return NativeControlCommand{ID: id + "-out", Outcome: &NativeOutcome{EpisodeID: id, TraceDigest: out.TraceDigest, Source: "controlled-action-world", OutcomeID: id, Provenance: "observed-environment", Reward: reward, Learn: true, ContrastID: "pair", ContrastMode: "action"}}
}

func TestNativeActionContrastCancelsRewardOffsetAndAccountsForCost(t *testing.T) {
	var reference []float64
	for _, offset := range []float64{0, .3, -.3} {
		rt, ids := nativeFixture(t)
		cfg := nativeControlConfig(ids)
		cfg.Version = k.NativeControlCausalVersion
		nativeApply(t, rt, NativeControlCommand{ID: "init", Initialize: &cfg})
		state := map[k.NodeID]float64{ids[0]: .8, ids[6]: .7}
		before, _ := rt.brain.NativePolicy(state, true, 0)
		// Delayed effect: the common second propagate delivers the first arm to
		// neuron 2. The relax arm only reaches neuron 1. Both actual traces execute.
		first := actionTrial(t, rt, state, "a", "block", []int{0, 0, 3}, offset+.4)
		out := nativeApply(t, rt, first)
		if !out.Learning.ContrastPending || out.Learning.UpdatedSynapses != 0 {
			t.Fatal("learned before factual comparator")
		}
		second := actionTrial(t, rt, state, "b", "block", []int{1, 0, 3}, offset)
		out = nativeApply(t, rt, second)
		after, _ := rt.brain.NativePolicy(state, true, 0)
		if after.Values[0]-after.Values[1] <= before.Values[0]-before.Values[1] {
			t.Fatal("positive delayed effect not learned")
		}
		if reference == nil {
			reference = after.Values
		} else {
			for i, v := range after.Values {
				if math.Abs(v-reference[i]) > 1e-14 {
					t.Fatal("common factual reward offset changed advantage update")
				}
			}
		}
		if out.Learning.UpdatedSynapses == 0 {
			t.Fatal("no canonical update")
		}
	}
	rt, ids := nativeFixture(t)
	cfg := nativeControlConfig(ids)
	cfg.Version = k.NativeControlCausalVersion
	cfg.Operations[0].Cost = .9
	nativeApply(t, rt, NativeControlCommand{ID: "init", Initialize: &cfg})
	state := map[k.NodeID]float64{ids[0]: .8}
	before, _ := rt.brain.NativePolicy(state, true, 0)
	nativeApply(t, rt, actionTrial(t, rt, state, "a", "cost-block", []int{0, 3}, .4))
	nativeApply(t, rt, actionTrial(t, rt, state, "b", "cost-block", []int{3}, 0))
	after, _ := rt.brain.NativePolicy(state, true, 0)
	if after.Values[0]-after.Values[3] >= before.Values[0]-before.Values[3] {
		t.Fatal("reward credited while ignoring compute cost")
	}
}

func TestNativeActionContrastMatchingAndPendingReplay(t *testing.T) {
	rt, ids := nativeFixture(t)
	cfg := nativeControlConfig(ids)
	cfg.Version = k.NativeControlCausalVersion
	cfg.Encoder = "local-relational-v1"
	wal, err := OpenEventLog(filepath.Join(t.TempDir(), "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer wal.Close()
	rt.SetEventLog(wal)
	nativeApply(t, rt, NativeControlCommand{ID: "init", Initialize: &cfg})
	state := map[k.NodeID]float64{ids[0]: .8, ids[6]: .4}
	nativeApply(t, rt, actionTrial(t, rt, state, "a", "block", []int{0, 3}, .4))
	cp := filepath.Join(t.TempDir(), "pending.json")
	if err = rt.Checkpoint(cp); err != nil {
		t.Fatal(err)
	}
	for _, probe := range []struct {
		name, block string
		state       map[k.NodeID]float64
		actions     []int
	}{
		{"block", "different", state, []int{3}},
		{"input", "block", map[k.NodeID]float64{ids[0]: .7, ids[6]: .4}, []int{3}},
		{"continuation", "block", state, []int{1, 0, 3}},
	} {
		t.Run(probe.name, func(t *testing.T) {
			bad := NewBrainRuntime(k.NewBrain())
			if err := bad.RestoreCheckpoint(cp); err != nil {
				t.Fatal(err)
			}
			command := actionTrial(t, bad, probe.state, "bad", probe.block, probe.actions, 0)
			before, _ := bad.brain.CanonicalJSON()
			seq := bad.LastSequence()
			_, err := bad.ApplyNativeControl(command, time.Date(2026, 11, 6, 0, 1, 0, 0, time.UTC))
			after, _ := bad.brain.CanonicalJSON()
			if err == nil || bad.LastSequence() != seq || !bytes.Equal(before, after) {
				t.Fatal("unmatched pair changed state")
			}
		})
	}
	second := actionTrial(t, rt, state, "b", "block", []int{3}, 0)
	nativeApply(t, rt, second)
	replay, err := RebuildRuntimeFromCheckpointAndJournal(wal, cp)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(continuationBytes(t, rt), continuationBytes(t, replay)) {
		t.Fatal("action contrast replay mismatch")
	}
	if !nativeApply(t, replay, second).Duplicate {
		t.Fatal("duplicate outcome not deduplicated")
	}
}

// A first stop makes the rest of an otherwise fixed continuation unreachable.
// The contrast must include the full cost of the completed alternative.
func TestNativeActionContrastDelayedContinuationVersusAbsorbingStop(t *testing.T) {
	rt, ids := nativeFixture(t)
	cfg := nativeControlConfig(ids)
	cfg.Version = k.NativeControlCausalVersion
	nativeApply(t, rt, NativeControlCommand{ID: "init", Initialize: &cfg})
	state := map[k.NodeID]float64{ids[0]: .8}
	before, _ := rt.brain.NativePolicy(state, true, 0)
	nativeApply(t, rt, actionTrial(t, rt, state, "a", "delayed", []int{0, 0, 3}, .4))
	nativeApply(t, rt, actionTrial(t, rt, state, "b", "delayed", []int{3}, 0))
	after, _ := rt.brain.NativePolicy(state, true, 0)
	if after.Values[0]-after.Values[3] <= before.Values[0]-before.Values[3] {
		t.Fatal("delayed operation advantage not learned against stop")
	}
}

// This records the remaining localization limitation, separately from learned
// effect accuracy: same-state action credit initially touches active nuisances.
func TestNativeActionContrastLocalizationLimit(t *testing.T) {
	rt, ids := nativeFixture(t)
	cfg := nativeControlConfig(ids)
	cfg.Version = k.NativeControlCausalVersion
	nativeApply(t, rt, NativeControlCommand{ID: "init", Initialize: &cfg})
	s := rt.brain.NativeControlSnapshot()
	original := map[k.NodeID][]float64{}
	for _, from := range []k.NodeID{ids[0], ids[6]} {
		for _, a := range []int{0, 3} {
			for _, to := range s.Outputs[a] {
				syn := rt.brain.Registry.GetByID(from).Synapses[to][0]
				v := syn.Weight
				if syn.Inhibitory {
					v = -v
				}
				original[from] = append(original[from], v)
			}
		}
	}
	state := map[k.NodeID]float64{ids[0]: .8, ids[6]: .8}
	nativeApply(t, rt, actionTrial(t, rt, state, "a", "localization", []int{0, 3}, .4))
	nativeApply(t, rt, actionTrial(t, rt, state, "b", "localization", []int{3}, 0))
	changed := []int{}
	for _, from := range []k.NodeID{ids[0], ids[6]} {
		count, index := 0, 0
		for _, a := range []int{0, 3} {
			for _, to := range s.Outputs[a] {
				syn := rt.brain.Registry.GetByID(from).Synapses[to][0]
				v := syn.Weight
				if syn.Inhibitory {
					v = -v
				}
				if v != original[from][index] {
					count++
				}
				index++
			}
		}
		changed = append(changed, count)
	}
	t.Logf("first action pair: relevant changed=%d, active nuisance changed=%d; exclusive non-bias precision=%g", changed[0], changed[1], float64(changed[0])/float64(changed[0]+changed[1]))
	if changed[0] == 0 || changed[1] == 0 {
		t.Fatal("localization assumptions changed; re-audit claim and gate")
	}
}
