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

func terminalFixtureBatch(t *testing.T, rt *BrainRuntime) k.Phase4BatchV5 {
	t.Helper()
	x := k.CognitiveInput{Features: make([]float64, 28), Mask: []bool{true, true, true, true, true, true, true, true}, Reset: true, Draw: .99, NeuralMeta: &k.NeuralMetaInput{Mode: "muse-bmps", Costs: []float64{.025, .025, .025}, Remaining: 3}}
	x.Features[0] = 1
	p, e := rt.PreviewPhase4V5(k.Phase2ProbeV5{Trace: []k.CognitiveInput{x}}, k.Phase4DropV5{})
	if e != nil {
		t.Fatal(e)
	}
	ps, e := rt.PreviewPhase4TerminalFeaturesV5([][]float64{p.Features}, []float64{.99}, false, k.Phase4DropV5{})
	if e != nil {
		t.Fatal(e)
	}
	p = ps[0]
	eps := []k.Phase4EpisodeV5{{CaseID: "terminal-observed", Acquisition: &k.Phase4TerminalAcquisitionV5{TraceDigest: k.CognitiveDigest([]k.CognitiveInput{x})}, Frames: []k.Phase4FrameV5{{Features: p.Features, Mask: []bool{false, false, false, true, true, true, true, true}, Action: p.Action, Reward: .35, Draw: .99, BehaviorProbability: p.Policy[p.Action], BehaviorValue: p.Value}}}}
	w, e := rt.CognitiveModelParameters()
	if e != nil {
		t.Fatal(e)
	}
	return k.Phase4BatchV5{Protocol: k.Phase4TerminalProtocolV5, Episodes: eps, ParametersHash: k.CognitiveDigest(w), Evidence: k.CognitiveEvidence{ID: "terminal-observed", Source: "controlled-acquisition", Verifier: "actual-terminal-outcome", Digest: k.CognitiveDigest(eps)}}
}

func TestTerminalExposureCanonicalReplayAndGuards(t *testing.T) {
	rt, wal, _ := phase4Fixture(t)
	s := k.Phase4SpecV5{Version: k.Phase4VersionV5, Seed: 101, LearningRate: .003, EntropyWeight: .02, ValueWeight: .2}
	if _, e := rt.ApplyCognitiveModel(CognitiveModelCommand{ID: "terminal-init", Phase4Init: &s}, time.Unix(2400000000, 0)); e != nil {
		t.Fatal(e)
	}
	cp := filepath.Join(t.TempDir(), "parent.json")
	if e := rt.Checkpoint(cp); e != nil {
		t.Fatal(e)
	}
	before, _ := rt.CognitiveModelParameters()
	brain := rt.brain
	b := terminalFixtureBatch(t, rt)
	bad := b
	bad.Protocol = ""
	if _, e := rt.ApplyCognitiveModel(CognitiveModelCommand{ID: "illegal-protocol", Phase4Training: &bad}, time.Now()); e == nil {
		t.Fatal("terminal exposure accepted as historical full policy")
	}
	copyBatch := func() k.Phase4BatchV5 {
		raw, _ := json.Marshal(b)
		var out k.Phase4BatchV5
		json.Unmarshal(raw, &out)
		return out
	}
	for _, fault := range []string{"answer-mask", "budget", "cost", "duplicate-acquisition", "reward"} {
		bad = copyBatch()
		f := &bad.Episodes[0].Frames[0]
		switch fault {
		case "answer-mask":
			f.Mask[7] = false
		case "budget":
			f.Features[48] = 2
		case "cost":
			bad.Episodes[0].Acquisition.Cost = .1
		case "duplicate-acquisition":
			bad.Episodes[0].Acquisition.Observed = []k.VerifiedMentalResult{{Operation: 0, Outcome: 0}, {Operation: 0, Outcome: 1}}
		case "reward":
			f.Reward = .4
		}
		bad.Evidence.Digest = k.CognitiveDigest(bad.Episodes)
		if _, e := rt.ApplyCognitiveModel(CognitiveModelCommand{ID: fault, Phase4Training: &bad}, time.Now()); e == nil {
			t.Fatal("bad terminal batch accepted", fault)
		}
	}
	out, e := rt.ApplyCognitiveModel(CognitiveModelCommand{ID: "terminal-fit", Phase4Training: &b}, time.Unix(2400000001, 0))
	if e != nil || out.Phase4Learning.Steps != 1 {
		t.Fatal(e)
	}
	after, _ := rt.CognitiveModelParameters()
	if rt.brain != brain || k.CognitiveDigest(before) == k.CognitiveDigest(after) {
		t.Fatal("not the same learned canonical brain")
	}
	for i := 0; i < len(before)-1097; i++ {
		if before[i] != after[i] {
			t.Fatal("frozen prefix changed")
		}
	}
	rec, e := RebuildRuntimeFromCheckpointAndJournal(wal, cp)
	if e != nil || !bytes.Equal(continuationBytes(t, rt), continuationBytes(t, rec)) {
		t.Fatal("terminal replay mismatch", e)
	}
	events, e := ReadEventLog(wal.file.Name())
	if e != nil {
		t.Fatal(e)
	}
	last := events[len(events)-1]
	if last.SchemaVersion != 12 {
		t.Fatal("not explicit schema12")
	}
	last.SchemaVersion = 11
	if validateJournalPayload(last) == nil {
		t.Fatal("terminal downgrade accepted")
	}
	last = events[len(events)-1]
	last.CognitiveModel.Phase4Training.Protocol = ""
	if validateJournalPayload(last) == nil {
		t.Fatal("schema12 accepted legacy payload")
	}
	if _, e = rt.ApplyCognitiveModel(CognitiveModelCommand{ID: "stale-terminal", Phase4Training: &b}, time.Now()); e == nil {
		t.Fatal("stale terminal behavior accepted")
	}
	t.Log("one canonical N; terminal policy and reward guards; schema12 downgrade/mixed rejection; strict native replay")
}

func TestTerminalExposureProcessExitAfterFsync(t *testing.T) {
	if path := os.Getenv("HORIZON_TERMINAL_CRASH_WAL"); path != "" {
		wal, e := OpenEventLog(path)
		if e != nil {
			t.Fatal(e)
		}
		rt, e := RebuildRuntimeFromCheckpointAndJournal(wal, os.Getenv("HORIZON_TERMINAL_CRASH_CP"))
		if e != nil {
			t.Fatal(e)
		}
		rt.SetEventLog(wal)
		b := terminalFixtureBatch(t, rt)
		wal.beforeAppend = func(ev LoggedEvent) error {
			if ev.SchemaVersion != 12 {
				t.Fatal("not schema12")
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
		rt.ApplyCognitiveModel(CognitiveModelCommand{ID: "terminal-crash", Phase4Training: &b}, time.Unix(2400000001, 0))
		t.Fatal("no process exit")
	}
	rt, wal, _ := phase4Fixture(t)
	spec := k.Phase4SpecV5{Version: k.Phase4VersionV5, Seed: 101, LearningRate: .003, EntropyWeight: .02, ValueWeight: .2}
	if _, e := rt.ApplyCognitiveModel(CognitiveModelCommand{ID: "terminal-init", Phase4Init: &spec}, time.Unix(2400000000, 0)); e != nil {
		t.Fatal(e)
	}
	cp := filepath.Join(t.TempDir(), "parent.json")
	if e := rt.Checkpoint(cp); e != nil {
		t.Fatal(e)
	}
	b := terminalFixtureBatch(t, rt)
	expected, e := rt.ApplyCognitiveModel(CognitiveModelCommand{ID: "terminal-crash", Phase4Training: &b}, time.Unix(2400000001, 0))
	if e != nil {
		t.Fatal(e)
	}
	_ = expected
	want := continuationBytes(t, rt)
	// Use a separately replayable, authenticated prefix before the first update.
	events, e := ReadEventLog(wal.file.Name())
	if e != nil {
		t.Fatal(e)
	}
	childPath := filepath.Join(t.TempDir(), "events.jsonl")
	var buf bytes.Buffer
	for _, ev := range events[:len(events)-1] {
		raw, _ := json.Marshal(ev)
		buf.Write(raw)
		buf.WriteByte('\n')
	}
	if e = os.WriteFile(childPath, buf.Bytes(), 0600); e != nil {
		t.Fatal(e)
	}
	child := exec.Command(os.Args[0], "-test.run=^TestTerminalExposureProcessExitAfterFsync$")
	child.Env = append(os.Environ(), "HORIZON_TERMINAL_CRASH_WAL="+childPath, "HORIZON_TERMINAL_CRASH_CP="+cp)
	if x, ok := child.Run().(*exec.ExitError); !ok || x.ExitCode() != 76 {
		t.Fatal("child did not exit after durable append")
	}
	log, e := OpenEventLog(childPath)
	if e != nil {
		t.Fatal(e)
	}
	defer log.Close()
	got, e := RebuildRuntimeFromCheckpointAndJournal(log, cp)
	if e != nil || !bytes.Equal(want, continuationBytes(t, got)) {
		t.Fatal("post-fsync terminal recovery mismatch", e)
	}
	got.SetEventLog(log)
	again, e := got.ApplyCognitiveModel(CognitiveModelCommand{ID: "terminal-crash", Phase4Training: &b}, time.Unix(2400000001, 0))
	if e != nil || !again.Duplicate {
		t.Fatal("terminal recovery not idempotent", e)
	}
}
