package runtime

import (
	"bytes"
	"encoding/json"
	"fmt"
	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func nativeFixture(t *testing.T) (*BrainRuntime, []k.NodeID) {
	t.Helper()
	brain := k.NewBrain()
	ids := []k.NodeID{}
	for i := 1; i <= 7; i++ {
		n, _, e := brain.Registry.GetOrCreateRepresentation(k.NewNeuralVector([]float64{float64(i) / 8, 1}), 1)
		if e != nil {
			t.Fatal(e)
		}
		ids = append(ids, n.ID)
	}
	at := time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 3; i++ {
		source := brain.Registry.GetByID(ids[i])
		target := ids[i+1]
		source.Synapses[target] = k.SynapseList{&k.Synapse{TargetID: target, Weight: 1, Confidence: 1, LastActivation: at, Dynamic: k.DynamicState{Weight: 1, Confidence: 1}}}
	}
	return NewBrainRuntime(brain), ids
}
func nativeControlConfig(ids []k.NodeID) k.NativeControlConfig {
	return k.NativeControlConfig{MaxActiveNodes: 4096, MaxSynapseReads: 100000, Version: k.NativeControlVersion, Sources: ids, Operations: []k.NativeOperation{{Kind: "propagate", Gain: 1, Cost: .08}, {Kind: "relax", Gain: .1, Cost: .08}, {Kind: "focus", Gain: .5, Cost: .08}, {Kind: "stop"}}, Seed: 37, LearningRate: .2, Gamma: 1, Lambda: 1, Temperature: .15, MaxSteps: 8}
}
func nativeApply(t *testing.T, r *BrainRuntime, c NativeControlCommand) NativeControlResult {
	t.Helper()
	o, e := r.ApplyNativeControl(c, time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC).Add(time.Duration(r.LastSequence()+1)*time.Millisecond))
	if e != nil {
		t.Fatalf("%s: %v", c.ID, e)
	}
	return o
}
func nativeEnd(t *testing.T, r *BrainRuntime, id string, reward float64, learn bool) NativeControlResult {
	t.Helper()
	s := r.brain.NativeControlSnapshot()
	return nativeApply(t, r, NativeControlCommand{ID: id, Outcome: &NativeOutcome{EpisodeID: s.Episode.ID, TraceDigest: k.NativeDigest(s.Episode), Source: "test-environment", OutcomeID: id, Provenance: "observed-environment", Reward: reward, Learn: learn}})
}
func TestNativeControlReplayAndCredit(t *testing.T) {
	r, ids := nativeFixture(t)
	path := filepath.Join(t.TempDir(), "events.jsonl")
	wal, e := OpenEventLog(path)
	if e != nil {
		t.Fatal(e)
	}
	defer wal.Close()
	r.SetEventLog(wal)
	cfg := nativeControlConfig(ids)
	nativeApply(t, r, NativeControlCommand{ID: "init", Initialize: &cfg})
	nativeApply(t, r, NativeControlCommand{ID: "begin", Begin: &NativeBegin{EpisodeID: "ep", Training: true, Observation: &NativeObservation{Source: "fixture", SampleID: "sample", State: map[k.NodeID]float64{ids[0]: 1}}}})
	cp := filepath.Join(t.TempDir(), "middle.json")
	if e = r.Checkpoint(cp); e != nil {
		t.Fatal(e)
	}
	before, _ := r.brain.CanonicalJSON()
	s := r.brain.NativeControlSnapshot()
	protected := r.brain.Registry.GetByID(ids[6]).Synapses[s.Outputs[0][0]][0].Weight
	for i := 0; i < 8; i++ {
		out := nativeApply(t, r, NativeControlCommand{ID: fmt.Sprintf("step-%d", i), Step: &NativeStep{EpisodeID: "ep", Sample: .01}})
		if out.Stopped {
			break
		}
	}
	out := nativeEnd(t, r, "outcome", .9, true)
	if out.Learning.UpdatedSynapses == 0 {
		t.Fatal("no credit")
	}
	if r.brain.Registry.GetByID(ids[6]).Synapses[s.Outputs[0][0]][0].Weight != protected {
		t.Fatal("inactive source got credit")
	}
	after, _ := r.brain.CanonicalJSON()
	if bytes.Equal(before, after) {
		t.Fatal("parameters unchanged")
	}
	recovered, e := RebuildRuntimeFromCheckpointAndJournal(wal, cp)
	if e != nil {
		t.Fatal(e)
	}
	got, _ := recovered.brain.CanonicalJSON()
	if !bytes.Equal(after, got) {
		t.Fatal("canonical replay mismatch")
	}
	if k.NativeDigest(recovered.activation.SnapshotState()) != k.NativeDigest(r.activation.SnapshotState()) {
		t.Fatal("activation replay mismatch")
	}
	final := filepath.Join(t.TempDir(), "final.json")
	if e = r.Checkpoint(final); e != nil {
		t.Fatal(e)
	}
	f := NewBrainRuntime(k.NewBrain())
	if e = f.RestoreCheckpoint(final); e != nil {
		t.Fatal(e)
	}
	// Mid-episode values, propensity, continuity and parameters are authenticated.
	blob, _ := os.ReadFile(cp)
	var snapshot BrainSnapshot
	_ = json.Unmarshal(blob, &snapshot)
	if snapshot.SchemaVersion != 20 {
		t.Fatal("native schema")
	}
	snapshot.SchemaVersion = 19
	blob, _ = json.Marshal(snapshot)
	bad := filepath.Join(t.TempDir(), "downgrade.json")
	_ = os.WriteFile(bad, blob, 0600)
	if NewBrainRuntime(k.NewBrain()).RestoreCheckpoint(bad) == nil {
		t.Fatal("downgrade accepted")
	}
}
func TestNativeControlRejectsFalseEvidenceAndConflicts(t *testing.T) {
	r, ids := nativeFixture(t)
	cfg := nativeControlConfig(ids)
	nativeApply(t, r, NativeControlCommand{ID: "init", Initialize: &cfg})
	n := r.LastSequence()
	invalid := NativeControlCommand{ID: "bad", Outcome: &NativeOutcome{EpisodeID: "ep", TraceDigest: string(bytes.Repeat([]byte("a"), 64)), Source: "internal", OutcomeID: "bad", Provenance: "hypothetical", Reward: 1, Learn: true}}
	if _, e := r.ApplyNativeControl(invalid, time.Now()); e == nil || r.LastSequence() != n {
		t.Fatal("hypothetical reward accepted")
	}
	original := NativeControlCommand{ID: "begin", Begin: &NativeBegin{EpisodeID: "ep", Training: true, Observation: &NativeObservation{Source: "fixture", SampleID: "sample", State: map[k.NodeID]float64{ids[0]: 1}}}}
	nativeApply(t, r, original)
	dup, e := r.ApplyNativeControl(original, time.Now())
	if e != nil || !dup.Duplicate {
		t.Fatal("retry not idempotent")
	}
	original.Begin.EpisodeID = "other"
	if _, e = r.ApplyNativeControl(original, time.Now()); e == nil {
		t.Fatal("conflicting ID accepted")
	}
	if _, e = r.ApplyNativeControl(NativeControlCommand{ID: "nan", Step: &NativeStep{EpisodeID: "ep", Sample: math.NaN()}}, time.Now()); e == nil {
		t.Fatal("NaN accepted")
	}
	before, _ := r.brain.CanonicalJSON()
	if _, e = r.CognitiveThink(1); e == nil {
		t.Fatal("mixed legacy mutation accepted")
	}
	after, _ := r.brain.CanonicalJSON()
	if !bytes.Equal(before, after) {
		t.Fatal("failed legacy command mutated")
	}
}

func TestNativeOutcomeSingleUseAcrossCommandsEpisodesAndRestart(t *testing.T) {
	r, ids := nativeFixture(t)
	cfg := nativeControlConfig(ids)
	nativeApply(t, r, NativeControlCommand{ID: "init", Initialize: &cfg})
	complete := func(rt *BrainRuntime, episode string) NativeControlResult {
		nativeApply(t, rt, NativeControlCommand{ID: episode, Begin: &NativeBegin{EpisodeID: episode, Observation: &NativeObservation{Source: "fixture", SampleID: episode, State: map[k.NodeID]float64{ids[0]: 1}}}})
		stop := 3
		return nativeApply(t, rt, NativeControlCommand{ID: episode + "-stop", Step: &NativeStep{EpisodeID: episode, ActionOverride: &stop}})
	}
	out := complete(r, "first")
	command := NativeControlCommand{ID: "feedback", Outcome: &NativeOutcome{EpisodeID: "first", TraceDigest: out.TraceDigest, Source: "sensor", OutcomeID: "one-observation", Provenance: "observed-sensor", Reward: .8}}
	nativeApply(t, r, command)
	cp := filepath.Join(t.TempDir(), "single-use.json")
	if err := r.Checkpoint(cp); err != nil {
		t.Fatal(err)
	}
	restored := NewBrainRuntime(k.NewBrain())
	if err := restored.RestoreCheckpoint(cp); err != nil {
		t.Fatal(err)
	}
	for _, rt := range []*BrainRuntime{r, restored} {
		retry, err := rt.ApplyNativeControl(command, time.Now())
		if err != nil || !retry.Duplicate {
			t.Fatal("exact retry must be idempotent", err)
		}
		out = complete(rt, "second")
		altered := NativeControlCommand{ID: "different-command", Outcome: &NativeOutcome{EpisodeID: "second", TraceDigest: out.TraceDigest, Source: "sensor", OutcomeID: "one-observation", Provenance: "observed-sensor", Reward: .8}}
		before := continuationBytes(t, rt)
		if _, err = rt.ApplyNativeControl(altered, time.Now()); err == nil || !bytes.Equal(before, continuationBytes(t, rt)) {
			t.Fatal("one observed outcome reused for another episode")
		}
	}
}

func TestNativeBudgetRejectionBeforeJournalAndMutation(t *testing.T) {
	r, ids := nativeFixture(t)
	cfg := nativeControlConfig(ids)
	cfg.MaxSynapseReads = 1
	nativeApply(t, r, NativeControlCommand{ID: "init", Initialize: &cfg})
	nativeApply(t, r, NativeControlCommand{ID: "begin", Begin: &NativeBegin{EpisodeID: "ep", Observation: &NativeObservation{Source: "fixture", SampleID: "sample", State: map[k.NodeID]float64{ids[0]: 1, ids[1]: 1}}}})
	force := 0
	before := continuationBytes(t, r)
	seq := r.LastSequence()
	_, err := r.ApplyNativeControl(NativeControlCommand{ID: "over-budget", Step: &NativeStep{EpisodeID: "ep", ActionOverride: &force}}, time.Now())
	if err == nil || r.LastSequence() != seq || r.RecoveryRequired() || !bytes.Equal(before, continuationBytes(t, r)) {
		t.Fatal("budget failure was not atomic", err)
	}
}

func TestNativeProcessExitAfterFsync(t *testing.T) {
	if path := os.Getenv("HORIZON_NATIVE_CRASH_WAL"); path != "" {
		wal, e := OpenEventLog(path)
		if e != nil {
			t.Fatal(e)
		}
		rt, e := RebuildRuntimeFromCheckpointAndJournal(wal, os.Getenv("HORIZON_NATIVE_CRASH_CP"))
		if e != nil {
			t.Fatal(e)
		}
		rt.SetEventLog(wal)
		wal.beforeAppend = func(ev LoggedEvent) error {
			if ev.SchemaVersion != 14 {
				t.Fatal("wrong WAL schema")
			}
			raw, e := json.Marshal(ev)
			if e != nil {
				t.Fatal(e)
			}
			if _, e = wal.file.Write(append(raw, '\n')); e != nil {
				t.Fatal(e)
			}
			if e = wal.file.Sync(); e != nil {
				t.Fatal(e)
			}
			os.Exit(76)
			return nil
		}
		nativeEnd(t, rt, "crash-outcome", .9, true)
		t.Fatal("no process exit")
	}
	rt, ids := nativeFixture(t)
	path := filepath.Join(t.TempDir(), "events.jsonl")
	wal, e := OpenEventLog(path)
	if e != nil {
		t.Fatal(e)
	}
	defer wal.Close()
	rt.SetEventLog(wal)
	cfg := nativeControlConfig(ids)
	nativeApply(t, rt, NativeControlCommand{ID: "init", Initialize: &cfg})
	nativeApply(t, rt, NativeControlCommand{ID: "begin", Begin: &NativeBegin{EpisodeID: "ep", Training: true, Observation: &NativeObservation{Source: "fixture", SampleID: "sample", State: map[k.NodeID]float64{ids[0]: 1}}}})
	for i := 0; i < 8; i++ {
		out := nativeApply(t, rt, NativeControlCommand{ID: fmt.Sprintf("step-%d", i), Step: &NativeStep{EpisodeID: "ep", Sample: .01}})
		if out.Stopped {
			break
		}
	}
	cp := filepath.Join(t.TempDir(), "pre-outcome.json")
	if e = rt.Checkpoint(cp); e != nil {
		t.Fatal(e)
	}
	prefix, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	nativeEnd(t, rt, "crash-outcome", .9, true)
	want := continuationBytes(t, rt)
	childPath := filepath.Join(t.TempDir(), "events.jsonl")
	if e = os.WriteFile(childPath, prefix, 0600); e != nil {
		t.Fatal(e)
	}
	child := exec.Command(os.Args[0], "-test.run=^TestNativeProcessExitAfterFsync$")
	child.Env = append(os.Environ(), "HORIZON_NATIVE_CRASH_WAL="+childPath, "HORIZON_NATIVE_CRASH_CP="+cp)
	childOutput, childErr := child.CombinedOutput()
	if x, ok := childErr.(*exec.ExitError); !ok || x.ExitCode() != 76 {
		t.Fatalf("child did not exit after fsync: %v %s", childErr, childOutput)
	}
	log, e := OpenEventLog(childPath)
	if e != nil {
		t.Fatal(e)
	}
	defer log.Close()
	got, e := RebuildRuntimeFromCheckpointAndJournal(log, cp)
	if e != nil || !bytes.Equal(want, continuationBytes(t, got)) {
		t.Fatal("crash recovery mismatch", e)
	}
	events, e := ReadEventLog(childPath)
	if e != nil {
		t.Fatal(e)
	}
	last := events[len(events)-1]
	got.SetEventLog(log)
	duplicate, e := got.ApplyNativeControl(*last.NativeControl, last.Timestamp)
	if e != nil || !duplicate.Duplicate {
		t.Fatal("recovered retry changed knowledge", e)
	}
}
func TestNativeConcurrentCanonicalRead(t *testing.T) {
	rt, ids := nativeFixture(t)
	cfg := nativeControlConfig(ids)
	nativeApply(t, rt, NativeControlCommand{ID: "init", Initialize: &cfg})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 20; i++ {
			_, _ = rt.brain.CanonicalJSON()
			_ = rt.brain.NativeControlSnapshot()
		}
	}()
	for ep := 0; ep < 8; ep++ {
		id := fmt.Sprintf("ep%d", ep)
		nativeApply(t, rt, NativeControlCommand{ID: id, Begin: &NativeBegin{EpisodeID: id, Training: true, Observation: &NativeObservation{Source: "fixture", SampleID: id, State: map[k.NodeID]float64{ids[0]: 1}}}})
		for j := 0; j < 8; j++ {
			out := nativeApply(t, rt, NativeControlCommand{ID: fmt.Sprintf("%s-%d", id, j), Step: &NativeStep{EpisodeID: id, Sample: .01}})
			if out.Stopped {
				break
			}
		}
		nativeEnd(t, rt, id+"-out", .7, true)
	}
	wg.Wait()
}
