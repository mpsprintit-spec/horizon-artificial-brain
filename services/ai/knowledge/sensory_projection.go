package knowledge

import (
	"errors"
	"math"
	"sort"
	"time"
)

// ProjectionScope identifies measurement coordinates, never an object class.
// Location and sample IDs belong to bindings, so the same physical factor
// may be reused at different locations without merging their scene bindings.
type ProjectionScope struct {
	Modality        string `json:"modality"`
	Field           string `json:"field"`
	TemporalScaleNS int64  `json:"temporal_scale_ns"`
}

func cloneProjectionScope(scope *ProjectionScope) *ProjectionScope {
	if scope == nil {
		return nil
	}
	out := *scope
	return &out
}

// ProjectSensoryFieldAt recruits units from the one canonical substrate.
// Receptive-field reuse is limited to compatible measurement coordinates.
// A compatible existing population is reactivated as a whole; a distinct
// factor can share its strongest compatible unit and grow new members.
func (k *KnowledgeBase) ProjectSensoryFieldAt(vector NeuralVector, scope ProjectionScope, threshold float64, populationSize int, at time.Time) (ProjectionPopulation, error) {
	if k == nil {
		return ProjectionPopulation{}, ErrNilBrain
	}
	if vector.Empty() {
		return ProjectionPopulation{}, ErrEmptyNeuralVector
	}
	if scope.Modality == "" || scope.Field == "" || scope.TemporalScaleNS < 0 || at.IsZero() {
		return ProjectionPopulation{}, errors.New("sensory projection requires measurement scope and event time")
	}
	for _, value := range vector.Values {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return ProjectionPopulation{}, errors.New("sensory projection is not finite")
		}
	}
	if populationSize <= 0 {
		populationSize = defaultProjectionPopulation
	}
	threshold = clamp01(threshold)
	k.mu.Lock()
	defer k.mu.Unlock()
	k.projectionMu.RLock()
	populations := append([]ProjectionPopulation(nil), k.ProjectionPopulations...)
	k.projectionMu.RUnlock()
	bestScore := -1.0
	var best ProjectionPopulation
	compatible := make(map[NodeID]struct{})
	for _, population := range populations {
		if population.Scope == nil || *population.Scope != scope {
			continue
		}
		for _, unit := range population.Units {
			compatible[unit.NodeID] = struct{}{}
		}
		if len(population.Prototype) != len(vector.Values) {
			continue
		}
		score := NewNeuralVector(population.Prototype).Similarity(vector)
		if score > bestScore {
			best, bestScore = population, score
		}
	}
	if bestScore >= threshold {
		return k.activateProjectionPopulationAt(best, vector, at.UTC()), nil
	}
	candidates := make([]populationCandidate, 0, len(compatible))
	for id := range compatible {
		node := k.Registry.GetByID(id)
		if node == nil || len(node.Representation) != len(vector.Values) {
			continue
		}
		score := NewNeuralVector(node.Representation).Similarity(vector)
		if score >= threshold {
			candidates = append(candidates, populationCandidate{node: node, score: score})
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score == candidates[j].score {
			return candidates[i].node.ID < candidates[j].node.ID
		}
		return candidates[i].score > candidates[j].score
	})
	population := ProjectionPopulation{
		Scope: cloneProjectionScope(&scope), Prototype: append([]float64(nil), vector.Values...),
		Units:      make([]PopulationUnit, 0, populationSize),
		Provenance: BootstrapProvenance{Origin: "sensor_measurement", DirectExperience: true, Status: "observed_state"},
	}
	if len(candidates) > 0 {
		population.Units = append(population.Units, PopulationUnit{NodeID: candidates[0].node.ID, Activation: candidates[0].score})
	}
	for slot := len(population.Units); slot < populationSize; slot++ {
		node := k.createRepresentationPrototypeAt(projectionPrototype(vector, slot, populationSize), at.UTC())
		population.Units = append(population.Units, PopulationUnit{NodeID: node.ID, Activation: NewNeuralVector(node.Representation).Similarity(vector)})
	}
	k.registerProjectionPopulation(population)
	return k.activateProjectionPopulationAt(population, vector, at.UTC()), nil
}
