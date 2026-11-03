package runtime

import (
	"bytes"
	"encoding/json"
	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func phase4Fixture(t *testing.T) (*BrainRuntime, *EventLog, string) {
	t.Helper()
	wal, e := OpenEventLog(filepath.Join(t.TempDir(), "events.jsonl"))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { wal.Close() })
	rt := NewBrainRuntime(k.NewBrain())
	rt.SetEventLog(wal)
	c := k.CognitiveModelConfig{Version: k.CognitiveModelV2, Inputs: 28, Hidden: 32, Actions: 8, Horizon: 4, Seed: 101, LearningRate: .003, Gamma: 1, ValueWeight: .2, EntropyWeight: .02, AssessmentWeight: .2, NeuralMeta: &k.NeuralMetaConfig{Classes: 4, Computations: 3, Outcomes: 3, ContextSize: 5, ResultOffset: 5, OperationOffset: 8, ActionOffset: 11, HypothesisOffset: 19, PlanMarker: 23, StepIndex: 24, JointWeight: .7, RewardWeight: .3, PlanDepth: 1}}
	at := time.Unix(2200200000, 0)
	if _, e = rt.ApplyCognitiveModel(CognitiveModelCommand{ID: "p4-init", Initialize: &c}, at); e != nil {
		t.Fatal(e)
	}
	s := k.Phase2SpecV5{Version: k.Phase2VersionV5, Representation: "pairwise", Optimizer: "historical-global"}
	if _, e = rt.ApplyCognitiveModel(CognitiveModelCommand{ID: "p4-world", Phase2Init: &s}, at); e != nil {
		t.Fatal(e)
	}
	base := filepath.Join(filepath.Dir(wal.file.Name()), "base.json")
	if e = rt.Checkpoint(base); e != nil {
		t.Fatal(e)
	}
	return rt, wal, base
}
func TestPhase4CanonicalNOnPolicyAdamAndSchemaReplay(t *testing.T) {
	rt, wal, base := phase4Fixture(t)
	brain := rt.brain
	before, _ := rt.CognitiveModelParameters()
	s := k.Phase4SpecV5{Version: k.Phase4VersionV5, Seed: 101, LearningRate: .003, EntropyWeight: .02, ValueWeight: .2}
	at := time.Unix(2200200001, 0)
	if _, e := rt.ApplyCognitiveModel(CognitiveModelCommand{ID: "p4-meta", Phase4Init: &s}, at); e != nil {
		t.Fatal(e)
	}
	allocated, _ := rt.CognitiveModelParameters()
	if len(allocated)-len(before) != 1097 || rt.brain != brain {
		t.Fatal("not1097 in canonical N")
	}
	for i, v := range before {
		if allocated[i] != v {
			t.Fatal("baseline moved at allocation")
		}
	}
	x := k.CognitiveInput{Features: make([]float64, 28), Mask: []bool{true, true, true, true, true, true, true, true}, Reset: true, Draw: .99, NeuralMeta: &k.NeuralMetaInput{Mode: "muse-bmps", Costs: []float64{.025, .025, .025}, Remaining: 3}}
	x.Features[0] = 1
	p, e := rt.PreviewPhase4V5(k.Phase2ProbeV5{Trace: []k.CognitiveInput{x}}, k.Phase4DropV5{})
	if e != nil || p.Action != 7 {
		t.Fatal(e, p.Action)
	}
	ep := k.Phase4EpisodeV5{CaseID: "observed-abstain", Frames: []k.Phase4FrameV5{{Features: p.Features, Mask: x.Mask, Action: p.Action, Reward: .35, Draw: x.Draw, BehaviorProbability: p.Policy[p.Action], BehaviorValue: p.Value}}}
	b := k.Phase4BatchV5{Episodes: []k.Phase4EpisodeV5{ep}, ParametersHash: k.CognitiveDigest(allocated), Evidence: k.CognitiveEvidence{ID: "p4-evidence", Source: "native-observed", Verifier: "actual-terminal", Digest: k.CognitiveDigest([]k.Phase4EpisodeV5{ep})}}
	out, e := rt.ApplyCognitiveModel(CognitiveModelCommand{ID: "p4-update", Phase4Training: &b}, at.Add(time.Second))
	if e != nil || out.Phase4Learning.Steps != 1 {
		t.Fatal(e)
	}
	after, _ := rt.CognitiveModelParameters()
	if k.CognitiveDigest(allocated) == k.CognitiveDigest(after) {
		t.Fatal("no neural update")
	}
	for i, v := range before {
		if after[i] != v {
			t.Fatal("frozen prefix moved")
		}
	}
	if _, e = rt.ApplyCognitiveModel(CognitiveModelCommand{ID: "stale", Phase4Training: &b}, at); e == nil {
		t.Fatal("offpolicy accepted")
	}
	recovered, e := RebuildRuntimeFromCheckpointAndJournal(wal, base)
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(continuationBytes(t, rt), continuationBytes(t, recovered)) {
		t.Fatal("native replay differs")
	}
	snap := filepath.Join(filepath.Dir(base), "trained.json")
	if e = rt.Checkpoint(snap); e != nil {
		t.Fatal(e)
	}
	rest := NewBrainRuntime(nil)
	if e = rest.RestoreCheckpoint(snap); e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(continuationBytes(t, rt), continuationBytes(t, rest)) {
		t.Fatal("snapshot18 roundtrip differs")
	}
	events, e := ReadEventLog(wal.file.Name())
	if e != nil || events[len(events)-1].SchemaVersion != 11 {
		t.Fatal(e)
	}
	bad := events[len(events)-1]
	bad.SchemaVersion = 10
	if validateJournalPayload(bad) == nil {
		t.Fatal("downgrade accepted")
	}
	t.Log("N+1097, snapshot18/schema11; on-policy guard, Adam clock1, exact native replay and roundtrip")
}

func phase4ObservedBatch(t *testing.T, rt *BrainRuntime) k.Phase4BatchV5 {
	t.Helper()
	x := k.CognitiveInput{Features: make([]float64, 28), Mask: []bool{true, true, true, true, true, true, true, true}, Reset: true, Draw: .99, NeuralMeta: &k.NeuralMetaInput{Mode: "muse-bmps", Costs: []float64{.025, .025, .025}, Remaining: 3}}
	x.Features[0] = 1
	p, e := rt.PreviewPhase4V5(k.Phase2ProbeV5{Trace: []k.CognitiveInput{x}}, k.Phase4DropV5{})
	if e != nil || p.Action != 7 {
		t.Fatal(e)
	}
	eps := []k.Phase4EpisodeV5{{CaseID: "process-exit-observed-terminal", Frames: []k.Phase4FrameV5{{Features: p.Features, Mask: x.Mask, Action: p.Action, Reward: .35, Draw: .99, BehaviorProbability: p.Policy[p.Action], BehaviorValue: p.Value}}}}
	w, e := rt.CognitiveModelParameters()
	if e != nil {
		t.Fatal(e)
	}
	return k.Phase4BatchV5{Episodes: eps, ParametersHash: k.CognitiveDigest(w), Evidence: k.CognitiveEvidence{ID: "exit-outcome", Source: "actual-observed", Verifier: "terminal-abstain", Digest: k.CognitiveDigest(eps)}}
}
func TestPhase4RealProcessExitSchema11AfterFsync(t *testing.T) {
	if path := os.Getenv("HORIZON_P4_CRASH_WAL"); path != "" {
		wal, e := OpenEventLog(path)
		if e != nil {
			t.Fatal(e)
		}
		rt, e := RebuildRuntimeFromCheckpointAndJournal(wal, os.Getenv("HORIZON_P4_CRASH_CP"))
		if e != nil {
			t.Fatal(e)
		}
		rt.SetEventLog(wal)
		b := phase4ObservedBatch(t, rt)
		wal.beforeAppend = func(ev LoggedEvent) error {
			if ev.SchemaVersion != 11 {
				t.Fatal("not schema11")
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
			os.Exit(75)
			return nil
		}
		rt.ApplyCognitiveModel(CognitiveModelCommand{ID: "p4-real-exit", Phase4Training: &b}, time.Unix(2300000002, 0))
		t.Fatal("child did not exit")
	}
	rt, wal, base := phase4Fixture(t)
	spec := k.Phase4SpecV5{Version: k.Phase4VersionV5, Seed: 101, LearningRate: .003, EntropyWeight: .02, ValueWeight: .2}
	if _, e := rt.ApplyCognitiveModel(CognitiveModelCommand{ID: "exit-meta-init", Phase4Init: &spec}, time.Unix(2300000001, 0)); e != nil {
		t.Fatal(e)
	}
	allocated := filepath.Join(filepath.Dir(base), "allocated.json")
	if e := rt.Checkpoint(allocated); e != nil {
		t.Fatal(e)
	}
	path := wal.file.Name()
	wal.Close()
	child := exec.Command(os.Args[0], "-test.run=^TestPhase4RealProcessExitSchema11AfterFsync$")
	child.Env = append(os.Environ(), "HORIZON_P4_CRASH_WAL="+path, "HORIZON_P4_CRASH_CP="+allocated)
	e := child.Run()
	if x, ok := e.(*exec.ExitError); !ok || x.ExitCode() != 75 {
		t.Fatalf("exit: %v", e)
	}
	reopened, e := OpenEventLog(path)
	if e != nil {
		t.Fatal(e)
	}
	defer reopened.Close()
	got, e := RebuildRuntimeFromCheckpointAndJournal(reopened, allocated)
	if e != nil {
		t.Fatal(e)
	}
	rt.SetEventLog(nil)
	b := phase4ObservedBatch(t, rt)
	if _, e = rt.ApplyCognitiveModel(CognitiveModelCommand{ID: "p4-real-exit", Phase4Training: &b}, time.Unix(2300000002, 0)); e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(continuationBytes(t, rt), continuationBytes(t, got)) {
		t.Fatal("schema11 durable redo differs")
	}
	before := continuationBytes(t, got)
	again, e := got.ApplyCognitiveModel(CognitiveModelCommand{ID: "p4-real-exit", Phase4Training: &b}, time.Unix(2300000003, 0))
	if e != nil || !again.Duplicate || !bytes.Equal(before, continuationBytes(t, got)) {
		t.Fatal("duplicate retrained N", e)
	}
	t.Log("actual process exit after schema11 fsync before neural mutation; snapshot18 native recovery bit-exact and no double training")
}
