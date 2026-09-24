package knowledge

import "time"

// GetOrCreateAt is the event-time language-adapter path. The lexical form is
// immediately encoded into the shared neural substrate; it is never retained
// as semantic identity on the neural unit.
func (r *TokenRegistry) GetOrCreateAt(token string, now time.Time) (*ConceptNode, bool, error) {
	canonical := canonicalToken(token)
	if canonical == "" {
		return nil, false, ErrEmptyNeuralVector
	}
	return r.GetOrCreateRepresentationAt(observationVector(canonical, "language"), 0.999999, now)
}

// StoreAt presents language through the same distributed substrate used by
// other modalities. No token-indexed memory is created.
func (k *KnowledgeBase) StoreAt(token string, now time.Time) *ConceptNode {
	if k == nil || k.Registry == nil {
		return nil
	}
	canonical := canonicalToken(token)
	if canonical == "" {
		return nil
	}
	population, err := k.ProjectVectorPopulationAt(observationVector(canonical, "language"), 0.999999, defaultProjectionPopulation, now)
	if err != nil || len(population.Units) == 0 {
		return nil
	}
	return k.Registry.GetByID(population.Units[0].NodeID)
}
