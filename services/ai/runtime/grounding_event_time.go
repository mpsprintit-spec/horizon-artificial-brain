package runtime

import (
	"errors"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// GroundObservationAt preserves event time through the runtime boundary.
func (r *BrainRuntime) GroundObservationAt(token, source, modality string, at time.Time) (knowledge.GroundedRepresentation, error) {
	if r == nil || r.brain == nil {
		return knowledge.GroundedRepresentation{}, errors.New("brain runtime is not initialized")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ensureWritableLocked(); err != nil {
		return knowledge.GroundedRepresentation{}, err
	}
	return r.brain.GroundObservationAt(token, source, modality, GroundingThreshold, at)
}
