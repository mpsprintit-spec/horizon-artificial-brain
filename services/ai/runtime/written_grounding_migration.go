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
	"reflect"
	"strings"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

const EventTypeWrittenGroundingMigration = "written_grounding_migration"

// Plan contains detached numeric receptive fields for the existing populations.
// Its reference includes all legacy annotations; editing any source invalidates it.
type WrittenGroundingPlan struct {
	ReferenceRevision uint64                           `json:"reference_revision"`
	ReferenceHash     string                           `json:"reference_hash"`
	Projections       []knowledge.ProjectionPopulation `json:"projections"`
}

type WrittenGroundingMigration struct {
	Plan              WrittenGroundingPlan `json:"plan"`
	BackupHash        string               `json:"backup_hash"`
	BackupJournalHash string               `json:"backup_journal_hash,omitempty"`
}

func isSHA256(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != 71 {
		return false
	}
	_, err := hex.DecodeString(value[7:])
	return err == nil
}

func (m WrittenGroundingMigration) validateShape() error {
	if !isSHA256(m.Plan.ReferenceHash) || !isSHA256(m.BackupHash) {
		return errors.New("written migration requires a reference and durable backup digest")
	}
	if m.BackupJournalHash != "" && !isSHA256(m.BackupJournalHash) {
		return errors.New("invalid migration journal backup digest")
	}
	return nil
}

func (r *BrainRuntime) PlanNeuralWrittenGrounding() (WrittenGroundingPlan, error) {
	if r == nil || r.brain == nil {
		return WrittenGroundingPlan{}, errors.New("brain runtime is not initialized")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ensureWritableLocked(); err != nil {
		return WrittenGroundingPlan{}, err
	}
	if r.dynamicsEncoding.HasCanonicalPropagation() {
		return WrittenGroundingPlan{}, errors.New("written migration requires a historical propagation runtime; canonical migration is not supported")
	}
	if r.neuralWrittenGrounding {
		return WrittenGroundingPlan{}, errors.New("numeric written grounding is already enabled")
	}
	projections, err := r.brain.PlanWrittenNumericProjection()
	if err != nil {
		return WrittenGroundingPlan{}, err
	}
	reference, err := r.surfaceReferenceLocked()
	return WrittenGroundingPlan{r.seq, reference, projections}, err
}

func (r *BrainRuntime) validateWrittenGroundingPlanLocked(plan WrittenGroundingPlan) error {
	if r.dynamicsEncoding.HasCanonicalPropagation() {
		return errors.New("written migration cannot downgrade the canonical propagation backup")
	}
	if r.neuralWrittenGrounding {
		return errors.New("numeric written grounding is already enabled")
	}
	if r.surfaceEncoding != knowledge.WrittenTextEncoding {
		return errors.New("migration requires the written-text-v1 serving contract")
	}
	if plan.ReferenceRevision != r.seq {
		return errors.New("written migration reference revision changed")
	}
	reference, err := r.surfaceReferenceLocked()
	if err != nil {
		return err
	}
	if reference != plan.ReferenceHash {
		return errors.New("written migration reference state changed")
	}
	expected, err := r.brain.PlanWrittenNumericProjection()
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(expected, plan.Projections) {
		return errors.New("written migration numeric routes changed")
	}
	return nil
}

// Apply requires a NEW backup path. It persists the complete old checkpoint
// before appending the migration receipt and touching canonical projections.
// A failed/uncertain append retains the backup and requires owner recovery.
func (r *BrainRuntime) ApplyNeuralWrittenGrounding(plan WrittenGroundingPlan, backupPath string, at time.Time) error {
	if r == nil || r.brain == nil {
		return errors.New("brain runtime is not initialized")
	}
	if strings.TrimSpace(backupPath) == "" {
		return errors.New("written migration requires a new backup path")
	}
	data, err := json.Marshal(plan)
	if err != nil {
		return err
	}
	var frozen WrittenGroundingPlan
	if err := json.Unmarshal(data, &frozen); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ensureWritableLocked(); err != nil {
		return err
	}
	if err := r.validateWrittenGroundingPlanLocked(frozen); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(backupPath), 0755); err != nil {
		return err
	}
	reservation, err := os.OpenFile(backupPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("reserve migration backup: %w", err)
	}
	if err := reservation.Close(); err != nil {
		_ = os.Remove(backupPath)
		return err
	}
	if r.eventLog != nil {
		journalReservation, err := os.OpenFile(backupPath+".events.jsonl", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			_ = os.Remove(backupPath)
			return fmt.Errorf("reserve migration journal backup: %w", err)
		}
		if err := journalReservation.Close(); err != nil {
			_ = os.Remove(backupPath)
			_ = os.Remove(backupPath + ".events.jsonl")
			return err
		}
	}
	// The pre-migration copy is deliberately v12 so the H18 binary can also
	// restore it for rollback. Numeric serving itself requires v13.
	if err := r.checkpointVersionLocked(backupPath, evidenceProvenanceBrainSnapshotSchemaVersion); err != nil {
		_ = os.Remove(backupPath)
		if r.eventLog != nil {
			_ = os.Remove(backupPath + ".events.jsonl")
		}
		return err
	}
	var journalHash string
	if r.eventLog != nil {
		var err error
		journalHash, err = r.backupMigrationJournalLocked(backupPath + ".events.jsonl")
		if err != nil {
			_ = os.Remove(backupPath + ".events.jsonl")
			return err
		}
	}
	directory, err := os.Open(filepath.Dir(backupPath))
	if err != nil {
		return err
	}
	syncErr, closeDirectoryErr := directory.Sync(), directory.Close()
	if syncErr != nil {
		return syncErr
	}
	if closeDirectoryErr != nil {
		return closeDirectoryErr
	}
	file, err := os.Open(backupPath)
	if err != nil {
		return err
	}
	hash := sha256.New()
	_, copyErr := io.Copy(hash, file)
	closeErr := file.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	migration := WrittenGroundingMigration{Plan: frozen, BackupHash: "sha256:" + hex.EncodeToString(hash.Sum(nil)), BackupJournalHash: journalHash}
	if at.IsZero() {
		at = r.nowLocked()
	}
	return r.applyWrittenGroundingMigrationLocked(migration, at.UTC())
}

// Caller owns r.mu. Copy the old physical log under its writer lock, retaining
// a prefix that the H18 reader can validate without seeing the new receipt.
func (r *BrainRuntime) backupMigrationJournalLocked(path string) (string, error) {
	log := r.eventLog
	log.mu.Lock()
	defer log.mu.Unlock()
	if log.file == nil || log.lockFile == nil {
		return "", errors.New("migration journal is not open and locked")
	}
	events, err := ReadEventLog(log.file.Name())
	if err != nil {
		return "", err
	}
	if uint64(len(events)) != r.seq {
		return "", errors.New("migration journal prefix changed")
	}
	input, err := os.Open(log.file.Name())
	if err != nil {
		return "", err
	}
	defer input.Close()
	output, err := os.CreateTemp(filepath.Dir(path), ".horizon-journal-backup-*.tmp")
	if err != nil {
		return "", err
	}
	defer os.Remove(output.Name())
	defer output.Close()
	hash := sha256.New()
	if _, err := io.Copy(io.MultiWriter(output, hash), input); err != nil {
		return "", err
	}
	if err := output.Sync(); err != nil {
		return "", err
	}
	if err := output.Close(); err != nil {
		return "", err
	}
	if err := os.Rename(output.Name(), path); err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}

func (r *BrainRuntime) applyWrittenGroundingMigrationLocked(migration WrittenGroundingMigration, at time.Time) error {
	if err := migration.validateShape(); err != nil {
		return err
	}
	if err := r.validateWrittenGroundingPlanLocked(migration.Plan); err != nil {
		return err
	}
	return r.runJournalCommandLocked(LoggedEvent{Type: EventTypeWrittenGroundingMigration, Timestamp: at, WrittenGroundingMigration: &migration}, func() error {
		if err := r.brain.ApplyWrittenNumericProjection(migration.Plan.Projections); err != nil {
			return err
		}
		r.neuralWrittenGrounding = true
		r.surfaceEncoding = knowledge.NeuralWrittenEncoding
		r.inspectionEpoch = ""
		r.seq++
		return nil
	})
}

// Replay consumes the authenticated reference/digest, never a local backup path.
// Only the live public command creates backups; replay must not overwrite files.
func (r *BrainRuntime) replayWrittenGroundingMigration(m WrittenGroundingMigration, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ensureWritableLocked(); err != nil {
		return err
	}
	return r.applyWrittenGroundingMigrationLocked(m, at)
}

// WrittenGroundingContract reports a serving/compatibility contract, not a
// language capability or completion status for the P0 neural-knowledge work.
func (r *BrainRuntime) WrittenGroundingContract() (knowledge.SurfaceEncoding, error) {
	if r == nil || r.brain == nil {
		return "", errors.New("brain runtime is not initialized")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ensureWritableLocked(); err != nil {
		return "", err
	}
	return r.surfaceEncoding, nil
}
