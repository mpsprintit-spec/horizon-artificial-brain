package knowledge

import (
	"errors"
	"sort"
	"time"
)

const defaultProjectionPopulation = 4
const projectionExactThreshold = 0.999999

var (
	ErrNilBrain          = errors.New("brain is nil")
	ErrEmptyNeuralVector = errors.New("neural vector is empty")
)

type ProjectionPopulation struct{ Units []PopulationUnit }
type PopulationUnit struct {
	NodeID     NodeID
	Activation float64
}

// ProjectVectorPopulation recruits a sparse population from the same Brain
// substrate. An exact repeated experience recovers its previously registered
// population. A similar but distinct experience reuses only its strongest
// compatible unit and grows new members, preserving both shared structure and
// population separation.
func (k *KnowledgeBase) ProjectVectorPopulation(vector NeuralVector, threshold float64, populationSize int) (ProjectionPopulation, error) {
	return k.ProjectVectorPopulationAt(vector, threshold, populationSize, time.Time{})
}

// ProjectVectorPopulationAt is the event-time-aware population projection path.
// All node activation timestamps created by the projection use the supplied
// timestamp, making replay independent of wall-clock time.
func (k *KnowledgeBase) ProjectVectorPopulationAt(vector NeuralVector, threshold float64, populationSize int, now time.Time) (ProjectionPopulation, error) {
	if k == nil {
		return ProjectionPopulation{}, ErrNilBrain
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	if vector.Empty() {
		return ProjectionPopulation{}, ErrEmptyNeuralVector
	}
	threshold = clamp(threshold, 0, 1)
	if populationSize <= 0 {
		populationSize = defaultProjectionPopulation
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	// Population identity is structural state, so recover an exact learned
	// projection before considering generic unit similarity. This prevents a
	// later similar experience from accidentally stealing members from an
	// earlier population merely because it ranks highly against them.
	if population, ok := k.findExactProjectionPopulation(vector); ok {
		return k.activateProjectionPopulationAt(population, vector, now), nil
	}

	candidates := k.matchRepresentationPopulation(vector, threshold)
	population := ProjectionPopulation{Units: make([]PopulationUnit, 0, populationSize)}

	if len(candidates) > 0 {
		// Similar experiences may share a compatible substrate unit, but they
		// must retain a distinct distributed population. Reusing only the
		// strongest candidate creates a shared anchor while leaving room for
		// experience-specific structure to grow.
		population.Units = append(population.Units, PopulationUnit{
			NodeID: candidates[0].node.ID,
			Activation: clamp01(candidates[0].score),
		})
	}

	for slot := len(population.Units); slot < populationSize; slot++ {
		node := k.createRepresentationPrototype(projectionPrototype(vector, slot, populationSize))
		score := NewNeuralVector(node.Representation).Similarity(vector)
		population.Units = append(population.Units, PopulationUnit{
			NodeID: node.ID,
			Activation: clamp01(score),
		})
	}

	k.registerProjectionPopulation(population)
	return k.activateProjectionPopulationAt(population, vector, now), nil
}

type populationCandidate struct {
	node  *ConceptNode
	score float64
}

func (k *KnowledgeBase) matchRepresentationPopulation(vector NeuralVector, threshold float64) []populationCandidate {
	nodes := k.Registry.Nodes()
	candidates := make([]populationCandidate, 0, len(nodes))
	for _, node := range nodes {
		if len(node.Representation) != len(vector.Values) {
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
	return candidates
}

func (k *KnowledgeBase) findExactProjectionPopulation(vector NeuralVector) (ProjectionPopulation, bool) {
	k.projectionMu.RLock()
	populations := append([]ProjectionPopulation(nil), k.ProjectionPopulations...)
	k.projectionMu.RUnlock()

	bestScore := 0.0
	var best ProjectionPopulation
	found := false
	for _, population := range populations {
		for _, unit := range population.Units {
			node := k.Registry.GetByID(unit.NodeID)
			if node == nil || len(node.Representation) != len(vector.Values) {
				continue
			}
			score := NewNeuralVector(node.Representation).Similarity(vector)
			if score >= projectionExactThreshold && (!found || score > bestScore) {
				best = population
				bestScore = score
				found = true
			}
		}
	}
	return best, found
}

func (k *KnowledgeBase) registerProjectionPopulation(population ProjectionPopulation) {
	k.projectionMu.Lock()
	defer k.projectionMu.Unlock()
	copyUnits := append([]PopulationUnit(nil), population.Units...)
	k.ProjectionPopulations = append(k.ProjectionPopulations, ProjectionPopulation{Units: copyUnits})
}

func (k *KnowledgeBase) activateProjectionPopulation(population ProjectionPopulation, vector NeuralVector) ProjectionPopulation {
	return k.activateProjectionPopulationAt(population, vector, time.Now().UTC())
}

func (k *KnowledgeBase) activateProjectionPopulationAt(population ProjectionPopulation, vector NeuralVector, now time.Time) ProjectionPopulation {
	if now.IsZero() { now = time.Now().UTC() } else { now = now.UTC() }
	out := ProjectionPopulation{Units: make([]PopulationUnit, len(population.Units))}
	for i, unit := range population.Units {
		node := k.Registry.GetByID(unit.NodeID)
		if node == nil {
			continue
		}
		score := NewNeuralVector(node.Representation).Similarity(vector)
		node.Activation = clamp01(score)
		node.LastActivation = now
		out.Units[i] = PopulationUnit{NodeID: node.ID, Activation: clamp01(score)}
	}
	return out
}

func (k *KnowledgeBase) createRepresentationPrototype(vector NeuralVector) *ConceptNode {
	k.Registry.mu.Lock()
	defer k.Registry.mu.Unlock()
	node := newRepresentationNode(k.Registry.nextID, vector.Values)
	node.Frequency = 1
	node.LastActivation = time.Now().UTC()
	k.Registry.nextID++
	k.Registry.byID[node.ID] = node
	return node
}

// projectionPrototype creates nearby receptive fields only for the first
// presentation of a previously unseen vector. The perturbation is deliberately
// narrow so the members remain in one reusable representation basin.
func projectionPrototype(vector NeuralVector, slot, populationSize int) NeuralVector {
	if slot == 0 || len(vector.Values) == 0 {
		return NewNeuralVector(vector.Values)
	}
	out := append([]float64(nil), vector.Values...)
	spread := 0.02 + 0.01*float64(slot)/float64(maxInt(populationSize, 1))
	for i := range out {
		direction := -1.0
		if (i+slot)%2 == 0 {
			direction = 1
		}
		out[i] = clamp(out[i]+direction*spread, -1, 1)
	}
	return NewNeuralVector(out)
}
