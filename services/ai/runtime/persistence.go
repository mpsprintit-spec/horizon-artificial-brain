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
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

const BrainSnapshotSchemaVersion = 6
const priorBrainSnapshotSchemaVersion = 5
const legacyBrainSnapshotSchemaVersion = 4

type BrainSnapshot struct {
	SchemaVersion int                      `json:"schema_version"`
	BrainIdentity string                   `json:"brain_identity"`
	Sequence      uint64                   `json:"sequence"`
	Timestamp     time.Time                `json:"timestamp"`
	Checksum      string                   `json:"checksum"`
	Brain         json.RawMessage          `json:"brain"`
	Activation    activation.StateSnapshot `json:"activation"`
	CognitiveState CognitiveState `json:"cognitive_state"`
	InquiryValence map[InquiryAction]float64 `json:"inquiry_valence"`
	// Snapshot v5: temporal predecessor and dedup state must survive a restart.
	LastObservationPopulation []knowledge.NodeID `json:"last_observation_population,omitempty"`
	LastObservationAt time.Time `json:"last_observation_at,omitempty"`
	ProcessedEvents map[string]uint64 `json:"processed_events,omitempty"`
	// Adapter-only provenance, stored in the checkpoint envelope rather than
	// the lexical-identity-free canonical Brain substrate.
	SurfaceAnnotations []knowledge.SurfaceAnnotation `json:"surface_annotations,omitempty"`
	// Empty for standalone checkpoints. Required for journal-offset recovery.
	JournalPrefixHash string `json:"journal_prefix_hash,omitempty"`
}

type checkpointPayload struct {
	Brain          json.RawMessage          `json:"brain"`
	Activation     activation.StateSnapshot `json:"activation"`
	CognitiveState CognitiveState           `json:"cognitive_state"`
	InquiryValence map[InquiryAction]float64 `json:"inquiry_valence"`
}

// checkpointChecksum covers the complete logical neural/runtime state rather
// than only the serialized Brain graph. Envelope metadata is intentionally
// excluded so a checkpoint can be verified independently of its timestamp or
// sequence metadata.
func checkpointChecksum(brain json.RawMessage, state activation.StateSnapshot, cognitiveState CognitiveState, inquiryValence map[InquiryAction]float64) (string, error) {
	payload, err := json.Marshal(checkpointPayload{Brain: brain, Activation: state, CognitiveState: cognitiveState, InquiryValence: inquiryValence})
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
	if err := r.ensureWritableLocked(); err != nil { return err }

	// Checkpoint v5 must persist the ENTIRE canonical substrate, including
	// developmental experiences, bootstrap episodes, learned policy and
	// neural dynamics. The legacy nodes/patterns-only snapshot silently lost
	// these states and made checkpoint-offset replay diverge from live memory.
	brainBytes, err := r.brain.CanonicalJSON()
	if err != nil {
		return fmt.Errorf("serialize complete canonical brain: %w", err)
	}

	activationState := r.activation.SnapshotState()
	cognitiveState := cloneCognitiveState(r.lastCognitiveState)
	inquiryValence := cloneInquiryValence(r.inquiryValence)
	// Journal anchoring is evaluated under the runtime lock, so no event may
	// advance r.seq between the snapshot and its durable prefix digest.
	journalHash, err := r.checkpointJournalPrefixLocked()
	if err != nil {
		return fmt.Errorf("anchor checkpoint to committed journal: %w", err)
	}
	snapshot := BrainSnapshot{
		SchemaVersion: BrainSnapshotSchemaVersion,
		BrainIdentity: BrainIdentity,
		Sequence:      r.seq,
		Timestamp:     r.nowLocked(),
		Brain:         json.RawMessage(brainBytes),
		Activation:     activationState,
		CognitiveState: cognitiveState,
		InquiryValence: inquiryValence,
		LastObservationPopulation: append([]knowledge.NodeID(nil), r.lastObservationPopulation...),
		LastObservationAt: r.lastObservationAt,
		ProcessedEvents: cloneProcessedEventIndex(r.processedEvents),
		SurfaceAnnotations: r.brain.SurfaceAnnotationsSnapshot(),
		JournalPrefixHash: journalHash,
	}
	snapshot.Checksum, err = checkpointChecksumV6(snapshot)
	if err != nil {
		return fmt.Errorf("checksum checkpoint: %w", err)
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
	if snapshot.SchemaVersion != BrainSnapshotSchemaVersion && snapshot.SchemaVersion != priorBrainSnapshotSchemaVersion && snapshot.SchemaVersion != legacyBrainSnapshotSchemaVersion {
		return fmt.Errorf("unsupported checkpoint schema version %d", snapshot.SchemaVersion)
	}
	if snapshot.BrainIdentity != BrainIdentity {
		return fmt.Errorf("checkpoint belongs to brain %q", snapshot.BrainIdentity)
	}
	if snapshot.SchemaVersion != BrainSnapshotSchemaVersion && len(snapshot.SurfaceAnnotations) > 0 {
		return errors.New("legacy checkpoint cannot carry unauthenticated adapter annotations")
	}
	var checksum string
	switch snapshot.SchemaVersion {
	case legacyBrainSnapshotSchemaVersion:
		checksum, err = checkpointChecksum(snapshot.Brain, snapshot.Activation, snapshot.CognitiveState, snapshot.InquiryValence)
	case priorBrainSnapshotSchemaVersion:
		checksum, err = checkpointChecksumV5(snapshot)
	default:
		checksum, err = checkpointChecksumV6(snapshot)
	}
	if err != nil {
		return fmt.Errorf("checksum checkpoint: %w", err)
	}
	if checksum != snapshot.Checksum {
		return errors.New("checkpoint checksum mismatch")
	}
	if snapshot.SchemaVersion >= priorBrainSnapshotSchemaVersion {
		if (len(snapshot.LastObservationPopulation) == 0) != snapshot.LastObservationAt.IsZero() {
			return errors.New("checkpoint temporal population and timestamp disagree")
		}
		for id, seq := range snapshot.ProcessedEvents {
			if id == "" || seq == 0 || seq > snapshot.Sequence {
				return fmt.Errorf("checkpoint processed event index invalid for %q", id)
			}
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	// A poisoned runtime may have mutated neural state or runtime caches that
	// a checkpoint restore cannot safely unwind. Always use a new runtime.
	if err := r.ensureWritableLocked(); err != nil { return err }
	brainPath := path + ".restore.tmp"
	if err := os.WriteFile(brainPath, snapshot.Brain, 0600); err != nil {
		return fmt.Errorf("stage brain restore: %w", err)
	}
	defer os.Remove(brainPath)
	if err := r.brain.Load(brainPath); err != nil {
		return fmt.Errorf("restore brain: %w", err)
	}
	if err := r.brain.RestoreSurfaceAnnotations(snapshot.SurfaceAnnotations); err != nil {
		r.markRecoveryRequiredLocked()
		return fmt.Errorf("%w: restore checkpoint adapter provenance: %v", ErrRuntimeRecoveryRequired, err)
	}
	r.activation.RestoreState(snapshot.Activation)
	r.lastCognitiveState = cloneCognitiveState(snapshot.CognitiveState)
	r.inquiryValence = cloneInquiryValence(snapshot.InquiryValence)
	r.seq = snapshot.Sequence
	r.lastObservationPopulation = append([]knowledge.NodeID(nil), snapshot.LastObservationPopulation...)
	r.lastObservationAt = snapshot.LastObservationAt
	r.processedEvents = cloneProcessedEventIndex(snapshot.ProcessedEvents)
	return nil
}


func cloneInquiryValence(in map[InquiryAction]float64) map[InquiryAction]float64 {
	out := make(map[InquiryAction]float64, len(in))
	for action, value := range in {
		out[action] = clampSigned(value)
	}
	return out
}
