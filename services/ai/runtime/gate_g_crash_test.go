package runtime

import (
	"bytes"
	"encoding/json"
	"errors"
	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

type gateGCrashPlan struct {
	Kind, Boundary, EventType string
	Command                   NativeControlCommand
	Sensor                    SensorObservation
	At                        time.Time
}

// The child is killed inside the real transaction boundaries, not by returning
// an exception to the caller. All paths are isolated fixtures owned by this test.
func TestGateGProcessCrash(t *testing.T) {
	gateGProcessCrash(t, k.NativeRelationalEncoderV1)
}

func TestNativeV2GateGProcessCrash(t *testing.T) {
	gateGProcessCrash(t, k.NativeRelationalEncoderV2)
}

func gateGProcessCrash(t *testing.T, encoder string) {
	if dir := os.Getenv("HORIZON_GATE_G_CRASH_CHILD"); dir != "" {
		var plan gateGCrashPlan
		b, e := os.ReadFile(filepath.Join(dir, "plan.json"))
		if e != nil {
			t.Fatal(e)
		}
		if e = json.Unmarshal(b, &plan); e != nil {
			t.Fatal(e)
		}
		wal, e := OpenEventLog(filepath.Join(dir, "events.jsonl"))
		if e != nil {
			t.Fatal(e)
		}
		rt, e := RebuildRuntimeFromCheckpointAndJournal(wal, filepath.Join(dir, "checkpoint.json"))
		if e != nil {
			t.Fatal(e)
		}
		rt.SetEventLog(wal)
		rt.transactionFault = func(boundary string) {
			if boundary == plan.Boundary {
				os.Exit(86)
			}
		}
		wal.transactionFault = func(boundary string, event LoggedEvent) {
			if boundary == plan.Boundary && (plan.EventType == "" || plan.EventType == event.Type) {
				os.Exit(86)
			}
		}
		switch plan.Kind {
		case "native":
			_, e = rt.ApplyNativeControl(plan.Command, plan.At)
		case "sensor":
			_, e = rt.ProcessSensorObservation(plan.Sensor)
		case "checkpoint":
			e = rt.Checkpoint(filepath.Join(dir, "checkpoint.json"))
		case "supervisor":
			var s *RuntimeSupervisor
			s, e = NewRuntimeSupervisor(rt)
			if e == nil {
				e = s.RecoverFromCheckpoint(wal, filepath.Join(dir, "checkpoint.json"))
			}
		}
		t.Fatalf("fault did not exit at %s: %v", plan.Boundary, e)
	}
	cases := []struct {
		name, scenario, boundary, event string
		committed, orphan               bool
	}{
		{"payload", "step", "payload-preparation", "", false, false},
		{"before-append", "step", "before-append", "", false, false},
		{"write-before-sync", "step", "after-write-before-sync", "", true, false},
		{"fsync-before-apply", "step", "after-sync-before-apply", "", true, false},
		{"apply-before-ack", "step", "after-apply-before-ack", "", true, false},
		{"sensor-intent", "sensor", "sensor-after-intent", "", false, true},
		{"sensor-partial-apply", "sensor", "sensor-after-apply-before-commit", "", false, true},
		{"sensor-commit-fsync", "sensor", "after-sync-before-apply", EventTypeSensorCommit, true, false},
		{"active-episode", "begin", "after-apply-before-ack", "", true, false},
		{"first-outcome", "first", "after-sync-before-apply", "", true, false},
		{"second-outcome", "second", "after-sync-before-apply", "", true, false},
		{"pending-contrast", "pending", "checkpoint-after-rename-sync", "", true, false},
		{"checkpoint-before-rename", "checkpoint", "checkpoint-before-rename", "", true, false},
		{"checkpoint-after-sync", "checkpoint", "checkpoint-after-rename-sync", "", true, false},
		{"supervisor-rebuild", "supervisor", "supervisor-after-rebuild", "", true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rt, ids := nativeFixture(t)
			parent := t.TempDir()
			wal, e := OpenEventLog(filepath.Join(parent, "events.jsonl"))
			if e != nil {
				t.Fatal(e)
			}
			defer wal.Close()
			rt.SetEventLog(wal)
			gateGSensor(t, rt, "production-input")
			cfg := nativeControlConfig(ids)
			cfg.Version = k.NativeControlCausalVersion
			cfg.Encoder = encoder
			nativeApply(t, rt, NativeControlCommand{ID: "init", Initialize: &cfg})
			gateGNative(t, rt, "prior-learning")
			gateGSensor(t, rt, "target-context")
			plan := gateGCrashPlan{Kind: "native", Boundary: tc.boundary, EventType: tc.event}
			cp := filepath.Join(parent, "checkpoint.json")
			if tc.scenario == "checkpoint" || tc.scenario == "supervisor" {
				if e = rt.Checkpoint(cp); e != nil {
					t.Fatal(e)
				}
				gateGSensor(t, rt, "after-old-checkpoint")
				gateGNative(t, rt, "suffix-learning")
			}
			switch tc.scenario {
			case "step":
				nativeApply(t, rt, NativeControlCommand{ID: "begin-target", Begin: &NativeBegin{EpisodeID: "target", Training: true}})
				plan.Command = NativeControlCommand{ID: "target-step", Step: &NativeStep{EpisodeID: "target", Sample: .01}}
			case "begin":
				plan.Command = NativeControlCommand{ID: "target-begin", Begin: &NativeBegin{EpisodeID: "target", Training: true}}
			case "sensor":
				plan.Kind = "sensor"
				plan.Sensor = testSensorObservation()
				plan.Sensor.SampleID = "target-sensor"
				plan.Sensor.Timestamp = gateGTime(rt)
			case "first", "second", "pending":
				state := map[k.NodeID]float64{ids[0]: .8, ids[6]: .4}
				first := actionTrial(t, rt, state, "first", "frozen-world", []int{0, 3}, .8)
				if tc.scenario == "first" {
					plan.Command = first
				} else {
					nativeApply(t, rt, first)
					if tc.scenario == "second" {
						plan.Command = actionTrial(t, rt, state, "second", "frozen-world", []int{3}, .2)
					} else {
						plan.Kind = "checkpoint"
					}
				}
			case "checkpoint":
				plan.Kind = "checkpoint"
			case "supervisor":
				plan.Kind = "supervisor"
			}
			if tc.scenario != "checkpoint" && tc.scenario != "supervisor" {
				if e = rt.Checkpoint(cp); e != nil {
					t.Fatal(e)
				}
			}
			cpBytes, e := os.ReadFile(cp)
			if e != nil {
				t.Fatal(e)
			}
			var oldCP BrainSnapshot
			_ = json.Unmarshal(cpBytes, &oldCP)
			plan.At = gateGTime(rt)
			before := continuationBytes(t, rt)
			raw, e := os.ReadFile(wal.file.Name())
			if e != nil {
				t.Fatal(e)
			}
			// The parent executes the exact command once to establish the reference.
			switch plan.Kind {
			case "native":
				_, e = rt.ApplyNativeControl(plan.Command, plan.At)
			case "sensor":
				_, e = rt.ProcessSensorObservation(plan.Sensor)
			}
			if e != nil {
				t.Fatal(e)
			}
			after := continuationBytes(t, rt)
			childDir := t.TempDir()
			for name, data := range map[string][]byte{"checkpoint.json": cpBytes, "events.jsonl": raw} {
				if e = os.WriteFile(filepath.Join(childDir, name), data, 0600); e != nil {
					t.Fatal(e)
				}
			}
			b, e := json.Marshal(plan)
			if e != nil {
				t.Fatal(e)
			}
			if e = os.WriteFile(filepath.Join(childDir, "plan.json"), b, 0600); e != nil {
				t.Fatal(e)
			}
			cmd := exec.Command(os.Args[0], "-test.run=^TestGateGProcessCrash$")
			cmd.Env = append(os.Environ(), "HORIZON_GATE_G_CRASH_CHILD="+childDir)
			output, e := cmd.CombinedOutput()
			if ex, ok := e.(*exec.ExitError); !ok || ex.ExitCode() != 86 {
				t.Fatalf("unexpected child exit: %v\n%s", e, output)
			}
			log, e := OpenEventLog(filepath.Join(childDir, "events.jsonl"))
			if e != nil {
				t.Fatal(e)
			}
			defer log.Close()
			if tc.orphan {
				if _, e = RebuildRuntimeFromCheckpointAndJournal(log, filepath.Join(childDir, "checkpoint.json")); !errors.Is(e, ErrUncommittedSensorTransaction) {
					t.Fatalf("orphan not rejected: %v", e)
				}
				original, e := os.ReadFile(log.file.Name())
				if e != nil {
					t.Fatal(e)
				}
				backup, e := log.RecoverTrailingSensorIntent()
				if e != nil {
					t.Fatal(e)
				}
				back, e := os.ReadFile(backup)
				if e != nil || !bytes.Equal(back, original) {
					t.Fatal("repair lost original evidence", e)
				}
			}
			rebuilt, e := RebuildRuntimeFromCheckpointAndJournal(log, filepath.Join(childDir, "checkpoint.json"))
			if e != nil {
				t.Fatal(e)
			}
			want := before
			if tc.committed {
				want = after
			}
			if !bytes.Equal(want, continuationBytes(t, rebuilt)) {
				t.Fatal("incorrect complete durable state after process crash")
			}
			if tc.scenario == "second" {
				retry, e := rebuilt.ApplyNativeControl(plan.Command, plan.At)
				if e != nil || !retry.Duplicate || !bytes.Equal(want, continuationBytes(t, rebuilt)) {
					t.Fatal("recovery duplicated factual-pair learning", e)
				}
				changed := plan.Command
				out := *changed.Outcome
				out.Reward = .9
				changed.Outcome = &out
				if _, e = rebuilt.ApplyNativeControl(changed, plan.At); e == nil {
					t.Fatal("changed observed outcome accepted")
				}
			}
			if tc.scenario == "checkpoint" {
				data, e := os.ReadFile(filepath.Join(childDir, "checkpoint.json"))
				if e != nil {
					t.Fatal(e)
				}
				var recoveredCP BrainSnapshot
				_ = json.Unmarshal(data, &recoveredCP)
				expected := oldCP.Sequence
				if tc.name == "checkpoint-after-sync" {
					expected = rt.LastSequence()
				}
				if recoveredCP.Sequence != expected {
					t.Fatal("wrong checkpoint publication boundary")
				}
			}
			if root := os.Getenv("HORIZON_GATE_G_EVIDENCE_DIR"); root != "" {
				if encoder == k.NativeRelationalEncoderV2 {
					root = filepath.Join(root, encoder)
				}
				dest := filepath.Join(root, "crashes", tc.name)
				if e = os.MkdirAll(dest, 0755); e != nil {
					t.Fatal(e)
				}
				for _, name := range []string{"events.jsonl", "plan.json", "checkpoint.json"} {
					data, e := os.ReadFile(filepath.Join(childDir, name))
					if e != nil {
						t.Fatal(e)
					}
					if e = os.WriteFile(filepath.Join(dest, name), data, 0600); e != nil {
						t.Fatal(e)
					}
				}
				if e = os.WriteFile(filepath.Join(dest, "expected-continuation.json"), want, 0600); e != nil {
					t.Fatal(e)
				}
			}
			t.Logf("exit=86 expected_seq=%d checksum=%s pending=%s strict_orphan=%t", rebuilt.LastSequence(), k.NativeDigest(json.RawMessage(want)), rebuilt.brain.NativeAdmission(), tc.orphan)
		})
	}
}
