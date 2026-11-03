package runtime

import (
	"bytes"
	"encoding/json"
	"fmt"
	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"
)

func nativeConfig() k.CognitiveModelConfig {
	return k.CognitiveModelConfig{Version: k.CognitiveModelV2, Inputs: 12, Hidden: 4, Actions: 4, Horizon: 3, Seed: 619, LearningRate: .003, Gamma: 1, ValueWeight: .2, EntropyWeight: .01, AssessmentWeight: .2, NeuralMeta: &k.NeuralMetaConfig{Classes: 2, Computations: 1, Outcomes: 2, ContextOffset: 0, ContextSize: 1, ResultOffset: 1, OperationOffset: 3, ActionOffset: 4, HypothesisOffset: 8, PlanMarker: 10, StepIndex: 11, JointWeight: .7, RewardWeight: .3, PlanDepth: 1}}
}
func nativeInput() k.CognitiveInput {
	x := k.CognitiveInput{Features: make([]float64, 12), Mask: []bool{true, true, true, true}, Draw: .5, Reset: true, NeuralMeta: &k.NeuralMetaInput{Mode: "recurrent", Costs: []float64{.05}, Remaining: 2}}
	x.Features[0] = 1
	return x
}
func nativeBatch(t testing.TB, rt *BrainRuntime, id string) k.CognitiveTrainingBatch {
	t.Helper()
	x := nativeInput()
	x.Mask[0] = false
	p, e := rt.PreviewCognitiveModel(x)
	if e != nil {
		t.Fatal(e)
	}
	goal := 0
	if p.Action == 2 {
		goal = 1
	}
	success := -1
	reward := .35
	if p.Action != 3 {
		success = 0
		reward = 1
	}
	eps := []k.CognitiveTrainingEpisode{{VerifiedTarget: &goal, SuccessAt: success, Frames: []k.CognitiveTrainingFrame{{Features: x.Features, Mask: x.Mask, Draw: x.Draw, Behaviour: p.Policy, Action: p.Action, Reward: reward, NeuralMeta: x.NeuralMeta}}}}
	return k.CognitiveTrainingBatch{Episodes: eps, ParametersHash: p.ParametersHash, LearnPolicy: true, Evidence: k.CognitiveEvidence{ID: id, Source: "controlled-native-test", Verifier: "terminal-verified", Digest: k.CognitiveDigest(eps)}}
}
func TestNativeMetaV2CheckpointLearningCalibrationReplay(t *testing.T) {
	dir := t.TempDir()
	log, e := OpenEventLog(filepath.Join(dir, "events.jsonl"))
	if e != nil {
		t.Fatal(e)
	}
	defer log.Close()
	rt := NewBrainRuntime(nil)
	brain := rt.brain
	rt.SetEventLog(log)
	rt.SetClock(NewFixedClock(time.Unix(1800000000, 0)))
	cfg := nativeConfig()
	if _, e = rt.ApplyCognitiveModel(CognitiveModelCommand{ID: "native-init", Initialize: &cfg}, time.Unix(1800000000, 0)); e != nil {
		t.Fatal(e)
	}
	if rt.brain != brain || rt.activation.Memory != brain || rt.learning.Kb != brain || rt.learning.Memory.Kb != brain {
		t.Fatal("more than one substrate")
	}
	mid := filepath.Join(dir, "middle.json")
	if e = rt.Checkpoint(mid); e != nil {
		t.Fatal(e)
	}
	for i := 0; i < 3; i++ {
		b := nativeBatch(t, rt, fmt.Sprint("train-", i))
		if _, e = rt.ApplyCognitiveModel(CognitiveModelCommand{ID: fmt.Sprint("learn-", i), Training: &b}, time.Unix(1800000010+int64(i), 0)); e != nil {
			t.Fatal(e)
		}
	}
	beforeWeights, _ := rt.CognitiveModelParameters()
	b := nativeBatch(t, rt, "calibration")
	b.LearnPolicy = false
	cmd := CognitiveModelCommand{ID: "calibrate", Calibration: &b}
	applied, e := rt.ApplyCognitiveModel(cmd, time.Unix(1800000020, 0))
	if e != nil {
		t.Fatal(e)
	}
	again, e := rt.ApplyCognitiveModel(cmd, time.Unix(1800000021, 0))
	if e != nil || !again.Duplicate || again.Sequence != applied.Sequence {
		t.Fatal("calibration is not exactly once")
	}
	afterWeights, _ := rt.CognitiveModelParameters()
	if !reflect.DeepEqual(beforeWeights, afterWeights) {
		t.Fatal("temperature calibration changed recurrent or head weights")
	}
	x := nativeInput()
	x.NeuralMeta.Mode = "muse-bmps"
	if _, e = rt.ApplyCognitiveModel(CognitiveModelCommand{ID: "native-input", Input: &x}, time.Unix(1800000030, 0)); e != nil {
		t.Fatal(e)
	}
	final := filepath.Join(dir, "final.json")
	if e = rt.Checkpoint(final); e != nil {
		t.Fatal(e)
	}
	restored := NewBrainRuntime(nil)
	if e = restored.RestoreCheckpoint(final); e != nil {
		t.Fatal(e)
	}
	suffix, e := RebuildRuntimeFromCheckpointAndJournal(log, mid)
	if e != nil {
		t.Fatal(e)
	}
	events, e := ReadEventLog(filepath.Join(dir, "events.jsonl"))
	if e != nil {
		t.Fatal(e)
	}
	full := NewBrainRuntime(nil)
	if e = ReplayEventLog(full, events); e != nil {
		t.Fatal(e)
	}
	want, _ := rt.brain.CanonicalJSON()
	for name, other := range map[string]*BrainRuntime{"restore": restored, "full": full, "suffix": suffix} {
		got, _ := other.brain.CanonicalJSON()
		if !bytes.Equal(want, got) {
			t.Fatalf("%s canonical state differs", name)
		}
		a, _ := rt.PreviewCognitiveModel(x)
		b, _ := other.PreviewCognitiveModel(x)
		if !reflect.DeepEqual(a, b) {
			t.Fatalf("%s next neural action differs", name)
		}
	}
	data, _ := os.ReadFile(final)
	var snap BrainSnapshot
	if e = json.Unmarshal(data, &snap); e != nil {
		t.Fatal(e)
	}
	if snap.SchemaVersion != 16 {
		t.Fatal("native snapshot is not schema 16")
	}
	snap.SchemaVersion = 15
	data, _ = json.Marshal(snap)
	bad := filepath.Join(dir, "downgrade.json")
	os.WriteFile(bad, data, 0600)
	if NewBrainRuntime(nil).RestoreCheckpoint(bad) == nil {
		t.Fatal("silent schema downgrade accepted")
	}
	old := events[0]
	old.SchemaVersion = 5
	if validateJournalPayload(old) == nil {
		t.Fatal("V2 payload silently accepted under V1 journal schema")
	}
	t.Logf("one owning Brain, canonical schema 5/checkpoint16/journal6; %d replayed records byte-identical", len(events))
}
func TestNativeMetaV2ConcurrentReadWrite(t *testing.T) {
	rt := NewBrainRuntime(nil)
	cfg := nativeConfig()
	if _, e := rt.ApplyCognitiveModel(CognitiveModelCommand{ID: "init", Initialize: &cfg}, time.Unix(1800000000, 0)); e != nil {
		t.Fatal(e)
	}
	var wg sync.WaitGroup
	for worker := 0; worker < 5; worker++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < 20; i++ {
				x := nativeInput()
				if w == 0 {
					_, e := rt.ApplyCognitiveModel(CognitiveModelCommand{ID: fmt.Sprintf("input-%d", i), Input: &x}, time.Unix(1800000010+int64(i), 0))
					if e != nil {
						t.Error(e)
					}
				} else if _, e := rt.PreviewCognitiveModel(x); e != nil {
					t.Error(e)
				}
			}
		}(worker)
	}
	wg.Wait()
}
func TestNativeMetaV2ReplacedJournalPathFailsBeforeMutation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.jsonl")
	log, e := OpenEventLog(path)
	if e != nil {
		t.Fatal(e)
	}
	defer log.Close()
	rt := NewBrainRuntime(nil)
	rt.SetEventLog(log)
	cfg := nativeConfig()
	if _, e = rt.ApplyCognitiveModel(CognitiveModelCommand{ID: "init", Initialize: &cfg}, time.Unix(1800000000, 0)); e != nil {
		t.Fatal(e)
	}
	before, _ := rt.brain.CanonicalHash()
	if e = os.Rename(path, filepath.Join(dir, "original-inode.jsonl")); e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(path, nil, 0600); e != nil {
		t.Fatal(e)
	}
	x := nativeInput()
	if _, e = rt.ApplyCognitiveModel(CognitiveModelCommand{ID: "after-replace", Input: &x}, time.Unix(1800000001, 0)); e == nil || !rt.RecoveryRequired() {
		t.Fatal("replacement did not stop journal writer")
	}
	after, _ := rt.brain.CanonicalHash()
	if before != after {
		t.Fatal("replacement caused unlogged neural mutation")
	}
}

func BenchmarkNativeMetaV2Cycle(b *testing.B) {
	for _, mode := range []string{"recurrent", "muse", "bmps", "muse-bmps"} {
		b.Run(mode, func(b *testing.B) {
			rt := NewBrainRuntime(nil)
			cfg := nativeConfig()
			rt.ApplyCognitiveModel(CognitiveModelCommand{ID: "init", Initialize: &cfg}, time.Unix(1800000000, 0))
			x := nativeInput()
			x.NeuralMeta.Mode = mode
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, e := rt.PreviewCognitiveModel(x); e != nil {
					b.Fatal(e)
				}
			}
		})
	}
}

// Opt-in full-size benchmark uses a COPY of the actually trained native
// checkpoint, after the multi-seed cohort has stopped consuming CPU.
func BenchmarkNativeMetaV2TrainedCanonical(b *testing.B) {
	source := os.Getenv("HORIZON_NATIVE_BENCH_CHECKPOINT")
	if source == "" {
		b.Skip("explicit trained internal checkpoint required")
	}
	for _, mode := range []string{"recurrent", "muse", "bmps", "muse-bmps"} {
		b.Run(mode, func(b *testing.B) {
			data, e := os.ReadFile(source)
			if e != nil {
				b.Fatal(e)
			}
			path := filepath.Join(b.TempDir(), "trained.json")
			if e = os.WriteFile(path, data, 0600); e != nil {
				b.Fatal(e)
			}
			rt := NewBrainRuntime(nil)
			if e = rt.RestoreCheckpoint(path); e != nil {
				b.Fatal(e)
			}
			x := k.CognitiveInput{Features: make([]float64, 28), Mask: []bool{true, true, true, true, true, true, true, true}, Draw: .5, Reset: true, NeuralMeta: &k.NeuralMetaInput{Mode: mode, Costs: []float64{.025, .03, .035}, Remaining: 3, Greedy: true}}
			x.Features[0] = 1
			x.Features[25] = .25
			x.Features[26] = .3
			x.Features[27] = .35
			if _, e = rt.PreviewCognitiveModel(x); e != nil {
				b.Fatal(e)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, e = rt.PreviewCognitiveModel(x); e != nil {
					b.Fatal(e)
				}
			}
		})
	}
}
