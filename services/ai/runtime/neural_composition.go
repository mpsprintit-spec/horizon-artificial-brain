package runtime

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

const (
	EventTypeCompositionMigration         = "composition_migration"
	EventTypeCompositionExposure          = "composition_exposure"
	compositionJournalSchemaVersion       = 4
	compositionBrainSnapshotSchemaVersion = 14
)

var ErrCompositionRequiresWrittenMigration = errors.New("composition requires the explicit written-neural-v2 migration first")

// Only configuration and current perceptual/recurrent continuation live here.
// There are no trainable weights, lexical maps or semantic answers. Learned
// parameters are exclusively the canonical Brain's dynamic synapses.
type CompositionContinuation struct {
	Config      knowledge.CompositionConfig `json:"config"`
	Scene       *knowledge.CompositionScene `json:"scene,omitempty"`
	Words       [][]float64                 `json:"words,omitempty"`
	Unsupported bool                        `json:"unsupported,omitempty"`
}

type CompositionPlan struct {
	ReferenceRevision uint64                      `json:"reference_revision"`
	ReferenceHash     string                      `json:"reference_hash"`
	Config            knowledge.CompositionConfig `json:"config"`
}

type CompositionMigration struct {
	Plan              CompositionPlan `json:"plan"`
	BackupHash        string          `json:"backup_hash"`
	BackupJournalHash string          `json:"backup_journal_hash,omitempty"`
}

func freezeComposition[T any](in T) (T, error) {
	var out T
	data, err := json.Marshal(in)
	if err == nil {
		err = json.Unmarshal(data, &out)
	}
	return out, err
}

func cloneCompositionState(in *CompositionContinuation) *CompositionContinuation {
	if in == nil {
		return nil
	}
	out, _ := freezeComposition(*in)
	return &out
}

func (s *CompositionContinuation) validate() error {
	if s == nil {
		return nil
	}
	if err := s.Config.Validate(); err != nil {
		return err
	}
	if s.Scene == nil {
		return knowledge.ValidateCompositionWords(s.Words)
	}
	return (knowledge.CompositionInput{Words: s.Words, Scene: *s.Scene}).Validate()
}

func (m CompositionMigration) validateShape() error {
	if !isSHA256(m.Plan.ReferenceHash) || !isSHA256(m.BackupHash) || m.BackupJournalHash != "" && !isSHA256(m.BackupJournalHash) {
		return errors.New("composition migration requires a verified state reference and backup digests")
	}
	return m.Plan.Config.Validate()
}

func (r *BrainRuntime) PlanNeuralComposition(c knowledge.CompositionConfig) (CompositionPlan, error) {
	if r == nil || r.brain == nil {
		return CompositionPlan{}, errors.New("runtime is unavailable")
	}
	if err := c.Validate(); err != nil {
		return CompositionPlan{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ensureWritableLocked(); err != nil {
		return CompositionPlan{}, err
	}
	if !r.neuralWrittenGrounding {
		return CompositionPlan{}, ErrCompositionRequiresWrittenMigration
	}
	if r.composition != nil || r.brain.HasNeuralComposition() {
		return CompositionPlan{}, errors.New("composition is already initialized")
	}
	hash, err := r.surfaceReferenceLocked()
	return CompositionPlan{r.seq, hash, c}, err
}

func (r *BrainRuntime) validateCompositionPlanLocked(p CompositionPlan) error {
	if err := p.Config.Validate(); err != nil {
		return err
	}
	if !r.neuralWrittenGrounding || r.composition != nil || r.brain.HasNeuralComposition() {
		return errors.New("composition requires a migrated, not-yet-initialized Brain")
	}
	if p.ReferenceRevision != r.seq {
		return errors.New("composition reference revision changed")
	}
	hash, err := r.surfaceReferenceLocked()
	if err != nil {
		return err
	}
	if hash != p.ReferenceHash {
		return errors.New("composition reference state changed")
	}
	return nil
}

// ApplyNeuralComposition saves a schema-13 rollback pair before committing
// the new version. It never replaces the user's Brain with an experiment file.
func (r *BrainRuntime) ApplyNeuralComposition(plan CompositionPlan, backupPath string, at time.Time) error {
	if r == nil || r.brain == nil {
		return errors.New("runtime is unavailable")
	}
	if strings.TrimSpace(backupPath) == "" {
		return errors.New("composition requires a new rollback backup path")
	}
	frozen, err := freezeComposition(plan)
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ensureWritableLocked(); err != nil {
		return err
	}
	if err := r.validateCompositionPlanLocked(frozen); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(backupPath), 0755); err != nil {
		return err
	}
	for _, p := range []string{backupPath, backupPath + ".events.jsonl"} {
		if p != backupPath && r.eventLog == nil {
			continue
		}
		f, err := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			return fmt.Errorf("reserve composition rollback copy: %w", err)
		}
		if err := f.Close(); err != nil {
			return err
		}
	}
	if err := r.checkpointVersionLocked(backupPath, neuralWrittenBrainSnapshotSchemaVersion); err != nil {
		return err
	}
	journalHash := ""
	if r.eventLog != nil {
		journalHash, err = r.backupMigrationJournalLocked(backupPath + ".events.jsonl")
		if err != nil {
			return err
		}
	}
	directory, err := os.Open(filepath.Dir(backupPath))
	if err != nil {
		return err
	}
	syncErr := directory.Sync()
	closeErr := directory.Close()
	if syncErr != nil {
		return syncErr
	}
	if closeErr != nil {
		return closeErr
	}
	file, err := os.Open(backupPath)
	if err != nil {
		return err
	}
	hash := sha256.New()
	_, copyErr := io.Copy(hash, file)
	closeFileErr := file.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeFileErr != nil {
		return closeFileErr
	}
	m := CompositionMigration{Plan: frozen, BackupHash: "sha256:" + hex.EncodeToString(hash.Sum(nil)), BackupJournalHash: journalHash}
	if at.IsZero() {
		at = r.nowLocked()
	}
	return r.applyCompositionMigrationLocked(m, at.UTC())
}

func (r *BrainRuntime) applyCompositionMigrationLocked(m CompositionMigration, at time.Time) error {
	if err := m.validateShape(); err != nil {
		return err
	}
	if err := r.validateCompositionPlanLocked(m.Plan); err != nil {
		return err
	}
	return r.runJournalCommandLocked(LoggedEvent{Type: EventTypeCompositionMigration, Timestamp: at, CompositionMigration: &m}, func() error {
		if err := r.brain.InitializeCompositionAt(m.Plan.Config, at); err != nil {
			return err
		}
		r.composition = &CompositionContinuation{Config: m.Plan.Config}
		r.seq++
		r.inspectionEpoch = ""
		return nil
	})
}

// LearnComposition is one redo command / revision, including all grounding,
// supervision and controller updates. Demonstration provenance is retained in
// the journal; repetition is not counted as independent outcome evidence.
func (r *BrainRuntime) LearnComposition(e knowledge.CompositionExposure, at time.Time) (uint64, error) {
	if r == nil || r.brain == nil {
		return 0, errors.New("runtime is unavailable")
	}
	if err := e.Validate(); err != nil {
		return 0, err
	}
	frozen, err := freezeComposition(e)
	if err != nil {
		return 0, err
	}
	if err := frozen.Validate(); err != nil {
		return 0, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ensureWritableLocked(); err != nil {
		return r.seq, err
	}
	if r.composition == nil {
		return r.seq, errors.New("neural composition is disabled")
	}
	if at.IsZero() {
		at = r.nowLocked()
	}
	at = at.UTC()
	if frozen.After.ObservedAt.After(at) {
		return r.seq, errors.New("composition cannot learn an unobserved future outcome")
	}
	if _, exists := r.processedEvents[frozen.ID]; exists {
		return r.seq, fmt.Errorf("duplicate event ID %q", frozen.ID)
	}
	if err := r.brain.ValidateCompositionExposure(r.composition.Config, frozen); err != nil {
		return r.seq, err
	}
	err = r.runJournalCommandLocked(LoggedEvent{Type: EventTypeCompositionExposure, Timestamp: at, CompositionExposure: &frozen}, func() error {
		if err := r.brain.LearnCompositionAt(r.composition.Config, frozen, at); err != nil {
			return err
		}
		r.seq++
		r.processedEvents[frozen.ID] = r.seq
		// Keep the most recent actual physical observation; never regress a
		// live scene when a delayed teaching outcome arrives.
		if r.composition.Scene == nil || !frozen.After.ObservedAt.Before(r.composition.Scene.ObservedAt) {
			scene, _ := freezeComposition(frozen.After)
			r.composition.Scene = &scene
		}
		return nil
	})
	return r.seq, err
}

// ReadComposition is a detached numerical preview. It does not ground,
// initialize, learn, authorize or execute anything.
func (r *BrainRuntime) ReadComposition(input knowledge.CompositionInput, at time.Time) (knowledge.CompositionReadout, error) {
	if r == nil || r.brain == nil {
		return knowledge.CompositionReadout{}, errors.New("runtime is unavailable")
	}
	if err := input.Validate(); err != nil {
		return knowledge.CompositionReadout{}, err
	}
	frozen, err := freezeComposition(input)
	if err != nil {
		return knowledge.CompositionReadout{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ensureWritableLocked(); err != nil {
		return knowledge.CompositionReadout{}, err
	}
	if r.composition == nil {
		return knowledge.CompositionReadout{Status: "disabled"}, nil
	}
	if at.IsZero() {
		at = r.nowLocked()
	}
	out, err := r.brain.ReadComposition(r.composition.Config, frozen, at)
	out.Sequence = r.seq
	out.BrainIdentity = BrainIdentity
	return out, err
}

func (r *BrainRuntime) CompositionStatus() (*CompositionContinuation, error) {
	if r == nil || r.brain == nil {
		return nil, errors.New("runtime is unavailable")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ensureWritableLocked(); err != nil {
		return nil, err
	}
	return cloneCompositionState(r.composition), nil
}

func (r *BrainRuntime) compositionReadoutLocked(at time.Time) (*knowledge.CompositionReadout, error) {
	if r.composition == nil {
		return nil, nil
	}
	if r.composition.Unsupported {
		return &knowledge.CompositionReadout{Status: "position-window-exceeded", Sequence: r.seq, BrainIdentity: BrainIdentity}, nil
	}
	if r.composition.Scene == nil {
		return &knowledge.CompositionReadout{Status: "missing-scene", Sequence: r.seq, BrainIdentity: BrainIdentity}, nil
	}
	out, err := r.brain.ReadComposition(r.composition.Config, knowledge.CompositionInput{Words: r.composition.Words, Scene: *r.composition.Scene}, at)
	out.Sequence = r.seq
	out.BrainIdentity = BrainIdentity
	return &out, err
}

func (r *BrainRuntime) compositionContextLocked(event *Event) (*knowledge.CompositionReadout, error) {
	readout, err := r.compositionReadoutLocked(event.Timestamp)
	if err != nil || readout == nil {
		return readout, err
	}
	context, err := r.brain.CompositionActivation(*readout)
	if err != nil {
		return nil, err
	}
	if len(context) > 0 && event.Context == nil {
		event.Context = map[knowledge.NodeID]float64{}
	}
	for id, v := range context {
		event.Context[id] = max(event.Context[id], v)
	}
	return readout, nil
}

func (r *BrainRuntime) decorateCompositionJournalLocked(e *LoggedEvent) {
	if r.composition != nil {
		e.CompositionEncoding = r.composition.Config.Version
	}
	if e.CompositionEncoding != "" || e.Type == EventTypeCompositionMigration || e.Type == EventTypeCompositionExposure {
		e.SchemaVersion = compositionJournalSchemaVersion
	}
}

// A typed simulator adapter into the existing sensor intent/commit boundary.
func (r *BrainRuntime) ObserveCompositionScene(id, source string, scene knowledge.CompositionScene) (CognitiveInterpretation, error) {
	if err := scene.Validate(); err != nil {
		return CognitiveInterpretation{}, err
	}
	fields := make([]SensorField, len(scene.Patches))
	for i, p := range scene.Patches {
		fields[i] = SensorField{FieldID: fmt.Sprintf("patch-%d", i), GroupID: fmt.Sprintf("patch-%d", i), Kind: "physical-patch-v1", Values: p.Values, Coordinates: p.XY}
	}
	return NewCognitiveOrchestrator(r).ProcessSensorObservation(SensorObservation{Modality: "simulator", Source: source, SampleID: id, Timestamp: scene.ObservedAt, Fields: fields, Confidence: 1, CompositionScene: &scene})
}
