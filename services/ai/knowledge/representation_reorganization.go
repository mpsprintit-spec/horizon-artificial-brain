package knowledge

import "time"

// AdaptPopulationRepresentationAt applies experience-driven plasticity to a
// distributed population. It does not assign a semantic identity and does not
// merge populations. Member representations drift toward the experienced
// state according to their own plasticity and current activation, while the
// population prototype tracks the same developmental trajectory.
//
// Repeated exposure to structurally compatible experiences can therefore make
// initially different distributed representations converge without requiring
// a hard-coded equivalence rule.
func (k *KnowledgeBase) AdaptPopulationRepresentationAt(population ProjectionPopulation, experience NeuralVector, learningRate float64, now time.Time) (ProjectionPopulation, error) {
	if k == nil {
		return ProjectionPopulation{}, ErrNilBrain
	}
	if experience.Empty() {
		return ProjectionPopulation{}, ErrEmptyNeuralVector
	}
	learningRate = clamp01(learningRate)
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	k.mu.Lock()
	defer k.mu.Unlock()

	updated := ProjectionPopulation{
		Domain:                 population.Domain,
		Units:                  append([]PopulationUnit(nil), population.Units...),
		Prototype:              append([]float64(nil), population.Prototype...),
		LearningTarget:         append([]NodeID(nil), population.LearningTarget...),
		PredictionTargets:      append([]NodeID(nil), population.PredictionTargets...),
		ErrorTargets:           append([]NodeID(nil), population.ErrorTargets...),
		PlasticityTargets:      append([]NodeID(nil), population.PlasticityTargets...),
		CounterEvidenceTargets: append([]NodeID(nil), population.CounterEvidenceTargets...),
		TemporalScale:          population.TemporalScale,
		BootstrapExperienceID:  population.BootstrapExperienceID,
		Provenance:             population.Provenance,
	}

	if len(updated.Prototype) != len(experience.Values) {
		updated.Prototype = append([]float64(nil), experience.Values...)
	} else {
		for i, value := range experience.Values {
			updated.Prototype[i] = clamp(updated.Prototype[i]+learningRate*(value-updated.Prototype[i]), -1, 1)
		}
	}

	for i, unit := range updated.Units {
		node := k.Registry.byID[unit.NodeID]
		if node == nil || len(node.Representation) != len(experience.Values) {
			continue
		}
		activation := clamp01(unit.Activation)
		rate := learningRate * clamp01(node.Plasticity) * activation
		for dimension, value := range experience.Values {
			node.Representation[dimension] = clamp(
				node.Representation[dimension]+rate*(value-node.Representation[dimension]),
				-1, 1,
			)
		}
		node.Frequency++
		node.LastActivation = now
		node.UsageHistory = append(node.UsageHistory, now)

		score := NewNeuralVector(node.Representation).Similarity(experience)
		node.Activation = clamp01(score)
		updated.Units[i].Activation = clamp01(score)
	}

	// Persist the reorganized population trace without replacing its substrate
	// membership. This is the distinction between representation learning and
	// destructive deduplication.
	k.projectionMu.Lock()
	for index := range k.ProjectionPopulations {
		if PopulationEquivalent(k.ProjectionPopulations[index], population) {
			k.ProjectionPopulations[index].Prototype = append([]float64(nil), updated.Prototype...)
			for unitIndex := range k.ProjectionPopulations[index].Units {
				for _, unit := range updated.Units {
					if k.ProjectionPopulations[index].Units[unitIndex].NodeID == unit.NodeID {
						k.ProjectionPopulations[index].Units[unitIndex].Activation = unit.Activation
						break
					}
				}
			}
			break
		}
	}
	k.projectionMu.Unlock()

	return updated, nil
}
