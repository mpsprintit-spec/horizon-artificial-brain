package runtime

import (
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// GroundObservationAt preserves event time through the runtime boundary.
func (r *BrainRuntime) GroundObservationAt(token, source, modality string, at time.Time) (knowledge.GroundedRepresentation, error) {
	if r == nil || r.brain == nil {
		return knowledge.GroundedRepresentation{}, ErrBrainRuntimeNotInitialized
	}
	return r.brain.GroundObservationAt(token, source, modality, GroundingThreshold, at)
}
