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

func TestCoordinatedV5ProcessExitAfterFsync(t *testing.T) {
	testCoordinatedCrashV5(t, "recurrent")
}

func TestFidelityV5ProcessExitAfterFsync(t *testing.T) {
	testCoordinatedCrashV5(t, k.CoordinatedFidelityV5)
}

func testCoordinatedCrashV5(t *testing.T, mode string) {
	if requested := os.Getenv("HORIZON_COORDINATED_TEST_MODE"); requested != "" {
		mode = requested
	}
	if path := os.Getenv("HORIZON_COORDINATED_CRASH_WAL"); path != "" {
		wal, e := OpenEventLog(path)
		if e != nil {
			t.Fatal(e)
		}
		rt, e := RebuildRuntimeFromCheckpointAndJournal(wal, os.Getenv("HORIZON_COORDINATED_CRASH_CP"))
		if e != nil {
			t.Fatal(e)
		}
		rt.SetEventLog(wal)
		b := coordinatedRuntimeBatchWithMode(t, rt, mode)
		wal.beforeAppend = func(ev LoggedEvent) error {
			if ev.SchemaVersion != 13 {
				t.Fatal("not schema13")
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
		rt.ApplyCognitiveModel(CognitiveModelCommand{ID: "terminal-crash", Coordinated: &b}, time.Unix(2400000001, 0))
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
	b := coordinatedRuntimeBatchWithMode(t, rt, mode)
	expected, e := rt.ApplyCognitiveModel(CognitiveModelCommand{ID: "terminal-crash", Coordinated: &b}, time.Unix(2400000001, 0))
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
	child := exec.Command(os.Args[0], "-test.run=^TestCoordinatedV5ProcessExitAfterFsync$")
	child.Env = append(os.Environ(), "HORIZON_COORDINATED_CRASH_WAL="+childPath, "HORIZON_COORDINATED_CRASH_CP="+cp, "HORIZON_COORDINATED_TEST_MODE="+mode)
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
	again, e := got.ApplyCognitiveModel(CognitiveModelCommand{ID: "terminal-crash", Coordinated: &b}, time.Unix(2400000001, 0))
	if e != nil || !again.Duplicate {
		t.Fatal("terminal recovery not idempotent", e)
	}
}
