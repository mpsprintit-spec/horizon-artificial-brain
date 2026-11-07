package nativefoundation

import (
	"encoding/json"
	"fmt"
	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
	r "github.com/project-horizon/horizon-core/services/ai/runtime"
	"os"
	"path/filepath"
)

// RecoveryAudit preserves a complete WAL and a checkpoint after 64 training
// episodes on an independent replica. Replay executes internal computation and
// learning only. Existing journal contracts require the complete prefix.
func RecoveryAudit(dir string) error {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	brain, err := NewFixture(dir, false)
	if err != nil {
		return err
	}
	rt := r.NewBrainRuntime(brain)
	wal, err := r.OpenEventLog(filepath.Join(dir, "events.jsonl"))
	if err != nil {
		return err
	}
	defer wal.Close()
	rt.SetEventLog(wal)
	cfg := Config(700001, 1)
	apply(rt, r.NativeControlCommand{ID: "initialize", Initialize: &cfg})
	evaluate(rt, tasks(700002, 64, "train"), true, 700003, "trained")
	before := filepath.Join(dir, "before.json")
	if err = rt.Checkpoint(before); err != nil {
		return err
	}
	evaluate(rt, tasks(80000001, 5, "test"), true, 80000002, "trained")
	after := filepath.Join(dir, "after.json")
	if err = rt.Checkpoint(after); err != nil {
		return err
	}
	recovered, err := r.RebuildRuntimeFromCheckpointAndJournal(wal, before)
	if err != nil {
		return err
	}
	// Rebuild deliberately leaves replay read-only; reattach the verified log
	// before writing a checkpoint so both sides authenticate the same prefix.
	recovered.SetEventLog(wal)
	rebuilt := filepath.Join(dir, "rebuilt.json")
	if err = recovered.Checkpoint(rebuilt); err != nil {
		return err
	}
	read := func(path string) (r.BrainSnapshot, error) {
		var s r.BrainSnapshot
		data, e := os.ReadFile(path)
		if e == nil {
			e = json.Unmarshal(data, &s)
		}
		return s, e
	}
	a, err := read(after)
	if err != nil {
		return err
	}
	b, err := read(rebuilt)
	if err != nil {
		return err
	}
	if k.NativeDigest(a.Brain) != k.NativeDigest(b.Brain) || k.NativeDigest(a.Activation) != k.NativeDigest(b.Activation) || a.Sequence != b.Sequence || a.Checksum != b.Checksum {
		return fmt.Errorf("trained recovery differs")
	}
	data, _ := json.MarshalIndent(map[string]any{"canonical_equal": true, "activation_equal": true, "checksum_equal": true, "sequence": a.Sequence, "checkpoint_schema": a.SchemaVersion, "journal_schema": 14, "physical_actions_executed": 0}, "", "  ")
	return os.WriteFile(filepath.Join(dir, "verification.json"), data, 0600)
}
