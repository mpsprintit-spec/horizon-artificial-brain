package runtime

import (
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// GroundObservationAt preserves the supplied event timestamp through the
// grounding boundary, making temporal replay deterministic.
func (g *GroundingIntegration) GroundObservationAt(vector knowledge.NeuralVector, now time.Time) (knowledge.GroundedVector, error) {
	if g == nil || g.brain == nil {
		return knowledge.GroundedVector{}, ErrGroundingIntegrationNotInitialized
	}
	nodeID, similarity, created, err := g.brain.ProjectVectorAt(vector, 0.90, now)
	if err != nil {
		return knowledge.GroundedVector{}, err
	}
	return knowledge.GroundedVector{NodeID: nodeID, Similarity: similarity, Existing: !created}, nil
}
