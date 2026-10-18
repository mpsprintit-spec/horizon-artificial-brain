package runtime

import (
	"errors"
	"sync"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

var ErrManagedGroundingRequired = errors.New("canonical brain is supervisor-owned; use managed grounding integration")

// Managed brain identities are never released: a retired substrate must not
// become writable again through a previously held legacy adapter pointer.
// This is a legacy-adapter guard, NOT general pointer revocation; direct
// callers of knowledge.Brain APIs still require separate migration.
var supervisorOwnedBrains sync.Map

func markSupervisorOwnedBrain(brain *knowledge.Brain) {
	if brain != nil { supervisorOwnedBrains.Store(brain, struct{}{}) }
}
func supervisorOwnsBrain(brain *knowledge.Brain) bool {
	if brain == nil { return false }
	_, exists := supervisorOwnedBrains.Load(brain)
	return exists
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
