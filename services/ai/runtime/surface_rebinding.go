package runtime

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

const EventTypeSurfaceRebinding = "surface_rebinding"

// WrittenSurfaceRebinding is a reviewable migration receipt, not a new memory
// store. Its reference binds the revision, canonical Brain and adapter metadata.
type WrittenSurfaceRebinding struct {
	ReferenceRevision uint64                      `json:"reference_revision"`
	ReferenceHash     string                      `json:"reference_hash"`
	Source            knowledge.SurfaceAnnotation `json:"source"`
}

func (r *BrainRuntime) surfaceReferenceLocked() (string, error) {
	brain, err := r.brain.CanonicalJSON()
	if err != nil {
		return "", err
	}
	data, err := json.Marshal(struct {
		Revision    uint64                        `json:"revision"`
		Brain       json.RawMessage               `json:"brain"`
		Annotations []knowledge.SurfaceAnnotation `json:"annotations"`
	}{r.seq, brain, r.brain.SurfaceAnnotationsSnapshot()})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

// PlanWrittenSurfaceRebinding is read-only. Select the exact legacy surface and
// modality found in the reference checkpoint; conflicting populations are not
// automatically merged. Keep the plan and apply under the same owner lease.
func (r *BrainRuntime) PlanWrittenSurfaceRebinding(surface, modality string) (WrittenSurfaceRebinding, error) {
	if r == nil || r.brain == nil {
		return WrittenSurfaceRebinding{}, errors.New("brain runtime is not initialized")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ensureWritableLocked(); err != nil {
		return WrittenSurfaceRebinding{}, err
	}
	if r.neuralWrittenGrounding { return WrittenSurfaceRebinding{}, errors.New("metadata rebinding cannot change numeric written routes") }
	source, err := r.brain.LegacyWrittenSurface(surface, modality)
	if err != nil {
		return WrittenSurfaceRebinding{}, err
	}
	reference, err := r.surfaceReferenceLocked()
	return WrittenSurfaceRebinding{ReferenceRevision: r.seq, ReferenceHash: reference, Source: source}, err
}

// ApplyWrittenSurfaceRebinding records the explicit selection before binding
// new written input to that existing population. Stale or changed plans fail
// before journaling. Take a reference checkpoint before planning a migration.
func (r *BrainRuntime) ApplyWrittenSurfaceRebinding(plan WrittenSurfaceRebinding, at time.Time) error {
	if r == nil || r.brain == nil {
		return errors.New("brain runtime is not initialized")
	}
	data, err := json.Marshal(plan)
	if err != nil {
		return err
	}
	var frozen WrittenSurfaceRebinding
	if err := json.Unmarshal(data, &frozen); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ensureWritableLocked(); err != nil {
		return err
	}
	if r.neuralWrittenGrounding { return errors.New("metadata rebinding cannot change numeric written routes") }
	if frozen.ReferenceRevision != r.seq {
		return errors.New("surface rebinding reference revision changed")
	}
	reference, err := r.surfaceReferenceLocked()
	if err != nil {
		return err
	}
	if frozen.ReferenceHash != reference {
		return errors.New("surface rebinding reference state changed")
	}
	if err := r.brain.ValidateWrittenSurfaceSource(frozen.Source); err != nil {
		return err
	}
	if at.IsZero() {
		at = r.nowLocked()
	}
	at = at.UTC()
	return r.runJournalCommandLocked(LoggedEvent{Type: EventTypeSurfaceRebinding, Timestamp: at, SurfaceRebinding: &frozen}, func() error {
		if err := r.brain.RebindWrittenSurface(frozen.Source, frozen.ReferenceHash, at); err != nil {
			return err
		}
		r.seq++
		return nil
	})
}
