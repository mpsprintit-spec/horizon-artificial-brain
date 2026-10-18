package runtime

import (
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// GroundObservationAt preserves event time across numeric grounding and
// journal replay. Managed calls resolve the current runtime within an
// exclusive supervisor mutation lease, never a cached Brain pointer.
func (g *GroundingIntegration) GroundObservationAt(vector knowledge.NeuralVector, now time.Time) (knowledge.GroundedVector, error) {
	if g == nil { return knowledge.GroundedVector{}, ErrGroundingIntegrationNotInitialized }
	if g.supervisor != nil {
		var grounded knowledge.GroundedVector
		err := g.supervisor.WithMutation(func(rt *BrainRuntime) error {
			var err error
			grounded, err = rt.GroundVectorAt(vector, now)
			return err
		})
		return grounded, err
	}
	if g.brain == nil { return knowledge.GroundedVector{}, ErrGroundingIntegrationNotInitialized }
	if supervisorOwnsBrain(g.brain) { return knowledge.GroundedVector{}, ErrManagedGroundingRequired }
	nodeID, similarity, created, err := g.brain.ProjectVectorAt(vector, 0.90, now)
	if err != nil { return knowledge.GroundedVector{}, err }
	return knowledge.GroundedVector{NodeID: nodeID, Similarity: similarity, Existing: !created}, nil
}
