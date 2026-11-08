package runtime

import (
	"bytes"
	"fmt"
	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
	"path/filepath"
	"testing"
	"time"
)

func TestNativeContrastPendingRestartAndJournalReplay(t *testing.T) {
	rt, ids := nativeFixture(t)
	wal, err := OpenEventLog(filepath.Join(t.TempDir(), "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer wal.Close()
	rt.SetEventLog(wal)
	cfg := nativeControlConfig(ids)
	cfg.Version = k.NativeControlRelationalVersion
	cfg.Encoder = "local-relational-v1"
	nativeApply(t, rt, NativeControlCommand{ID: "init", Initialize: &cfg})
	trial := func(run *BrainRuntime, id string, v float64) NativeControlCommand {
		nativeApply(t, run, NativeControlCommand{ID: id, Begin: &NativeBegin{EpisodeID: id, Training: true, Intervention: true, Observation: &NativeObservation{Source: "paired-world", SampleID: id, State: map[k.NodeID]float64{ids[0]: v, ids[6]: .8}}}})
		for i, a := range []int{0, 3} {
			action := a
			nativeApply(t, run, NativeControlCommand{ID: fmt.Sprintf("%s-%d", id, i), Step: &NativeStep{EpisodeID: id, ExperimentAction: &action}})
		}
		ep := run.brain.NativeControlSnapshot().Episode
		return NativeControlCommand{ID: id + "-out", Outcome: &NativeOutcome{EpisodeID: id, TraceDigest: k.NativeDigest(ep), Source: "paired-world", OutcomeID: id, Provenance: "observed-environment", Reward: v, Learn: true, ContrastID: "observed-pair"}}
	}
	first := trial(rt, "a", .8)
	out := nativeApply(t, rt, first)
	if !out.Learning.ContrastPending || out.Learning.UpdatedSynapses != 0 {
		t.Fatal("first trial learned without comparator")
	}
	cp := filepath.Join(t.TempDir(), "pending.json")
	if err = rt.Checkpoint(cp); err != nil {
		t.Fatal(err)
	}
	restored := NewBrainRuntime(k.NewBrain())
	if err = restored.RestoreCheckpoint(cp); err != nil {
		t.Fatal(err)
	}
	second := trial(rt, "b", .2)
	out = nativeApply(t, rt, second)
	if out.Learning.ContrastPending || out.Learning.UpdatedSynapses == 0 {
		t.Fatal("no paired update")
	}
	replay, err := RebuildRuntimeFromCheckpointAndJournal(wal, cp)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := rt.brain.CanonicalJSON()
	got, _ := replay.brain.CanonicalJSON()
	if !bytes.Equal(want, got) {
		t.Fatal("paired suffix differs")
	}
	dup := nativeApply(t, replay, second)
	if !dup.Duplicate {
		t.Fatal("duplicate observed outcome learned again")
	}
	// Same recovery, then execute the remaining real trial through the normal
	// command path: no physical adapter exists or is invoked during replay.
	secondRestored := trial(restored, "b", .2)
	nativeApply(t, restored, secondRestored)
	got, _ = restored.brain.CanonicalJSON()
	if !bytes.Equal(want, got) {
		t.Fatal("restart lost pending credit")
	}
	conflict := trial(restored, "c", .6)
	if _, err = restored.ApplyNativeControl(conflict, time.Date(2026, 11, 7, 0, 0, 0, 0, time.UTC)); err == nil {
		t.Fatal("consumed contrast reused")
	}
}

func TestNativeContrastDiscardAndV2BudgetBeforeMutation(t *testing.T) {
	rt, ids := nativeFixture(t)
	cfg := nativeControlConfig(ids)
	cfg.Version = k.NativeControlRelationalVersion
	cfg.MaxSynapseReads = 1
	nativeApply(t, rt, NativeControlCommand{ID: "init", Initialize: &cfg})
	before, _ := rt.brain.CanonicalJSON()
	seq := rt.LastSequence()
	_, err := rt.ApplyNativeControl(NativeControlCommand{ID: "begin", Begin: &NativeBegin{EpisodeID: "ep", Observation: &NativeObservation{Source: "test", SampleID: "sample", State: map[k.NodeID]float64{ids[0]: .8}}}}, time.Date(2026, 11, 7, 0, 0, 0, 0, time.UTC))
	after, _ := rt.brain.CanonicalJSON()
	if err == nil || seq != rt.LastSequence() || !bytes.Equal(before, after) {
		t.Fatal("v2 budget rejection mutated state")
	}
	// A failed pairing may be explicitly abandoned without deleting its evidence
	// or changing learned parameters. Discarding a wrong ID is rejected.
	rt, ids = nativeFixture(t)
	cfg = nativeControlConfig(ids)
	cfg.Version = k.NativeControlRelationalVersion
	nativeApply(t, rt, NativeControlCommand{ID: "init", Initialize: &cfg})
	nativeApply(t, rt, NativeControlCommand{ID: "begin", Begin: &NativeBegin{EpisodeID: "ep", Training: true, Intervention: true, Observation: &NativeObservation{Source: "test", SampleID: "sample", State: map[k.NodeID]float64{ids[0]: .8}}}})
	action := 3
	step := nativeApply(t, rt, NativeControlCommand{ID: "stop", Step: &NativeStep{EpisodeID: "ep", ExperimentAction: &action}})
	nativeApply(t, rt, NativeControlCommand{ID: "out", Outcome: &NativeOutcome{EpisodeID: "ep", TraceDigest: step.TraceDigest, Source: "test", OutcomeID: "one", Provenance: "observed-environment", Reward: .3, Learn: true, ContrastID: "pair"}})
	before, _ = rt.brain.CanonicalJSON()
	seq = rt.LastSequence()
	if _, err = rt.ApplyNativeControl(NativeControlCommand{ID: "wrong", DiscardContrastID: "wrong"}, time.Date(2026, 11, 7, 0, 0, 0, 0, time.UTC)); err == nil {
		t.Fatal("wrong pair discarded")
	}
	after, _ = rt.brain.CanonicalJSON()
	if seq != rt.LastSequence() || !bytes.Equal(before, after) {
		t.Fatal("failed discard mutated state")
	}
	updates := rt.brain.NativeControlSnapshot().Updates
	nativeApply(t, rt, NativeControlCommand{ID: "discard", DiscardContrastID: "pair"})
	if s := rt.brain.NativeControlSnapshot(); s.PendingContrast != nil || s.Updates != updates {
		t.Fatal("discard changed learning")
	}
}
