package runtime

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

var ErrInvalidGroundVector = errors.New("invalid numeric grounding vector")

// validateGroundVector prevents non-finite values and unexpectedly large
// documents from entering the durable journal or canonical neural substrate.
func validateGroundVector(vector knowledge.NeuralVector) error {
	if len(vector.Values) == 0 || len(vector.Values) > 128 {
		return fmt.Errorf("%w: expected 1..128 numeric dimensions", ErrInvalidGroundVector)
	}
	for i, value := range vector.Values {
		if math.IsNaN(value) || math.IsInf(value, 0) || value < -1 || value > 1 {
			return fmt.Errorf("%w: dimension %d is not finite and normalized", ErrInvalidGroundVector, i)
		}
	}
	return nil
}

// GroundVectorAt is the single replayable runtime entry point for numeric
// adapter grounding. Unlike a bare knowledge.Brain projection, it acquires
// the runtime lock and advances one ordered journal revision per observation.
// Its only long-lived state is the canonical Brain; no second memory is made.
func (r *BrainRuntime) GroundVectorAt(vector knowledge.NeuralVector, at time.Time) (knowledge.GroundedVector, error) {
	if r == nil || r.brain == nil { return knowledge.GroundedVector{}, errors.New("brain runtime is not initialized") }
	if err := validateGroundVector(vector); err != nil { return knowledge.GroundedVector{}, err }
	// Freeze caller-owned values before waiting on the runtime lock.
	frozen := knowledge.NewNeuralVector(vector.Values)
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ensureWritableLocked(); err != nil { return knowledge.GroundedVector{}, err }
	if at.IsZero() { at = r.nowLocked() } else { at = at.UTC() }
	if r.eventLog != nil {
		if err := r.eventLog.Append(LoggedEvent{
			SchemaVersion: EventLogSchemaVersion, BrainIdentity: BrainIdentity,
			Sequence: r.seq + 1, Type: EventTypeVectorGround,
			Timestamp: at, Vector: &frozen,
		}); err != nil {
			r.markRecoveryRequiredLocked()
			return knowledge.GroundedVector{}, fmt.Errorf("%w: append numeric grounding: %v", ErrRuntimeRecoveryRequired, err)
		}
	}
	grounded, err := r.brain.GroundVectorAt(frozen, at)
	if err != nil {
		r.markRecoveryRequiredLocked()
		return knowledge.GroundedVector{}, fmt.Errorf("%w: apply numeric grounding: %v", ErrRuntimeRecoveryRequired, err)
	}
	r.seq++
	// Numeric adapter grounding does not change r.lastObservationPopulation:
	// that predecessor belongs to the sensor temporal learning pipeline.
	return grounded, nil
}

// GroundVector is the runtime-clock variant; the timestamp is captured under
// the runtime lock by GroundVectorAt so journal and neural event-time agree.
func (r *BrainRuntime) GroundVector(vector knowledge.NeuralVector) (knowledge.GroundedVector, error) {
	return r.GroundVectorAt(vector, time.Time{})
}
