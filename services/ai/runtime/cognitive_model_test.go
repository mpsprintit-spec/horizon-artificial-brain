package runtime

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"
)

func smConfig() knowledge.CognitiveModelConfig {
	return knowledge.CognitiveModelConfig{Version: knowledge.CognitiveModelVersion, Inputs: 3, Hidden: 3, Actions: 3, Horizon: 4, Seed: 91, LearningRate: .003, Gamma: 1, ValueWeight: .2, EntropyWeight: .01, AssessmentWeight: .3, WorldWeight: .2, RegulationSteps: 2, RegulationRate: .02}
}
func smInit(t testing.TB, r *BrainRuntime) {
	t.Helper()
	c := smConfig()
	if _, e := r.ApplyCognitiveModel(CognitiveModelCommand{ID: "sm-init", Initialize: &c}, time.Unix(1800000000, 0)); e != nil {
		t.Fatal(e)
	}
}
func smInput() knowledge.CognitiveInput {
	return knowledge.CognitiveInput{Features: []float64{.1, -.6, .2}, Mask: []bool{true, true, true}, Draw: .3}
}
func TestSelfModelSingleBrainOwnership(t *testing.T) {
	r := NewBrainRuntime(nil)
	brain := r.brain
	smInit(t, r)
	if brain != r.brain || r.activation.Memory != brain || r.learning.Kb != brain || r.learning.Memory.Kb != brain {
		t.Fatal("self model created or rebound another Brain")
	}
	if r.brain.BrainState.CognitiveModel == nil || len(r.brain.BrainState.CognitiveModel.Layout.Hidden) != 3 {
		t.Fatal("no canonical recurrent N")
	}
	if NewBrainRuntime(brain) != nil {
		t.Fatal("same Brain bound to two runtimes")
	}
}
func TestSelfModelCheckpointFullAndSuffixReplay(t *testing.T) {
	dir := t.TempDir()
	log, err := OpenEventLog(filepath.Join(dir, "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	r := NewBrainRuntime(nil)
	r.SetClock(NewFixedClock(time.Unix(1800000000, 0)))
	r.SetEventLog(log)
	smInit(t, r)
	mid := filepath.Join(dir, "middle.json")
	if err = r.Checkpoint(mid); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 4; i++ {
		x := smInput()
		x.Reset = i%2 == 0
		x.Features[0] = float64(i) / 5
		if _, err = r.ApplyCognitiveModel(CognitiveModelCommand{ID: fmt.Sprintf("step-%d", i), Input: &x}, time.Unix(1800000000+int64(i), 0)); err != nil {
			t.Fatal(err)
		}
	}
	x := smInput()
	x.Reset = true
	p, err := r.PreviewCognitiveModel(x)
	if err != nil {
		t.Fatal(err)
	}
	episodes := []knowledge.CognitiveTrainingEpisode{{SuccessAt: 0, Frames: []knowledge.CognitiveTrainingFrame{{Features: x.Features, Mask: x.Mask, Draw: x.Draw, Action: p.Action, Behaviour: p.Policy, Reward: 1}}}}
	batch := knowledge.CognitiveTrainingBatch{Episodes: episodes, ParametersHash: p.ParametersHash, LearnPolicy: true, Evidence: knowledge.CognitiveEvidence{ID: "outcome", Source: "test-simulator", Verifier: "actual-result", Digest: knowledge.CognitiveDigest(episodes)}}
	if _, err = r.ApplyCognitiveModel(CognitiveModelCommand{ID: "learn", Training: &batch}, time.Unix(1800000010, 0)); err != nil {
		t.Fatal(err)
	}
	final := filepath.Join(dir, "final.json")
	if err = r.Checkpoint(final); err != nil {
		t.Fatal(err)
	}
	restored := NewBrainRuntime(nil)
	if err = restored.RestoreCheckpoint(final); err != nil {
		t.Fatal(err)
	}
	suffix, err := RebuildRuntimeFromCheckpointAndJournal(log, mid)
	if err != nil {
		t.Fatal(err)
	}
	events, err := ReadEventLog(filepath.Join(dir, "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	full := NewBrainRuntime(nil)
	if err = ReplayEventLog(full, events); err != nil {
		t.Fatal(err)
	}
	want, _ := r.brain.CanonicalJSON()
	for name, other := range map[string]*BrainRuntime{"restore": restored, "suffix": suffix, "full": full} {
		got, _ := other.brain.CanonicalJSON()
		if !bytes.Equal(want, got) {
			t.Fatalf("%s canonical state differs", name)
		}
		a, _ := r.PreviewCognitiveModel(smInput())
		b, _ := other.PreviewCognitiveModel(smInput())
		if !reflect.DeepEqual(a, b) {
			t.Fatalf("%s next operation differs", name)
		}
	}
	var snapshot BrainSnapshot
	data, _ := os.ReadFile(final)
	json.Unmarshal(data, &snapshot)
	if snapshot.SchemaVersion != 15 {
		t.Fatalf("unsafe checkpoint version %d", snapshot.SchemaVersion)
	}
	snapshot.SchemaVersion = 14
	data, _ = json.Marshal(snapshot)
	os.WriteFile(filepath.Join(dir, "downgraded.json"), data, 0600)
	if NewBrainRuntime(nil).RestoreCheckpoint(filepath.Join(dir, "downgraded.json")) == nil {
		t.Fatal("downgraded cognitive checkpoint accepted")
	}
}
func TestSelfModelInvalidAndDuplicateCommands(t *testing.T) {
	r := NewBrainRuntime(nil)
	smInit(t, r)
	before, _ := r.brain.CanonicalHash()
	x := smInput()
	x.Features[0] = 2
	if _, e := r.ApplyCognitiveModel(CognitiveModelCommand{ID: "bad", Input: &x}, time.Unix(1800000001, 0)); e == nil {
		t.Fatal("invalid input accepted")
	}
	after, _ := r.brain.CanonicalHash()
	if before != after || r.RecoveryRequired() {
		t.Fatal("invalid command mutated or poisoned Brain")
	}
	x = smInput()
	command := CognitiveModelCommand{ID: "once", Input: &x}
	a, e := r.ApplyCognitiveModel(command, time.Unix(1800000002, 0))
	if e != nil {
		t.Fatal(e)
	}
	b, e := r.ApplyCognitiveModel(command, time.Unix(1800000003, 0))
	if e != nil || !b.Duplicate || a.Sequence != b.Sequence {
		t.Fatal("duplicate applied twice")
	}
	command.Input.Features[0] = .9
	if _, e = r.ApplyCognitiveModel(command, time.Unix(1800000004, 0)); e == nil {
		t.Fatal("ID reused for altered payload")
	}
}
func TestSelfModelConcurrentPreviewAndLearning(t *testing.T) {
	r := NewBrainRuntime(nil)
	smInit(t, r)
	var wg sync.WaitGroup
	for worker := 0; worker < 4; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for i := 0; i < 25; i++ {
				x := smInput()
				if worker == 0 {
					_, e := r.ApplyCognitiveModel(CognitiveModelCommand{ID: fmt.Sprintf("parallel-%d", i), Input: &x}, time.Unix(1800000010+int64(i), 0))
					if e != nil {
						t.Error(e)
					}
				} else if _, e := r.PreviewCognitiveModel(x); e != nil {
					t.Error(e)
				}
			}
		}(worker)
	}
	wg.Wait()
}
