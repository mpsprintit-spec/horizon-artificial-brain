package knowledge

import (
	"errors"
	"time"
)

// GroundedVector is the result of presenting a numeric observation to the
// canonical brain substrate. Existing reports whether the representation was
// already present before this grounding call.
type GroundedVector struct {
	NodeID     NodeID
	Population []NodeID
	Similarity float64
	Existing   bool
}

// GroundVector grounds a numeric observation directly into the canonical
// neural substrate. It does not create a second vector-memory store.
func (k *KnowledgeBase) GroundVector(vector NeuralVector) (GroundedVector, error) {
	if k == nil {
		return GroundedVector{}, errors.New("brain is nil")
	}
	return k.GroundVectorAt(vector, time.Time{})
}

// GroundVectorAt preserves the timestamp supplied by the observing adapter.
// Numeric grounding never creates another vector store; it only changes the
// same population substrate that the runtime uses for cognition and replay.
func (k *KnowledgeBase) GroundVectorAt(vector NeuralVector, at time.Time) (GroundedVector, error) {
	return k.GroundVectorWithDynamicsAt(vector, at, LegacyDynamics)
}

func (k *KnowledgeBase) GroundVectorWithDynamicsAt(vector NeuralVector, at time.Time, dynamics DynamicsEncoding) (GroundedVector, error) {
	if k == nil {
		return GroundedVector{}, errors.New("brain is nil")
	}
	if vector.Empty() {
		return GroundedVector{}, errors.New("neural vector is empty")
	}

	const threshold = 0.90
	_, hadPopulation := k.findExactProjectionPopulation(vector)
	if dynamics.HasStableProjection() {
		k.mu.RLock()
		_, hadPopulation = k.findStableProjectionPopulation(vector, defaultProjectionPopulation, nil)
		k.mu.RUnlock()
	}
	population, err := k.ProjectVectorPopulationWithDynamicsAt(vector, threshold, defaultProjectionPopulation, at, dynamics, nil)
	if err != nil || len(population.Units) == 0 {
		if err != nil {
			return GroundedVector{}, err
		}
		return GroundedVector{}, ErrEmptyNeuralVector
	}
	ids := make([]NodeID, 0, len(population.Units))
	for _, unit := range population.Units {
		ids = append(ids, unit.NodeID)
	}
	return GroundedVector{
		NodeID:     ids[0],
		Population: ids,
		Similarity: population.Units[0].Activation,
		Existing:   hadPopulation,
	}, nil
}
