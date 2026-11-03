package runtime

import (
	"bytes"
	"fmt"
	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
	"path/filepath"
	"testing"
	"time"
)

func TestV4ComponentJournalReplayAndDowngrade(t *testing.T) {
	dir := t.TempDir()
	wal, e := OpenEventLog(filepath.Join(dir, "events.jsonl"))
	if e != nil {
		t.Fatal(e)
	}
	defer wal.Close()
	rt := NewBrainRuntime(nil)
	rt.SetEventLog(wal)
	cfg := nativeConfig()
	if _, e = rt.ApplyCognitiveModel(CognitiveModelCommand{ID: "init", Initialize: &cfg}, time.Unix(1800000000, 0)); e != nil {
		t.Fatal(e)
	}
	mid := filepath.Join(dir, "mid.json")
	if e = rt.Checkpoint(mid); e != nil {
		t.Fatal(e)
	}
	for i, component := range []string{"world-decoder", "actor-critic", "actor-only"} {
		b := nativeBatch(t, rt, component)
		b.ComponentV4 = component
		b.LearnPolicy = component != "world-decoder"
		v := .1
		if b.LearnPolicy {
			b.EntropyWeightV4 = &v
		}
		if _, e = rt.ApplyCognitiveModel(CognitiveModelCommand{ID: fmt.Sprint("v4-", i), Training: &b}, time.Unix(1800000001+int64(i), 0)); e != nil {
			t.Fatal(e)
		}
	}
	events, e := ReadEventLog(filepath.Join(dir, "events.jsonl"))
	if e != nil {
		t.Fatal(e)
	}
	for _, ev := range events[1:] {
		if ev.SchemaVersion != 8 {
			t.Fatal("missing schema8")
		}
		for old := 5; old < 8; old++ {
			bad := ev
			bad.SchemaVersion = old
			if validateJournalPayload(bad) == nil {
				t.Fatal("accepted downgrade", old)
			}
		}
	}
	suffix, e := RebuildRuntimeFromCheckpointAndJournal(wal, mid)
	if e != nil {
		t.Fatal(e)
	}
	a, _ := rt.brain.CanonicalJSON()
	b, _ := suffix.brain.CanonicalJSON()
	if !bytes.Equal(a, b) {
		t.Fatal("suffix mismatch")
	}
	full, e := RebuildRuntimeFromCommittedJournal(wal, k.NewBootstrapBrain)
	if e != nil {
		t.Fatal(e)
	}
	b, _ = full.brain.CanonicalJSON()
	if !bytes.Equal(a, b) {
		t.Fatal("full replay mismatch")
	}
	cp := filepath.Join(dir, "final.json")
	if e = rt.Checkpoint(cp); e != nil {
		t.Fatal(e)
	}
	rest := NewBrainRuntime(nil)
	if e = rest.RestoreCheckpoint(cp); e != nil {
		t.Fatal(e)
	}
	b, _ = rest.brain.CanonicalJSON()
	if !bytes.Equal(a, b) {
		t.Fatal("checkpoint restart mismatch")
	}
	t.Log("one canonical Brain; schema8 downgrade rejection; full/suffix replay and checkpoint restart identical")
}
