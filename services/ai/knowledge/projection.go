package knowledge

import "sort"

const defaultProjectionPopulation = 4

// ProjectionPopulation is the distributed neural activity produced by a
// numeric experience. It contains only neural-unit IDs and their activation
// strengths; no semantic labels are attached to the population.
type ProjectionPopulation struct {
	Units []PopulationUnit
}

type PopulationUnit struct {
	NodeID     NodeID
	Activation float64
}

// ProjectVectorPopulation recruits a sparse population from the same Brain
// substrate. Existing compatible units are reused; new units are grown only
// when the substrate has insufficient compatible structure.
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

	nodes := k.Registry.Nodes()
	type candidate struct {
		node  *ConceptNode
		score float64
	}
	candidates := make([]candidate, 0, len(nodes))
	for _, node := range nodes {
		if len(node.Representation) != len(vector.Values) {
			continue
		}
		score := NewNeuralVector(node.Representation).Similarity(vector)
		if score >= threshold {
			candidates = append(candidates, candidate{node: node, score: score})
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	if len(candidates) < populationSize {
		// The residual structure is represented by a new prototype. Its
		// prototype is the current experience itself, so later repetitions
		// naturally recruit it instead of creating another copy.
		node, _, err := k.Registry.GetOrCreateRepresentation(vector, threshold)
		if err != nil {
			return ProjectionPopulation{}, err
		}
		found := false
		for _, item := range candidates {
			if item.node.ID == node.ID {
				found = true
				break
			}
		}
		if !found {
			candidates = append(candidates, candidate{node: node, score: 1})
		}
	}

	if len(candidates) > populationSize {
		candidates = candidates[:populationSize]
	}
	population := ProjectionPopulation{Units: make([]PopulationUnit, len(candidates))}
	for i, item := range candidates {
		item.node.Activation = item.score
		population.Units[i] = PopulationUnit{NodeID: item.node.ID, Activation: item.score}
	}
	return population, nil
}
