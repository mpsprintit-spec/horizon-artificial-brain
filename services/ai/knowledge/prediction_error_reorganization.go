package knowledge

import "time"

// ApplyPredictionErrorRepresentationPlasticity converts a prediction mismatch
// into distributed representation drift. The target state is reconstructed from
// the currently active neural substrate; no semantic label or hand-authored
// meaning is introduced. Only populations participating in the observed state
// are eligible to reorganize.
func (k *KnowledgeBase) ApplyPredictionErrorRepresentationPlasticity(
	predicted, actual map[NodeID]float64,
	errorSignal, learningRate float64,
	now time.Time,
) {
	if k == nil || len(actual) == 0 || errorSignal <= 0 {
		return
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	errorSignal = clamp01(errorSignal)
	learningRate = clamp01(learningRate)
	if learningRate == 0 {
		return
	}

	k.mu.Lock()
	defer k.mu.Unlock()

	actualVector, ok := weightedRepresentationVector(k.Registry, actual)
	if !ok || actualVector.Empty() {
		return
	}
	predictedVector, predictedOK := weightedRepresentationVector(k.Registry, predicted)
	if !predictedOK || predictedVector.Empty() || len(predictedVector.Values) != len(actualVector.Values) {
		return
	}

	// Prediction error has a direction as well as a magnitude. The direction
	// is the learned discrepancy between the predicted distributed state and
	// the observed distributed state; plasticity follows that error vector.
	errorVector := make([]float64, len(actualVector.Values))
	errorMagnitude := 0.0
	for dimension := range errorVector {
		delta := actualVector.Values[dimension] - predictedVector.Values[dimension]
		errorVector[dimension] = delta
		errorMagnitude += delta * delta
	}
	if errorMagnitude <= 1e-12 {
		return
	}

	// A large mismatch should produce more reorganization, but remain bounded.
	rate := learningRate * errorSignal
	if rate <= 0 {
		return
	}

	k.projectionMu.Lock()
	defer k.projectionMu.Unlock()

	for index := range k.ProjectionPopulations {
		population := &k.ProjectionPopulations[index]
		participation := populationParticipation(population, actual)
		if participation <= 0 {
			continue
		}

		effectiveRate := clamp01(rate * participation)

		if len(population.Prototype) != len(actualVector.Values) {
			population.Prototype = append([]float64(nil), actualVector.Values...)
		} else {
			for dimension, value := range actualVector.Values {
				population.Prototype[dimension] = clamp(
					population.Prototype[dimension]+effectiveRate*errorVector[dimension],
					-1, 1,
				)
			}
		}

		for unitIndex := range population.Units {
			unit := &population.Units[unitIndex]
			node := k.Registry.byID[unit.NodeID]
			if node == nil || len(node.Representation) != len(actualVector.Values) {
				continue
			}

			activation := clamp01(actual[unit.NodeID])
			if activation <= 0 {
				activation = clamp01(unit.Activation)
			}
			unitRate := effectiveRate * clamp01(node.Plasticity) * clamp01(activation)
			if unitRate <= 0 {
				continue
			}

			for dimension, value := range actualVector.Values {
				node.Representation[dimension] = clamp(
					node.Representation[dimension]+unitRate*errorVector[dimension],
					-1, 1,
				)
			}
			node.Activation = clamp01(NewNeuralVector(node.Representation).Similarity(actualVector))
			node.LastActivation = now
			node.Frequency++
			node.UsageHistory = append(node.UsageHistory, now)
			unit.Activation = node.Activation
		}
	}
}

// weightedRepresentationVector reconstructs a modality-neutral state vector
// from the active distributed substrate. It is deliberately a numeric
// aggregation: semantic interpretation remains a property of learned network
// dynamics, not this plasticity mechanism.
func weightedRepresentationVector(registry *NeuralRegistry, state map[NodeID]float64) (NeuralVector, bool) {
	if registry == nil {
		return NeuralVector{}, false
	}

	var values []float64
	total := 0.0
	for id, activation := range state {
		activation = clamp01(activation)
		if activation <= 0 {
			continue
		}
		node := registry.GetByID(id)
		if node == nil || len(node.Representation) == 0 {
			continue
		}
		if values == nil {
			values = make([]float64, len(node.Representation))
		}
		if len(values) != len(node.Representation) {
			continue
		}
		for dimension, value := range node.Representation {
			values[dimension] += value * activation
		}
		total += activation
	}
	if total <= 0 || len(values) == 0 {
		return NeuralVector{}, false
	}
	for dimension := range values {
		values[dimension] /= total
	}
	return NewNeuralVector(values), true
}

func populationParticipation(population *ProjectionPopulation, actual map[NodeID]float64) float64 {
	if population == nil {
		return 0
	}
	total := 0.0
	count := 0
	for _, unit := range population.Units {
		activation := clamp01(actual[unit.NodeID])
		if activation <= 0 {
			continue
		}
		total += activation
		count++
	}
	if count == 0 {
		return 0
	}
	return clamp01(total / float64(count))
}
