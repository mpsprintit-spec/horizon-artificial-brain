package runtime

import (
	"errors"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// GroundingIntegration is the runtime boundary between observation handling
// and the canonical knowledge substrate. It does not create a second memory
// store and does not turn a candidate representation into trusted knowledge.
type GroundingIntegration struct {
	Brain *knowledge.Brain
}

func NewGroundingIntegration(brain *knowledge.Brain) *GroundingIntegration {
	return &GroundingIntegration{Brain: brain}
}

// GroundObservation presents an observation vector to the shared Brain and
// reports whether the representation was already present. Existing
// representations are reused; novel observations remain candidates until a
// separate learning/promotion policy accepts them.
func (g *GroundingIntegration) GroundObservation(vector knowledge.NeuralVector) (knowledge.GroundingResult, error) {
	if g == nil || g.Brain == nil {
		return knowledge.GroundingResult{}, errors.New("grounding integration is not initialized")
	}
	if vector.Empty() {
		return knowledge.GroundingResult{}, errors.New("grounding vector is empty")
	}
	return g.Brain.GroundVector(vector)
}
