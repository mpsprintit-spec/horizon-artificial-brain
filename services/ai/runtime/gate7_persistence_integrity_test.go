package runtime

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGate7EventLogRejectsSequenceDiscontinuity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	log, err := OpenEventLog(path)
	if err != nil {
		t.Fatalf("open event log: %v", err)
	}
	defer log.Close()

	stamp := time.Date(2026, 9, 15, 10, 0, 0, 0, time.UTC)
	for _, sequence := range []uint64{1, 3} {
		if err := log.Append(LoggedEvent{
			SchemaVersion: EventLogSchemaVersion,
			BrainIdentity: BrainIdentity,
			Sequence:      sequence,
			Type:          EventTypeThink,
			Timestamp:     stamp,
			Event:         &Event{Cycles: 1, Timestamp: stamp},
		}); err != nil {
			t.Fatalf("append sequence %d: %v", sequence, err)
		}
	}
	if _, err := ReadEventLog(path); err == nil || !strings.Contains(err.Error(), "sequence discontinuity") {
		t.Fatalf("expected sequence discontinuity, got %v", err)
	}
}

func TestGate7EventLogRejectsCorruptedRecord(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	if err := os.WriteFile(path, []byte("{\"schema_version\":1,\"brain_identity\":\"horizon-primary-brain\",\"sequence\":1\n"), 0600); err != nil {
		t.Fatalf("write corrupted log: %v", err)
	}
	if _, err := ReadEventLog(path); err == nil {
		t.Fatal("corrupted event record was accepted")
	}
}

func TestGate7CheckpointRejectsTampering(t *testing.T) {
	clock := NewFixedClock(time.Date(2026, 9, 15, 11, 0, 0, 0, time.UTC))
	r := NewBrainRuntime(nil)
	r.SetClock(clock)
	if _, err := r.LearnExperience(testExperience(), clock.Now()); err != nil {
		t.Fatalf("learn: %v", err)
	}

	path := filepath.Join(t.TempDir(), "checkpoint.json")
	if err := r.Checkpoint(path); err != nil {
		t.Fatalf("checkpoint: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read checkpoint: %v", err)
	}
	var snapshot BrainSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatalf("decode checkpoint: %v", err)
	}
	if len(snapshot.Brain) == 0 {
		t.Fatal("checkpoint contains no brain state")
	}
	snapshot.Brain = append([]byte(nil), snapshot.Brain...)
	snapshot.Brain[len(snapshot.Brain)-1] ^= 1
	tampered, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("encode tampered checkpoint: %v", err)
	}
	if err := os.WriteFile(path, tampered, 0600); err != nil {
		t.Fatalf("write tampered checkpoint: %v", err)
	}

	restored := NewBrainRuntime(nil)
	if err := restored.RestoreCheckpoint(path); err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("expected checksum mismatch, got %v", err)
	}
}

func TestGate7CheckpointEnvelopeIsVersionedAndIdentified(t *testing.T) {
	r := NewBrainRuntime(nil)
	r.SetClock(NewFixedClock(time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)))
	path := filepath.Join(t.TempDir(), "checkpoint.json")
	if err := r.Checkpoint(path); err != nil {
		t.Fatalf("checkpoint: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read checkpoint: %v", err)
	}
	var snapshot BrainSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatalf("decode checkpoint: %v", err)
	}
	if snapshot.SchemaVersion != BrainSnapshotSchemaVersion {
		t.Fatalf("schema version: got %d want %d", snapshot.SchemaVersion, BrainSnapshotSchemaVersion)
	}
	if snapshot.BrainIdentity != BrainIdentity {
		t.Fatalf("brain identity: got %q want %q", snapshot.BrainIdentity, BrainIdentity)
	}
	if snapshot.Checksum == "" {
		t.Fatal("checkpoint checksum is empty")
	}
}
