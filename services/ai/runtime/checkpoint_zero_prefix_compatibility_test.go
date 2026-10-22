package runtime

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestZeroPrefixFrozenPR44CheckpointAndNextThought(t *testing.T) {
	dir := filepath.Join("testdata", "zero_journal_prefix_v1")
	var reference struct {
		Producer string    `json:"producer_commit"`
		Zero     string    `json:"zero_checkpoint_sha256"`
		Events   string    `json:"events_sha256"`
		Before   string    `json:"before_thought_checksum"`
		After    string    `json:"after_thought_checksum"`
		Output   string    `json:"next_thought_sha256"`
		NextAt   time.Time `json:"next_thought_at"`
		Sequence uint64    `json:"recovered_sequence"`
	}
	data, err := os.ReadFile(filepath.Join(dir, "reference.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &reference); err != nil {
		t.Fatal(err)
	}
	if reference.Producer != "56afe5dd449eeebc6cb69e0e13561589fb54002f" || reference.Sequence != 2 {
		t.Fatal("fixture is not the pre-fix PR #44 checkpoint")
	}
	digest := func(data []byte) string {
		sum := sha256.Sum256(data)
		return hex.EncodeToString(sum[:])
	}
	local := t.TempDir()
	for name, expected := range map[string]string{"zero-checkpoint.json": reference.Zero, "events.jsonl": reference.Events} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil || digest(data) != expected {
			t.Fatalf("frozen %s changed: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(local, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(local, "zero-checkpoint.json")
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	fresh := NewBrainRuntime(knowledge.NewBrain())
	fresh.SetClock(NewFixedClock(time.Unix(100, 0).UTC()))
	fresh.SetEventLog(openKnowledgeJournal(t))
	newPath := filepath.Join(t.TempDir(), "new-zero.json")
	if err := fresh.Checkpoint(newPath); err != nil {
		t.Fatal(err)
	}
	newBytes, err := os.ReadFile(newPath)
	if err != nil || !bytes.Equal(original, newBytes) {
		t.Fatal("new writer changed the original zero checkpoint bytes or checksum")
	}
	snapshot, err := readCheckpointForTest(path)
	if err != nil || snapshot.Sequence != 0 || snapshot.JournalPrefixHash != emptyJournalDigest {
		t.Fatalf("bad historical zero checkpoint: %v", err)
	}
	log, err := OpenEventLog(filepath.Join(local, "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	rebuilt, err := RebuildRuntimeFromCheckpointAndJournal(log, path)
	if err != nil {
		t.Fatalf("original checksum-bound checkpoint must recover without rewriting it: %v", err)
	}
	if rebuilt.LastSequence() != reference.Sequence || rebuilt.eventLog != nil {
		t.Fatal("historical suffix did not recover exactly once, detached")
	}
	stateChecksum := func() string {
		t.Helper()
		path := filepath.Join(t.TempDir(), "detached-state.json")
		if err := rebuilt.Checkpoint(path); err != nil {
			t.Fatal(err)
		}
		s, err := readCheckpointForTest(path)
		if err != nil {
			t.Fatal(err)
		}
		return s.Checksum
	}
	if got := stateChecksum(); got != reference.Before {
		t.Fatalf("historical continuation differs: %s", got)
	}
	rebuilt.SetClock(NewFixedClock(reference.NextAt))
	out, err := rebuilt.CognitiveThink(1)
	if err != nil {
		t.Fatal(err)
	}
	data, err = json.Marshal(out)
	if err != nil || digest(data) != reference.Output || stateChecksum() != reference.After {
		t.Fatalf("historical next thought or complete resulting state changed: %v", err)
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(original, after) {
		t.Fatal("historical checkpoint was rewritten")
	}
}
