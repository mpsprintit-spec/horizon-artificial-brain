package runtime

import (
	"errors"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// GroundingIntegration is the runtime boundary for presenting numeric
// observations to the canonical Horizon brain. It intentionally contains no
// second memory store.
type GroundingIntegration struct {
	brain *knowledge.Brain
}

func NewGroundingIntegration(brain *knowledge.Brain) *GroundingIntegration {
	return &GroundingIntegration{brain: brain}
}

// GroundObservation projects a numeric observation into the shared brain and
// reports whether the resulting representation was reused.
func (g *GroundingIntegration) GroundObservation(vector knowledge.NeuralVector) (knowledge.GroundedVector, error) {
	if g == nil || g.brain == nil {
		return knowledge.GroundedVector{}, errors.New("grounding integration is not initialized")
	}
	return g.brain.GroundVector(vector)
}
