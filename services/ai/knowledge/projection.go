package knowledge

import (
	"errors"
	"sort"
	"time"
)

const defaultProjectionPopulation = 4

var (
	ErrNilBrain          = errors.New("brain is nil")
	ErrEmptyNeuralVector = errors.New("neural vector is empty")
)

type ProjectionPopulation struct {
	Units []PopulationUnit
}

type PopulationUnit struct {
	NodeID     NodeID
	Activation float64
}

// ProjectVectorPopulation recruits a sparse population from the same Brain
// substrate. Existing compatible units are reused; new receptive prototypes
// are grown only when the substrate has insufficient compatible structure.
func (k *KnowledgeBase) ProjectVectorPopulation(vector NeuralVector, threshold float64, populationSize int) (ProjectionPopulation, error) {
	if k == nil {
		return ProjectionPopulation{}, ErrNilBrain
	}
	if vector.Empty() {
		return ProjectionPopulation{}, ErrEmptyNeuralVector
	}
	threshold = clamp(threshold, 0, 1)
	if populationSize <= 0 {
		populationSize = defaultProjectionPopulation
	}

	candidates := k.matchRepresentationPopulation(vector, threshold)
	for slot := 0; len(candidates) < populationSize; slot++ {
		prototype := projectionPrototype(vector, slot, populationSize)
		node := k.createRepresentationPrototype(prototype)
		score := NewNeuralVector(node.Representation).Similarity(vector)
		if score >= threshold && !containsPopulationNode(candidates, node.ID) {
			candidates = append(candidates, populationCandidate{node: node, score: score})
		}
		if slot >= populationSize*2 {
			break
		}
	}

	sort.Slice(candidates, func(i, j int) bool { return candidates[i].score > candidates[j].score })
	if len(candidates) > populationSize {
		candidates = candidates[:populationSize]
	}
	population := ProjectionPopulation{Units: make([]PopulationUnit, len(candidates))}
	now := time.Now().UTC()
	for i, item := range candidates {
		item.node.Activation = clamp01(item.score)
		item.node.LastActivation = now
		population.Units[i] = PopulationUnit{NodeID: item.node.ID, Activation: clamp01(item.score)}
	}
	return population, nil
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
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].score > candidates[j].score })
	return candidates
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

func containsPopulationNode(candidates []populationCandidate, id NodeID) bool {
	for _, candidate := range candidates {
		if candidate.node.ID == id {
			return true
		}
	}
	return false
}

// projectionPrototype creates nearby receptive fields around the presented
// vector. The variation is deterministic and coordinate-local, so repeating
// the same experience can recruit the same distributed population.
func projectionPrototype(vector NeuralVector, slot, populationSize int) NeuralVector {
	if slot == 0 || len(vector.Values) == 0 {
		return NewNeuralVector(vector.Values)
	}
	out := append([]float64(nil), vector.Values...)
	spread := 0.08 + 0.04*float64(slot)/float64(maxInt(populationSize, 1))
	for i := range out {
		direction := -1.0
		if (i+slot)%2 == 0 {
			direction = 1
		}
		out[i] = clamp(out[i]+direction*spread, -1, 1)
	}
	return NewNeuralVector(out)
}
