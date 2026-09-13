package runtime

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const BrainSnapshotSchemaVersion = 1

type BrainSnapshot struct {
	SchemaVersion int             `json:"schema_version"`
	BrainIdentity string          `json:"brain_identity"`
	Sequence      uint64          `json:"sequence"`
	Timestamp     time.Time       `json:"timestamp"`
	Checksum      string          `json:"checksum"`
	Brain         json.RawMessage `json:"brain"`
}

// Checkpoint persists the current neural substrate together with the runtime
// sequence. The temporary file + fsync + rename sequence makes replacement
// atomic from the perspective of readers and avoids exposing partial snapshots.
func (r *BrainRuntime) Checkpoint(path string) error {
	if r == nil || r.brain == nil {
		return errors.New("brain runtime is not initialized")
	}
	if path == "" {
		return errors.New("checkpoint path is empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	brainPath := path + ".brain.tmp"
	if err := r.brain.Save(brainPath); err != nil {
		return fmt.Errorf("save brain: %w", err)
	}
	brainBytes, err := os.ReadFile(brainPath)
	if err != nil {
		_ = os.Remove(brainPath)
		return fmt.Errorf("read brain snapshot: %w", err)
	}
	_ = os.Remove(brainPath)

	checksum := sha256.Sum256(brainBytes)
	snapshot := BrainSnapshot{
		SchemaVersion: BrainSnapshotSchemaVersion,
		BrainIdentity: BrainIdentity,
		Sequence:      r.seq,
		Timestamp:     time.Now().UTC(),
		Checksum:      hex.EncodeToString(checksum[:]),
		Brain:         json.RawMessage(brainBytes),
	}
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("encode checkpoint: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create checkpoint directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".horizon-checkpoint-*.tmp")
	if err != nil {
		return fmt.Errorf("create checkpoint temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write checkpoint: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync checkpoint: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close checkpoint: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("commit checkpoint: %w", err)
	}
	return nil
}

// RestoreCheckpoint loads a versioned checkpoint into the runtime's single
// existing Brain. It never creates a second neural substrate.
func (r *BrainRuntime) RestoreCheckpoint(path string) error {
	if r == nil || r.brain == nil {
		return errors.New("brain runtime is not initialized")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var snapshot BrainSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return fmt.Errorf("decode checkpoint: %w", err)
	}
	if snapshot.SchemaVersion != BrainSnapshotSchemaVersion {
		return fmt.Errorf("unsupported checkpoint schema version %d", snapshot.SchemaVersion)
	}
	if snapshot.BrainIdentity != BrainIdentity {
		return fmt.Errorf("checkpoint belongs to brain %q", snapshot.BrainIdentity)
	}
	checksum := sha256.Sum256(snapshot.Brain)
	if hex.EncodeToString(checksum[:]) != snapshot.Checksum {
		return errors.New("checkpoint checksum mismatch")
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	brainPath := path + ".restore.tmp"
	if err := os.WriteFile(brainPath, snapshot.Brain, 0600); err != nil {
		return fmt.Errorf("stage brain restore: %w", err)
	}
	defer os.Remove(brainPath)
	if err := r.brain.Load(brainPath); err != nil {
		return fmt.Errorf("restore brain: %w", err)
	}
	r.seq = snapshot.Sequence
	return nil
}
