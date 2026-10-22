package runtime

import (
	"errors"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

var ErrManagedGroundingRequired = errors.New("canonical brain is supervisor-owned; use managed grounding integration")

// Ownership is stored atomically on the canonical Brain itself, so an
// unlimited number of recoveries does not retain retired neural substrates
// through a global pointer map.
func markSupervisorOwnedBrain(brain *knowledge.Brain) {
	if brain != nil { brain.MarkRuntimeOwned() }
}
func supervisorOwnsBrain(brain *knowledge.Brain) bool {
	return brain != nil && brain.RuntimeOwned()
}

// GroundingIntegration presents numeric observations to the ONE Brain.
// Managed adapters resolve the active runtime on every observation; legacy
// adapters remain supported only for independently owned, standalone brains.
type GroundingIntegration struct {
	brain *knowledge.Brain
	supervisor *RuntimeSupervisor
}

// NewGroundingIntegration is the standalone/legacy constructor. It must not
// be used to access a Brain that has been enrolled into RuntimeSupervisor.
func NewGroundingIntegration(brain *knowledge.Brain) *GroundingIntegration {
	return &GroundingIntegration{brain: brain}
}

func NewManagedGroundingIntegration(supervisor *RuntimeSupervisor) (*GroundingIntegration, error) {
	if supervisor == nil { return nil, errors.New("grounding runtime supervisor is nil") }
	if err := supervisor.With(func(*BrainRuntime) error { return nil }); err != nil { return nil, err }
	return &GroundingIntegration{supervisor: supervisor}, nil
}

// GroundObservation preserves distributed numeric population grounding.
func (g *GroundingIntegration) GroundObservation(vector knowledge.NeuralVector) (knowledge.GroundedVector, error) {
	if g == nil { return knowledge.GroundedVector{}, ErrGroundingIntegrationNotInitialized }
	if g.supervisor != nil { return g.GroundObservationAt(vector, time.Time{}) }
	if g.brain == nil { return knowledge.GroundedVector{}, ErrGroundingIntegrationNotInitialized }
	if supervisorOwnsBrain(g.brain) { return knowledge.GroundedVector{}, ErrManagedGroundingRequired }
	return g.brain.GroundVector(vector)
}
