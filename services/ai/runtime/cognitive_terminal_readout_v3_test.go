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

func TestV3TerminalReadoutJournalGuardAndReplay(t *testing.T) {
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
	mid := filepath.Join(dir, "middle.json")
	if e = rt.Checkpoint(mid); e != nil {
		t.Fatal(e)
	}
	for i := 0; i < 2; i++ {
		b := nativeBatch(t, rt, "verified")
		b.LearnPolicy = false
		b.TerminalReadoutOnly = true
		c := CognitiveModelCommand{ID: []string{"repair", "calibrate"}[i]}
		if i == 0 {
			c.Training = &b
		} else {
			c.Calibration = &b
		}
		if _, e = rt.ApplyCognitiveModel(c, time.Unix(1800000001+int64(i), 0)); e != nil {
			t.Fatal(e)
		}
	}
	events, e := ReadEventLog(filepath.Join(dir, "events.jsonl"))
	if e != nil {
		t.Fatal(e)
	}
	for _, ev := range events[1:] {
		if ev.SchemaVersion != 7 {
			t.Fatal("missing V3 guard")
		}
		ev.SchemaVersion = 6
		if validateJournalPayload(ev) == nil {
			t.Fatal("accepted downgraded V3 event")
		}
	}
	suffix, e := RebuildRuntimeFromCheckpointAndJournal(wal, mid)
	if e != nil {
		t.Fatal(e)
	}
	a, _ := rt.brain.CanonicalJSON()
	b, _ := suffix.brain.CanonicalJSON()
	if !bytes.Equal(a, b) {
		t.Fatal("V3 suffix replay mismatch")
	}
	full, e := RebuildRuntimeFromCommittedJournal(wal, k.NewBootstrapBrain)
	if e != nil {
		t.Fatal(e)
	}
	c, _ := full.brain.CanonicalJSON()
	if !bytes.Equal(a, c) {
		t.Fatal("V3 full replay mismatch")
	}
	cp := filepath.Join(dir, "final.json")
	if e = rt.Checkpoint(cp); e != nil {
		t.Fatal(e)
	}
	rest := NewBrainRuntime(nil)
	if e = rest.RestoreCheckpoint(cp); e != nil {
		t.Fatal(e)
	}
	d, _ := rest.brain.CanonicalJSON()
	if !bytes.Equal(a, d) {
		t.Fatal("checkpoint mismatch")
	}
	data, _ := os.ReadFile(cp)
	var header map[string]json.RawMessage
	if json.Unmarshal(data, &header) != nil {
		t.Fatal("checkpoint corrupt")
	}
	t.Log("V3 journal7, checkpoint16, full/suffix replay and restart byte identical")
}
