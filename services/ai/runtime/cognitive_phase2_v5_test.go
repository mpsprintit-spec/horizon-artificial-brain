package runtime

import (
	"bytes"
	"encoding/json"
	"fmt"
	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func phase2RuntimeBatch(t *testing.T, rt *BrainRuntime) k.Phase2BatchV5 {
	t.Helper()
	b := nativeBatch(t, rt, "world-decoder")
	b.ComponentV4 = ""
	b.LearnPolicy = false
	b.EntropyWeightV4 = nil
	return k.Phase2BatchV5{Batch: b}
}
func TestPhase2NativeJournalSchemaRestartReplay(t *testing.T) {
	dir := t.TempDir()
	wal, e := OpenEventLog(filepath.Join(dir, "events.jsonl"))
	if e != nil {
		t.Fatal(e)
	}
	defer wal.Close()
	brain := k.NewBrain()
	rt := NewBrainRuntime(brain)
	rt.SetEventLog(wal)
	cfg := nativeConfig()
	at := time.Unix(1900000000, 0)
	if _, e = rt.ApplyCognitiveModel(CognitiveModelCommand{ID: "init", Initialize: &cfg}, at); e != nil {
		t.Fatal(e)
	}
	if rt.brain != brain {
		t.Fatal("canonical Brain ownership changed")
	}
	mid := filepath.Join(dir, "base16.json")
	if e = rt.Checkpoint(mid); e != nil {
		t.Fatal(e)
	}
	spec := k.Phase2SpecV5{Version: k.Phase2VersionV5, Representation: "pairwise", LearnGRU: true, Optimizer: "historical-global"}
	if _, e = rt.ApplyCognitiveModel(CognitiveModelCommand{ID: "allocate", Phase2Init: &spec}, at.Add(time.Second)); e != nil {
		t.Fatal(e)
	}
	b := phase2RuntimeBatch(t, rt)
	if _, e = rt.ApplyCognitiveModel(CognitiveModelCommand{ID: "world-learn", Phase2Training: &b}, at.Add(2*time.Second)); e != nil {
		t.Fatal(e)
	}
	events, e := ReadEventLog(filepath.Join(dir, "events.jsonl"))
	if e != nil {
		t.Fatal(e)
	}
	for _, ev := range events[1:] {
		if ev.SchemaVersion != 9 {
			t.Fatal("not journal9")
		}
		bad := ev
		bad.SchemaVersion = 8
		if validateJournalPayload(bad) == nil {
			t.Fatal("journal downgrade accepted")
		}
	}
	newCP := filepath.Join(dir, "state17.json")
	if e = rt.Checkpoint(newCP); e != nil {
		t.Fatal(e)
	}
	if e = rt.checkpointVersionLocked(filepath.Join(dir, "bad16.json"), 16); e == nil {
		t.Fatal("checkpoint downgrade accepted")
	}
	suffix, e := RebuildRuntimeFromCheckpointAndJournal(wal, mid)
	if e != nil {
		t.Fatal(e)
	}
	// The original fixture explicitly owns NewBrain(), so full replay must use
	// that same initial state, rather than adding an unrelated bootstrap corpus.
	full, e := RebuildRuntimeFromCommittedJournal(wal, k.NewBrain)
	if e != nil {
		t.Fatal(e)
	}
	rest := NewBrainRuntime(nil)
	if e = rest.RestoreCheckpoint(newCP); e != nil {
		t.Fatal(e)
	}
	a, _ := rt.brain.CanonicalJSON()
	for name, other := range map[string]*BrainRuntime{"suffix": suffix, "full": full, "restart": rest} {
		b, _ := other.brain.CanonicalJSON()
		if !bytes.Equal(a, b) {
			t.Fatal("native complete state/replay differs", name)
		}
	}
	var snap BrainSnapshot
	data, e := osReadPhase2(newCP)
	if e != nil {
		t.Fatal(e)
	}
	if e = json.Unmarshal(data, &snap); e != nil {
		t.Fatal(e)
	}
	if snap.SchemaVersion != 17 {
		t.Fatal("not checkpoint17")
	}
	t.Log("canonical Brain, strict journal9, checkpoint17, old16 restore, full/suffix fixture replay, restart and downgrade rejection passed")
}
func osReadPhase2(p string) ([]byte, error) { return os.ReadFile(p) }
func TestPhase2ConcurrentTypedProbeAndNativeUpdates(t *testing.T) {
	rt := NewBrainRuntime(nil)
	cfg := nativeConfig()
	at := time.Unix(1900000000, 0)
	if _, e := rt.ApplyCognitiveModel(CognitiveModelCommand{ID: "init", Initialize: &cfg}, at); e != nil {
		t.Fatal(e)
	}
	spec := k.Phase2SpecV5{Version: k.Phase2VersionV5, Representation: "pairwise", Optimizer: "historical-global"}
	if _, e := rt.ApplyCognitiveModel(CognitiveModelCommand{ID: "allocate", Phase2Init: &spec}, at); e != nil {
		t.Fatal(e)
	}
	x := k.CognitiveInput{Features: make([]float64, cfg.Inputs), Mask: make([]bool, cfg.Actions), Reset: true, Draw: .3, NeuralMeta: &k.NeuralMetaInput{Mode: "recurrent", Costs: []float64{.1}, Remaining: 2}}
	x.Features[0] = 1
	for i := range x.Mask {
		x.Mask[i] = true
	}
	var wg sync.WaitGroup
	for j := 0; j < 4; j++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 10; i++ {
				if _, e := rt.PreviewPhase2V5(k.Phase2ProbeV5{Trace: []k.CognitiveInput{x}}); e != nil {
					t.Error(e)
				}
			}
		}()
	}
	for i := 0; i < 4; i++ {
		b := phase2RuntimeBatch(t, rt)
		if _, e := rt.ApplyCognitiveModel(CognitiveModelCommand{ID: fmt.Sprint("batch", i), Phase2Training: &b}, at); e != nil {
			t.Fatal(e)
		}
	}
	wg.Wait()
}
