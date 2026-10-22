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
	if r.eventLog != nil {
		return knowledge.GroundedRepresentation{}, errors.New("standalone grounding is forbidden with an attached journal; use ProcessObservation")
	}
	return r.brain.GroundObservationWithEncodingAt(token, source, modality, GroundingThreshold, at, r.surfaceEncoding)
}
