package knowledge

import (
	"errors"
	"math"
	"slices"
	"time"
)

// ProjectVectorPopulationWithDynamicsAt retains historical projection unchanged.
// New dynamics recover the stored sensory prototype independently of learned
// member drift. Optional per-coordinate tolerances are an adapter measurement
// contract, never a claim that geometrically close experiences mean the same thing.
func (k *KnowledgeBase) ProjectVectorPopulationWithDynamicsAt(vector NeuralVector, threshold float64, size int, now time.Time, dynamics DynamicsEncoding, tolerance []float64) (ProjectionPopulation, error) {
	if err := dynamics.Validate(); err != nil {
		return ProjectionPopulation{}, err
	}
	if !dynamics.HasStableProjection() {
		if len(tolerance) > 0 {
			return ProjectionPopulation{}, errors.New("measurement tolerance requires stable projection dynamics")
		}
		return k.ProjectVectorPopulationAt(vector, threshold, size, now)
	}
	if k == nil {
		return ProjectionPopulation{}, ErrNilBrain
	}
	if vector.Empty() {
		return ProjectionPopulation{}, ErrEmptyNeuralVector
	}
	if len(tolerance) > 0 && len(tolerance) != len(vector.Values) {
		return ProjectionPopulation{}, errors.New("projection tolerance dimensions disagree")
	}
	for i, v := range vector.Values {
		if math.IsNaN(v) || math.IsInf(v, 0) || v < -1 || v > 1 {
			return ProjectionPopulation{}, errors.New("projection input must be finite and normalized")
		}
		if len(tolerance) > 0 && (math.IsNaN(tolerance[i]) || math.IsInf(tolerance[i], 0) || tolerance[i] < 0 || tolerance[i] > 1) {
			return ProjectionPopulation{}, errors.New("invalid projection measurement tolerance")
		}
	}
	if size <= 0 {
		size = defaultProjectionPopulation
	}
	threshold = clamp01(threshold)
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	if p, ok := k.findStableProjectionPopulation(vector, size, tolerance); ok {
		return k.activateProjectionPopulationAt(p, vector, now), nil
	}
	candidates := k.matchRepresentationPopulation(vector, threshold)
	population := ProjectionPopulation{Prototype: slices.Clone(vector.Values), Units: make([]PopulationUnit, 0, size)}
	if len(candidates) > 0 {
		population.Units = append(population.Units, PopulationUnit{NodeID: candidates[0].node.ID, Activation: clamp01(candidates[0].score)})
	}
	for slot := len(population.Units); slot < size; slot++ {
		node := k.createRepresentationPrototypeAt(projectionPrototype(vector, slot, size), now)
		population.Units = append(population.Units, PopulationUnit{NodeID: node.ID, Activation: clamp01(NewNeuralVector(node.Representation).Similarity(vector))})
	}
	k.registerProjectionPopulation(population)
	return k.activateProjectionPopulationAt(population, vector, now), nil
}

// Exact evidence wins. Tolerant recovery requires one eligible receptive field;
// overlapping uncertainty regions never arbitrarily merge existing populations.
// Caller holds the Brain lock; projection metadata is read under its own lock.
func (k *KnowledgeBase) findStableProjectionPopulation(vector NeuralVector, size int, tolerance []float64) (ProjectionPopulation, bool) {
	k.projectionMu.RLock()
	defer k.projectionMu.RUnlock()
	var candidate ProjectionPopulation
	matches := 0
	for _, p := range k.ProjectionPopulations {
		if p.Scope != nil || p.Domain != "" || len(p.Units) != size || len(p.Prototype) != len(vector.Values) {
			continue
		}
		valid := true
		for _, u := range p.Units {
			if k.Registry.GetByID(u.NodeID) == nil {
				valid = false
				break
			}
		}
		if !valid {
			continue
		}
		if slices.Equal(p.Prototype, vector.Values) {
			return p, true
		}
		if len(tolerance) == 0 {
			continue
		}
		inside := true
		for i, v := range vector.Values {
			if math.Abs(p.Prototype[i]-v) > tolerance[i] {
				inside = false
				break
			}
		}
		if inside {
			candidate = p
			matches++
		}
	}
	return candidate, matches == 1
}
