package runtime

import (
	"bytes"
	"encoding/json"
	"fmt"
	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func contrastCrashOutcome(t *testing.T, rt *BrainRuntime) NativeControlResult {
	ep := rt.brain.NativeControlSnapshot().Episode
	return nativeApply(t, rt, NativeControlCommand{ID: "second-out", Outcome: &NativeOutcome{EpisodeID: ep.ID, TraceDigest: k.NativeDigest(ep), Source: "crash-environment", OutcomeID: "second", Provenance: "observed-environment", Reward: .2, Learn: true, ContrastID: "pair"}})
}
func TestNativeContrastCrashBoundary(t *testing.T) {
	if path := os.Getenv("HORIZON_PAIR_CRASH_WAL"); path != "" {
		wal, err := OpenEventLog(path)
		if err != nil {
			t.Fatal(err)
		}
		rt, err := RebuildRuntimeFromCheckpointAndJournal(wal, os.Getenv("HORIZON_PAIR_CRASH_CP"))
		if err != nil {
			t.Fatal(err)
		}
		rt.SetEventLog(wal)
		wal.beforeAppend = func(event LoggedEvent) error {
			if os.Getenv("HORIZON_PAIR_CRASH_MODE") == "after" {
				data, err := json.Marshal(event)
				if err != nil {
					t.Fatal(err)
				}
				if _, err = wal.file.Write(append(data, '\n')); err != nil {
					t.Fatal(err)
				}
				if err = wal.file.Sync(); err != nil {
					t.Fatal(err)
				}
			}
			os.Exit(76)
			return nil
		}
		contrastCrashOutcome(t, rt)
		t.Fatal("no exit")
	}
	rt, ids := nativeFixture(t)
	path := filepath.Join(t.TempDir(), "events.jsonl")
	wal, err := OpenEventLog(path)
	if err != nil {
		t.Fatal(err)
	}
	defer wal.Close()
	rt.SetEventLog(wal)
	cfg := nativeControlConfig(ids)
	cfg.Version = k.NativeControlRelationalVersion
	cfg.Encoder = "local-relational-v1"
	nativeApply(t, rt, NativeControlCommand{ID: "init", Initialize: &cfg})
	for trial, v := range []float64{.8, .2} {
		id := fmt.Sprintf("trial%d", trial)
		nativeApply(t, rt, NativeControlCommand{ID: id, Begin: &NativeBegin{EpisodeID: id, Training: true, Intervention: true, Observation: &NativeObservation{Source: "crash-environment", SampleID: id, State: map[k.NodeID]float64{ids[0]: v, ids[6]: .8}}}})
		for j, a := range []int{0, 3} {
			action := a
			nativeApply(t, rt, NativeControlCommand{ID: fmt.Sprintf("%s-%d", id, j), Step: &NativeStep{EpisodeID: id, ExperimentAction: &action}})
		}
		if trial == 0 {
			ep := rt.brain.NativeControlSnapshot().Episode
			nativeApply(t, rt, NativeControlCommand{ID: "first-out", Outcome: &NativeOutcome{EpisodeID: id, TraceDigest: k.NativeDigest(ep), Source: "crash-environment", OutcomeID: "first", Provenance: "observed-environment", Reward: .8, Learn: true, ContrastID: "pair"}})
		}
	}
	cp := filepath.Join(t.TempDir(), "before.json")
	if err = rt.Checkpoint(cp); err != nil {
		t.Fatal(err)
	}
	prefix, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	before := continuationBytes(t, rt)
	contrastCrashOutcome(t, rt)
	after := continuationBytes(t, rt)
	for _, mode := range []string{"before", "after"} {
		t.Run(mode, func(t *testing.T) {
			childPath := filepath.Join(t.TempDir(), "events.jsonl")
			if err = os.WriteFile(childPath, prefix, 0600); err != nil {
				t.Fatal(err)
			}
			child := exec.Command(os.Args[0], "-test.run=^TestNativeContrastCrashBoundary$")
			child.Env = append(os.Environ(), "HORIZON_PAIR_CRASH_WAL="+childPath, "HORIZON_PAIR_CRASH_CP="+cp, "HORIZON_PAIR_CRASH_MODE="+mode)
			output, err := child.CombinedOutput()
			if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 76 {
				t.Fatalf("child exit: %v %s", err, output)
			}
			log, err := OpenEventLog(childPath)
			if err != nil {
				t.Fatal(err)
			}
			defer log.Close()
			rebuilt, err := RebuildRuntimeFromCheckpointAndJournal(log, cp)
			if err != nil {
				t.Fatal(err)
			}
			want := before
			if mode == "after" {
				want = after
			}
			if !bytes.Equal(want, continuationBytes(t, rebuilt)) {
				t.Fatal("incorrect credit at crash boundary")
			}
		})
	}
}
