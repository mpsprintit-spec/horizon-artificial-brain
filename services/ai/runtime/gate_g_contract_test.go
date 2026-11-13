package runtime

import (
	"bytes"
	"encoding/json"
	"errors"
	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGateGResolvedContrastCheckpointAndRollback(t *testing.T) {
	rt, ids := nativeFixture(t)
	wal := openKnowledgeJournal(t)
	rt.SetEventLog(wal)
	foundation := gateGFoundation(t, rt)
	oldCP := filepath.Join(t.TempDir(), "pre-native.json")
	if e := rt.Checkpoint(oldCP); e != nil {
		t.Fatal(e)
	}
	cfg := nativeControlConfig(ids)
	cfg.Version = k.NativeControlCausalVersion
	cfg.Encoder = "local-relational-v1"
	nativeApply(t, rt, NativeControlCommand{ID: "init", Initialize: &cfg})
	state := map[k.NodeID]float64{ids[0]: .8, ids[6]: .4}
	first := actionTrial(t, rt, state, "a", "world", []int{0, 3}, .8)
	nativeApply(t, rt, first)
	second := actionTrial(t, rt, state, "b", "world", []int{3}, .2)
	nativeApply(t, rt, second)
	cp := filepath.Join(t.TempDir(), "resolved.json")
	if e := rt.Checkpoint(cp); e != nil {
		t.Fatal(e)
	}
	before := continuationBytes(t, rt)
	for _, path := range []string{oldCP, cp} {
		recovered, e := RebuildRuntimeFromCheckpointAndJournal(wal, path)
		if e != nil {
			t.Fatal(e)
		}
		gateGCompare(t, rt, recovered)
		if !nativeApply(t, recovered, second).Duplicate || !bytes.Equal(before, continuationBytes(t, recovered)) {
			t.Fatal("duplicate settled pair relearned")
		}
	}
	// Explicit rollback to the matched prefix, with full history kept untouched.
	rollbackWal, e := OpenEventLog(filepath.Join(t.TempDir(), "prefix.jsonl"))
	if e != nil {
		t.Fatal(e)
	}
	defer rollbackWal.Close()
	rolled, e := RebuildRuntimeFromCheckpointAndJournal(rollbackWal, oldCP)
	if e != nil {
		t.Fatal(e)
	}
	pristine := NewBrainRuntime(foundation())
	gateGCompare(t, pristine, rolled)
	// New suffix cannot be paired with an old checkpoint belonging to another prefix.
	incompatible, e := OpenEventLog(filepath.Join(t.TempDir(), "other.jsonl"))
	if e != nil {
		t.Fatal(e)
	}
	defer incompatible.Close()
	if _, e = RebuildRuntimeFromCheckpointAndJournal(incompatible, cp); e == nil {
		t.Fatal("unmatched checkpoint/prefix accepted")
	}
	if e = rt.checkpointVersionLocked(filepath.Join(t.TempDir(), "downgrade.json"), BrainSnapshotSchemaVersion); e == nil {
		t.Fatal("native snapshot downgrade accepted")
	}
}

func TestGateGArchitectureDependencyBoundary(t *testing.T) {
	// The active dependency closure is checked, rather than banning retired
	// packages from migration tools elsewhere in the repository.
	cmd := exec.Command("go", "list", "-deps", ".")
	output, e := cmd.CombinedOutput()
	if e != nil {
		t.Fatalf("dependency audit: %v %s", e, output)
	}
	for _, line := range strings.Fields(string(output)) {
		for _, retired := range []string{"/thinking", "/understanding", "/decision", "/reasoning", "/execution"} {
			if strings.HasSuffix(line, retired) {
				t.Fatalf("active runtime depends on retired engine or physical dispatch: %s", line)
			}
		}
	}
	rt, ids := nativeFixture(t)
	cfg := nativeControlConfig(ids)
	nativeApply(t, rt, NativeControlCommand{ID: "init", Initialize: &cfg})
	if rt.brain.HasCognitiveModel() {
		t.Fatal("native opt-in initialized frozen V5 controller")
	}
	if rt.DNF() != nil {
		t.Fatal("native runtime exposed standalone mutation escape hatch")
	}
	called := false
	if e = rt.WithBrainView(func(*k.Brain) error { called = true; return nil }); e == nil || called {
		t.Fatal("native runtime exported mutable Brain")
	}
}

func TestGateGReplayFailureCannotBecomeServingRuntime(t *testing.T) {
	source, ids := nativeFixture(t)
	wal := openKnowledgeJournal(t)
	source.SetEventLog(wal)
	foundation := gateGFoundation(t, source)
	cfg := nativeControlConfig(ids)
	nativeApply(t, source, NativeControlCommand{ID: "init", Initialize: &cfg})
	events, e := ReadEventLog(wal.file.Name())
	if e != nil {
		t.Fatal(e)
	}
	bad := NativeControlCommand{ID: "bad", Step: &NativeStep{EpisodeID: "absent", Sample: .1}}
	events = append(events, LoggedEvent{SchemaVersion: 14, BrainIdentity: BrainIdentity, Sequence: 2, Type: EventTypeNativeControl, Timestamp: gateGTime(source), SurfaceEncoding: source.surfaceEncoding, DynamicsEncoding: source.dynamicsEncoding, NativeControl: &bad})
	target := NewBrainRuntime(foundation())
	if e = ReplayEventLog(target, events); e == nil || !target.RecoveryRequired() {
		t.Fatal("partial replay published usable state", e)
	}
	if _, e = NewRuntimeSupervisor(target); !errors.Is(e, ErrRuntimeRecoveryRequired) {
		t.Fatal("failed replay became serving runtime", e)
	}
	if e = ReplayEventLog(source, nil); e == nil {
		t.Fatal("replay allowed on journal-attached live runtime")
	}
}

func TestGateGRejectsCorruptMixedJournal(t *testing.T) {
	rt, ids := nativeFixture(t)
	wal := openKnowledgeJournal(t)
	rt.SetEventLog(wal)
	gateGSensor(t, rt, "sensor")
	cfg := nativeControlConfig(ids)
	nativeApply(t, rt, NativeControlCommand{ID: "init", Initialize: &cfg})
	raw, e := os.ReadFile(wal.file.Name())
	if e != nil {
		t.Fatal(e)
	}
	for name, data := range map[string][]byte{"truncated": raw[:len(raw)-2], "garbage": append(append([]byte(nil), raw...), []byte("{corrupt\n")...)} {
		p := filepath.Join(t.TempDir(), name+".jsonl")
		if e = os.WriteFile(p, data, 0600); e != nil {
			t.Fatal(e)
		}
		if _, e = ReadEventLog(p); e == nil {
			t.Fatal("corrupt mixed journal accepted", name)
		}
	}
	events, e := ReadEventLog(wal.file.Name())
	if e != nil {
		t.Fatal(e)
	}
	encoded, _ := json.Marshal(events[len(events)-1])
	t.Logf("last durable native record %s", encoded)
}

func TestGateGPolicyOwnershipAndHistoricalNativeVersions(t *testing.T) {
	for _, version := range []string{k.NativeControlVersion, k.NativeControlRelationalVersion, k.NativeControlCausalVersion} {
		t.Run(version, func(t *testing.T) {
			rt, ids := nativeFixture(t)
			wal := openKnowledgeJournal(t)
			rt.SetEventLog(wal)
			foundation := gateGFoundation(t, rt)
			cfg := nativeControlConfig(ids)
			cfg.Version = version
			if version != k.NativeControlVersion {
				cfg.Encoder = "local-relational-v1"
			}
			nativeApply(t, rt, NativeControlCommand{ID: "init", Initialize: &cfg})
			parameters := func() []byte {
				s := rt.brain.NativeControlSnapshot()
				values := []any{}
				for _, from := range s.ParameterSources() {
					for _, targets := range s.Outputs {
						for _, to := range targets {
							values = append(values, rt.brain.Registry.GetByID(from).Synapses[to])
						}
					}
				}
				b, e := json.Marshal(values)
				if e != nil {
					t.Fatal(e)
				}
				return b
			}
			before := parameters()
			world := mustGateGBrain(t, rt)
			gateGSensor(t, rt, "sensor-a")
			gateGText(t, rt, "written")
			gateGInquiry(t, rt, "inquiry")
			rt.SetClock(NewFixedClock(gateGTime(rt)))
			if _, e := rt.CognitiveThink(1); e != nil {
				t.Fatal(e)
			}
			if !bytes.Equal(before, parameters()) {
				t.Fatal("world dynamics overwrote native parameter ownership")
			}
			if bytes.Equal(world, mustGateGBrain(t, rt)) {
				t.Fatal("world learning was disabled")
			}
			s := rt.brain.NativeControlSnapshot()
			bad := s.Outputs[0][0]
			original := continuationBytes(t, rt)
			if _, _, e := rt.Process(Event{ID: "inject", Timestamp: gateGTime(rt), StimulusNodeIDs: []k.NodeID{bad}, Cycles: 1}); e == nil {
				t.Fatal("external policy-node injection accepted")
			}
			if e := rt.RegisterActionBinding(ActionBinding{RequestID: "bad-binding", TargetNodeIDs: []k.NodeID{bad}}); e == nil {
				t.Fatal("physical action bound to an internal native effector")
			}
			if !bytes.Equal(original, continuationBytes(t, rt)) {
				t.Fatal("invalid input changed canonical state")
			}
			gateGNative(t, rt, "native-learning")
			if bytes.Equal(before, parameters()) {
				t.Fatal("native credit failed to update its canonical parameters")
			}
			cp := filepath.Join(t.TempDir(), "complete.json")
			if e := rt.Checkpoint(cp); e != nil {
				t.Fatal(e)
			}
			recovered, e := RebuildRuntimeFromCheckpointAndJournal(wal, cp)
			if e != nil {
				t.Fatal(e)
			}
			gateGCompare(t, rt, recovered)
			genesis, e := RebuildRuntimeFromCommittedJournal(wal, foundation)
			if e != nil {
				t.Fatal(e)
			}
			gateGCompare(t, rt, genesis)
		})
	}
}

func TestGateGAdmissionMatrix(t *testing.T) {
	allowed := []string{EventTypeProcess, EventTypeCognitiveProcess, EventTypeObservation, EventTypeThink, EventTypeCognitiveThink, EventTypeSensorIntent, EventTypeLearn, EventTypeSpellingExposure, EventTypeCompositionExposure, EventTypeCompositionPerception, EventTypeVectorGround, EventTypeKnowledgeImport, EventTypeInquiryOutcome, EventTypeOutcome, EventTypeConsequence, EventTypeActionBinding}
	isolated := []string{EventTypeCognitiveModel, EventTypeCompositionMigration, EventTypeWrittenGroundingMigration, EventTypeSurfaceRebinding, "unrecognized-command"}
	rt, ids := nativeFixture(t)
	for _, kind := range append(append([]string{}, allowed...), isolated...) {
		if e := rt.admitCommandLocked(kind); e != nil {
			t.Fatal("uninitialized behavior changed", kind, e)
		}
	}
	cfg := nativeControlConfig(ids)
	nativeApply(t, rt, NativeControlCommand{ID: "init", Initialize: &cfg})
	for _, kind := range allowed {
		if e := rt.admitCommandLocked(kind); e != nil {
			t.Fatal("idle safe command blocked", kind, e)
		}
	}
	for _, kind := range isolated {
		if e := rt.admitCommandLocked(kind); !errors.Is(e, ErrCommandAdmission) {
			t.Fatal("unsafe coexistence admitted", kind, e)
		}
	}
	rt.markRecoveryRequiredLocked()
	for _, kind := range append(allowed, EventTypeNativeControl) {
		if e := rt.admitCommandLocked(kind); !errors.Is(e, ErrRuntimeRecoveryRequired) {
			t.Fatal("poisoned runtime admitted", kind, e)
		}
	}
}

func TestGateGAlreadyEnabledCompositionSharesNativeJournal(t *testing.T) {
	rt, wal, initial := newCompositionTestRuntime(t, 37, "random", true)
	train, _ := bindCases()
	lesson := nativeCompositionExposure(bindEpisodeFor(train[0], 37, 0, train[0].Words), "composition-before", compositionTestBase.Add(time.Millisecond))
	if _, e := rt.LearnComposition(lesson, lesson.After.ObservedAt); e != nil {
		t.Fatal(e)
	}
	gateGSensor(t, rt, "numeric-sensor")
	ids := []k.NodeID{rt.lastObservationPopulation[0]}
	cfg := nativeControlConfig(ids)
	cfg.Version = k.NativeControlCausalVersion
	cfg.Encoder = "local-relational-v1"
	nativeApply(t, rt, NativeControlCommand{ID: "init", Initialize: &cfg})
	gateGNative(t, rt, "native")
	lesson.ID, lesson.EpisodeID, lesson.CausalGroup = "composition-after", "composition-after", "composition-after"
	lesson.Input.Scene.ObservedAt = gateGTime(rt)
	lesson.After.ObservedAt = lesson.Input.Scene.ObservedAt.Add(time.Nanosecond)
	if _, e := rt.LearnComposition(lesson, lesson.After.ObservedAt); e != nil {
		t.Fatal(e)
	}
	cp := filepath.Join(t.TempDir(), "composed-native.json")
	if e := rt.Checkpoint(cp); e != nil {
		t.Fatal(e)
	}
	for _, p := range []string{initial, cp} {
		recovered, e := RebuildRuntimeFromCheckpointAndJournal(wal, p)
		if e != nil {
			t.Fatal(e)
		}
		gateGCompare(t, rt, recovered)
	}
}
