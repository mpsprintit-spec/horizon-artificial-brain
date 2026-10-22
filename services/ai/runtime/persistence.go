package runtime

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/activation"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

const physicalMemoryBrainSnapshotSchemaVersion = 10
const observedFeedbackBrainSnapshotSchemaVersion = 11
const evidenceProvenanceBrainSnapshotSchemaVersion = 12
const neuralWrittenBrainSnapshotSchemaVersion = 13
const BrainSnapshotSchemaVersion = neuralWrittenBrainSnapshotSchemaVersion
const recurrentBrainSnapshotSchemaVersion = 9
const orderedBrainSnapshotSchemaVersion = 8
const outcomeBrainSnapshotSchemaVersion = 7
const neuralBrainSnapshotSchemaVersion = 6
const priorBrainSnapshotSchemaVersion = 5
const legacyBrainSnapshotSchemaVersion = 4

type BrainSnapshot struct {
	NeuralWrittenGrounding bool `json:"neural_written_grounding,omitempty"`
	ActionBindings map[string]ActionBinding         `json:"action_bindings,omitempty"`
	EvidenceLedger *learning.EvidenceLedgerSnapshot `json:"evidence_ledger,omitempty"`
	SchemaVersion  int                              `json:"schema_version"`
	BrainIdentity  string                           `json:"brain_identity"`
	Sequence       uint64                           `json:"sequence"`
	Timestamp      time.Time                        `json:"timestamp"`
	Checksum       string                           `json:"checksum"`
	Brain          json.RawMessage                  `json:"brain"`
	Activation     activation.StateSnapshot         `json:"activation"`
	CognitiveState CognitiveState                   `json:"cognitive_state"`
	InquiryValence map[InquiryAction]float64        `json:"inquiry_valence"`
	// Snapshot v5: temporal predecessor and dedup state must survive a restart.
	LastObservationPopulation []knowledge.NodeID `json:"last_observation_population,omitempty"`
	LastObservationAt         time.Time          `json:"last_observation_at,omitempty"`
	ProcessedEvents           map[string]uint64  `json:"processed_events,omitempty"`
	// Adapter-only provenance, stored in the checkpoint envelope rather than
	// the lexical-identity-free canonical Brain substrate.
	SurfaceAnnotations []knowledge.SurfaceAnnotation `json:"surface_annotations,omitempty"`
	// Empty for standalone checkpoints. Required for journal-offset recovery.
	JournalPrefixHash string `json:"journal_prefix_hash,omitempty"`
}

type checkpointPayload struct {
	Brain          json.RawMessage           `json:"brain"`
	Activation     activation.StateSnapshot  `json:"activation"`
	CognitiveState CognitiveState            `json:"cognitive_state"`
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
	if err := r.ensureWritableLocked(); err != nil {
		return err
	}
	return r.checkpointLocked(path)
}

// checkpointLocked is also used to persist a private pre-migration backup
// while the same owner lock protects its reference and journal receipt.
func (r *BrainRuntime) checkpointLocked(path string) error {
	return r.checkpointVersionLocked(path, BrainSnapshotSchemaVersion)
}

func (r *BrainRuntime) checkpointVersionLocked(path string, version int) error {
	// Checkpoints must persist the ENTIRE canonical substrate, including
	// developmental experiences, bootstrap episodes, learned policy and
	// neural dynamics. The legacy nodes/patterns-only snapshot silently lost
	// these states and made checkpoint-offset replay diverge from live memory.

	activationState := r.activation.SnapshotState()
	cognitiveState := cloneCognitiveState(r.lastCognitiveState)
	inquiryValence := cloneInquiryValence(r.inquiryValence)
	// Journal anchoring is evaluated under the runtime lock, so no event may
	// advance r.seq between the snapshot and its durable prefix digest.
	journalHash, err := r.checkpointJournalPrefixLocked()
	if err != nil {
		return fmt.Errorf("anchor checkpoint to committed journal: %w", err)
	}
	ledger := r.evidence.Snapshot()
	snapshot := BrainSnapshot{
		NeuralWrittenGrounding:    r.neuralWrittenGrounding,
		ActionBindings:            cloneActionBindings(r.actions),
		EvidenceLedger:            &ledger,
		SchemaVersion:             version,
		BrainIdentity:             BrainIdentity,
		Sequence:                  r.seq,
		Timestamp:                 r.nowLocked(),
		Activation:                activationState,
		CognitiveState:            cognitiveState,
		InquiryValence:            inquiryValence,
		LastObservationPopulation: append([]knowledge.NodeID(nil), r.lastObservationPopulation...),
		LastObservationAt:         r.lastObservationAt,
		ProcessedEvents:           cloneProcessedEventIndex(r.processedEvents),
		SurfaceAnnotations:        r.brain.SurfaceAnnotationsSnapshot(),
		JournalPrefixHash:         journalHash,
	}
	return r.writeCheckpointLocked(path, snapshot)
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
	if snapshot.SchemaVersion != BrainSnapshotSchemaVersion && snapshot.SchemaVersion != evidenceProvenanceBrainSnapshotSchemaVersion && snapshot.SchemaVersion != observedFeedbackBrainSnapshotSchemaVersion && snapshot.SchemaVersion != physicalMemoryBrainSnapshotSchemaVersion && snapshot.SchemaVersion != recurrentBrainSnapshotSchemaVersion && snapshot.SchemaVersion != orderedBrainSnapshotSchemaVersion && snapshot.SchemaVersion != outcomeBrainSnapshotSchemaVersion && snapshot.SchemaVersion != neuralBrainSnapshotSchemaVersion && snapshot.SchemaVersion != priorBrainSnapshotSchemaVersion && snapshot.SchemaVersion != legacyBrainSnapshotSchemaVersion {
		return fmt.Errorf("unsupported checkpoint schema version %d", snapshot.SchemaVersion)
	}
	if snapshot.SchemaVersion < neuralWrittenBrainSnapshotSchemaVersion && snapshot.NeuralWrittenGrounding {
		return errors.New("numeric written serving contract requires checkpoint schema 13")
	}
	for _, annotation := range snapshot.SurfaceAnnotations {
		if err := annotation.Encoding.Validate(); err != nil {
			return fmt.Errorf("checkpoint adapter contract: %w", err)
		}
	}
	if snapshot.BrainIdentity != BrainIdentity {
		return fmt.Errorf("checkpoint belongs to brain %q", snapshot.BrainIdentity)
	}
	if snapshot.SchemaVersion < neuralBrainSnapshotSchemaVersion && len(snapshot.SurfaceAnnotations) > 0 {
		return errors.New("legacy checkpoint cannot carry unauthenticated adapter annotations")
	}
	if snapshot.SchemaVersion < outcomeBrainSnapshotSchemaVersion && (len(snapshot.ActionBindings) > 0 || snapshot.EvidenceLedger != nil) {
		return errors.New("legacy checkpoint cannot carry unauthenticated outcome state")
	}
	var ledger *learning.EvidenceLedger
	if snapshot.SchemaVersion >= outcomeBrainSnapshotSchemaVersion {
		if snapshot.EvidenceLedger == nil {
			return errors.New("checkpoint is missing outcome continuation state")
		}
		if snapshot.SchemaVersion < evidenceProvenanceBrainSnapshotSchemaVersion && len(snapshot.EvidenceLedger.Provenance) > 0 {
			return errors.New("typed evidence provenance requires checkpoint schema 12")
		}
	} else {
		ledger = learning.NewEvidenceLedger()
	}
	checksum, err := checkpointChecksumForSchema(snapshot)
	if err != nil {
		return fmt.Errorf("checksum checkpoint: %w", err)
	}
	if checksum != snapshot.Checksum {
		return errors.New("checkpoint checksum mismatch")
	}
	if snapshot.EvidenceLedger != nil {
		ledger, err = learning.RestoreEvidenceLedger(*snapshot.EvidenceLedger)
		if err != nil {
			return err
		}
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

	if err := knowledge.ValidateOrderedState(snapshot.Brain, snapshot.Activation.LastTrace, snapshot.SchemaVersion >= orderedBrainSnapshotSchemaVersion); err != nil {
		return fmt.Errorf("checkpoint neural contract: %w", err)
	}

	if err := knowledge.ValidateMemoryDecayState(snapshot.Brain, snapshot.SchemaVersion >= physicalMemoryBrainSnapshotSchemaVersion); err != nil {
		return fmt.Errorf("checkpoint memory contract: %w", err)
	}
	if err := knowledge.ValidateWrittenNumericState(snapshot.Brain, snapshot.NeuralWrittenGrounding); err != nil {
		return fmt.Errorf("checkpoint written grounding contract: %w", err)
	}
	if err := validateObservationPrediction(snapshot); err != nil {
		return err
	}
	if err := validateCheckpointContext(snapshot); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	// A poisoned runtime may have mutated neural state or runtime caches that
	// a checkpoint restore cannot safely unwind. Always use a new runtime.
	if err := r.ensureWritableLocked(); err != nil {
		return err
	}
	if r.eventLog != nil {
		return errors.New("checkpoint restore requires a detached runtime; use supervisor recovery")
	}
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
	actions := make(map[string]ActionBinding, len(snapshot.ActionBindings))
	for id, binding := range snapshot.ActionBindings {
		validated, bindingErr := r.validateActionBindingLocked(binding)
		if bindingErr != nil || id != binding.RequestID {
			r.markRecoveryRequiredLocked()
			return fmt.Errorf("%w: invalid checkpoint action binding %q", ErrRuntimeRecoveryRequired, id)
		}
		actions[id] = validated
	}
	if snapshot.EvidenceLedger != nil {
		for id := range snapshot.EvidenceLedger.ByNode {
			if r.brain.Registry.GetByID(id) == nil {
				r.markRecoveryRequiredLocked()
				return fmt.Errorf("%w: evidence targets an unknown node", ErrRuntimeRecoveryRequired)
			}
		}
	}
	r.inspectionEpoch = ""
	r.neuralWrittenGrounding = snapshot.NeuralWrittenGrounding
	if r.neuralWrittenGrounding { r.surfaceEncoding = knowledge.NeuralWrittenEncoding } else { r.surfaceEncoding = knowledge.WrittenTextEncoding }
	r.actions, r.evidence = actions, ledger
	r.activation.RestoreState(snapshot.Activation)
	r.lastCognitiveState = cloneCognitiveState(snapshot.CognitiveState)
	r.inquiryValence = cloneInquiryValence(snapshot.InquiryValence)
	r.seq = snapshot.Sequence
	r.lastObservationPopulation = append([]knowledge.NodeID(nil), snapshot.LastObservationPopulation...)
	r.lastObservationAt = snapshot.LastObservationAt
	r.processedEvents = cloneProcessedEventIndex(snapshot.ProcessedEvents)
	return nil
}

func checkpointChecksumForSchema(snapshot BrainSnapshot) (string, error) {
	switch snapshot.SchemaVersion {
	case neuralWrittenBrainSnapshotSchemaVersion:
		return checkpointChecksumV8(snapshot)
	case legacyBrainSnapshotSchemaVersion:
		return checkpointChecksum(snapshot.Brain, snapshot.Activation, snapshot.CognitiveState, snapshot.InquiryValence)
	case priorBrainSnapshotSchemaVersion:
		return checkpointChecksumV5(snapshot)
	case neuralBrainSnapshotSchemaVersion:
		return checkpointChecksumV6(snapshot)
	default:
		return checkpointChecksumV7(snapshot)
	}
}

func cloneInquiryValence(in map[InquiryAction]float64) map[InquiryAction]float64 {
	out := make(map[InquiryAction]float64, len(in))
	for action, value := range in {
		out[action] = clampSigned(value)
	}
	return out
}
