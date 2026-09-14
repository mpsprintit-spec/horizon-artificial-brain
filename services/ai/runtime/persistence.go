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

	"github.com/project-horizon/horizon-core/services/ai/activation"
)

const BrainSnapshotSchemaVersion = 2

type BrainSnapshot struct {
	SchemaVersion int                      `json:"schema_version"`
	BrainIdentity string                   `json:"brain_identity"`
	Sequence      uint64                   `json:"sequence"`
	Timestamp     time.Time                `json:"timestamp"`
	Checksum      string                   `json:"checksum"`
	Brain         json.RawMessage          `json:"brain"`
	Activation    activation.StateSnapshot `json:"activation"`
}

type checkpointPayload struct {
	Brain      json.RawMessage          `json:"brain"`
	Activation activation.StateSnapshot `json:"activation"`
}

// checkpointChecksum covers the complete logical neural/runtime state rather
// than only the serialized Brain graph. Envelope metadata is intentionally
// excluded so a checkpoint can be verified independently of its timestamp or
// sequence metadata.
func checkpointChecksum(brain json.RawMessage, state activation.StateSnapshot) (string, error) {
	payload, err := json.Marshal(checkpointPayload{Brain: brain, Activation: state})
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(payload)
	return hex.EncodeToString(hash[:]), nil
}

// Checkpoint persists the neural substrate and the runtime's recurrent
// activation state. The latter is process state, not a second memory store;
// without it a restart would restore the graph but lose the ongoing thought.
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

	activationState := r.activation.SnapshotState()
	checksum, err := checkpointChecksum(brainBytes, activationState)
	if err != nil {
		return fmt.Errorf("checksum checkpoint: %w", err)
	}
	snapshot := BrainSnapshot{
		SchemaVersion: BrainSnapshotSchemaVersion,
		BrainIdentity: BrainIdentity,
		Sequence:      r.seq,
		Timestamp:     r.now(),
		Checksum:      checksum,
		Brain:         json.RawMessage(brainBytes),
		Activation:    activationState,
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
// existing Brain and restores its recurrent activation state. It never
// creates a second neural substrate.
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
	checksum, err := checkpointChecksum(snapshot.Brain, snapshot.Activation)
	if err != nil {
		return fmt.Errorf("checksum checkpoint: %w", err)
	}
	if checksum != snapshot.Checksum {
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
	r.activation.RestoreState(snapshot.Activation)
	r.seq = snapshot.Sequence
	return nil
}
