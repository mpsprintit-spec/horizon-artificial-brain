package runtime

import (
	"bytes"
	"encoding/json"
	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func coordinatedRuntimeBatch(t *testing.T, rt *BrainRuntime) k.CoordinatedBatchV5 {
	t.Helper()
	return coordinatedRuntimeBatchWithMode(t, rt, "recurrent")
}

func coordinatedRuntimeBatchWithMode(t *testing.T, rt *BrainRuntime, mode string) k.CoordinatedBatchV5 {
	t.Helper()
	x := k.CognitiveInput{Features: make([]float64, 28), Mask: []bool{false, false, false, true, true, true, true, true}, Reset: true, Draw: .1, NeuralMeta: &k.NeuralMetaInput{Mode: mode, Costs: []float64{.025, .03, .035}, Remaining: 3}}
	x.Features[0] = 1
	tr := []k.CognitiveInput{x}
	p, e := rt.PreviewCoordinatedV5(tr, k.Phase4DropV5{})
	if e != nil {
		t.Fatal(e)
	}
	ep := k.CoordinatedEpisodeV5{Traces: [][]k.CognitiveInput{tr}, Frames: []k.Phase4FrameV5{{Features: p.Features, Mask: x.Mask, Action: p.Action, Reward: 1, Draw: x.Draw, BehaviorProbability: p.Policy[p.Action], BehaviorValue: p.Value}}}
	b := k.CoordinatedBatchV5{Episodes: []k.CoordinatedEpisodeV5{ep}, ParametersHash: p.ParametersHash}
	b.Evidence = k.CognitiveEvidence{ID: "actual-terminal", Source: "unit", Verifier: "actual selected result", Digest: k.CognitiveDigest(b.Episodes)}
	return b
}
func TestCoordinatedV5Schema19Schema13ReplayGuardsAndRestart(t *testing.T) {
	testCoordinatedV5Replay(t, "recurrent")
}

func TestCoordinatedV5ExplorationJournalReplayAndRestart(t *testing.T) {
	testCoordinatedV5Replay(t, k.CoordinatedExploreV5)
}

func testCoordinatedV5Replay(t *testing.T, mode string) {
	t.Helper()
	rt, wal, base := phase4Fixture(t)
	spec := k.Phase4SpecV5{Version: k.Phase4VersionV5, Seed: 101, LearningRate: .003, EntropyWeight: .08, ValueWeight: .2}
	at := time.Unix(2200400000, 0)
	if _, e := rt.ApplyCognitiveModel(CognitiveModelCommand{ID: "head", Phase4Init: &spec}, at); e != nil {
		t.Fatal(e)
	}
	b := coordinatedRuntimeBatchWithMode(t, rt, mode)
	cmd := CognitiveModelCommand{ID: "learn-coordinated", Coordinated: &b}
	result, e := rt.ApplyCognitiveModel(cmd, at.Add(time.Second))
	if e != nil {
		t.Fatal(e)
	}
	if result.Coordinated == nil || result.Coordinated.MUSEGradientNorm == 0 {
		t.Fatal("no MUSE outcome update")
	}
	dup, e := rt.ApplyCognitiveModel(cmd, at.Add(time.Second))
	if e != nil || !dup.Duplicate {
		t.Fatal("retry not idempotent", e)
	}
	events, e := ReadEventLog(wal.file.Name())
	if e != nil {
		t.Fatal(e)
	}
	last := events[len(events)-1]
	if last.SchemaVersion != 13 || last.CognitiveModel.Coordinated.Episodes[0].Traces[0][0].NeuralMeta.Mode != mode {
		t.Fatal("incorrect journal schema")
	}
	last.SchemaVersion = 11
	if validateJournalPayload(last) == nil {
		t.Fatal("journal downgrade accepted")
	}
	bad := coordinatedRuntimeBatch(t, rt)
	bad.Episodes = append([]k.CoordinatedEpisodeV5(nil), b.Episodes...)
	bad.Episodes[0].Frames = append([]k.Phase4FrameV5(nil), b.Episodes[0].Frames...)
	bad.Episodes[0].Frames[0].Reward = .99
	bad.Evidence.Digest = k.CognitiveDigest(bad.Episodes)
	if _, e = rt.ApplyCognitiveModel(CognitiveModelCommand{ID: "invalid", Coordinated: &bad}, at); e == nil || rt.RecoveryRequired() {
		t.Fatal("invalid batch accepted or poisoned runtime")
	}
	p := filepath.Join(filepath.Dir(wal.file.Name()), "coordinated.json")
	if e = rt.Checkpoint(p); e != nil {
		t.Fatal(e)
	}
	var cp BrainSnapshot
	data, _ := os.ReadFile(p)
	if e = json.Unmarshal(data, &cp); e != nil || cp.SchemaVersion != 19 {
		t.Fatal("snapshot schema", e, cp.SchemaVersion)
	}
	restored := NewBrainRuntime(k.NewBrain())
	if e = restored.RestoreCheckpoint(p); e != nil {
		t.Fatal(e)
	}
	replay, e := RebuildRuntimeFromCheckpointAndJournal(wal, base)
	if e != nil {
		t.Fatal(e)
	}
	want, _ := rt.brain.CanonicalJSON()
	for _, other := range []*BrainRuntime{restored, replay} {
		got, _ := other.brain.CanonicalJSON()
		if !bytes.Equal(want, got) {
			t.Fatal("native coordinated recovery differs")
		}
	}
	cp.SchemaVersion = 18
	data, _ = json.Marshal(cp)
	os.WriteFile(p, data, 0600)
	if restored.RestoreCheckpoint(p) == nil {
		t.Fatal("checkpoint downgrade accepted")
	}
}

func TestCoordinatedV5RejectsUninitializedAndWrongModelWithoutMutation(t *testing.T) {
	r := NewBrainRuntime(nil)
	smInit(t, r)
	before, _ := r.brain.CanonicalHash()
	b := k.CoordinatedBatchV5{}
	if _, e := r.ApplyCognitiveModel(CognitiveModelCommand{ID: "wrong-model", Coordinated: &b}, time.Unix(2200400000, 0)); e == nil || r.RecoveryRequired() {
		t.Fatal("wrong model accepted or poisoned")
	}
	after, _ := r.brain.CanonicalHash()
	if before != after {
		t.Fatal("invalid command mutated state")
	}
}
