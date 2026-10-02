package knowledge

import "time"

// InformationDynamics is the numeric state produced by prediction/observation
// dynamics. It is not a semantic category and does not encode a question.
// The values describe mismatch, novelty, instability, and their resulting
// pressure to acquire information.
type InformationDynamics struct {
	ErrorEMA       float64
	ErrorVariance  float64
	Novelty        float64
	Uncertainty    float64
	InformationValue float64
	Samples        uint64
}

// UpdateInformationDynamics updates the internal information-seeking pressure
// from prediction error and the observed distributed representation.
//
// No semantic label, question, or hard-coded inquiry text is created here.
// Uncertainty comes from persistent mismatch, representational novelty, and
// instability of the prediction-error stream. InformationValue is the current
// numeric value of obtaining more evidence about the active state.
func (k *KnowledgeBase) UpdateInformationDynamics(
	predicted, actual map[NodeID]float64,
	predictionError float64,
	now time.Time,
) InformationDynamics {
	if k == nil {
		return InformationDynamics{}
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	predictionError = clamp01(predictionError)
	k.mu.Lock()
	defer k.mu.Unlock()

	actualVector, actualOK := weightedRepresentationVector(k.Registry, actual)
	if !actualOK || actualVector.Empty() {
		return k.informationDynamicsLocked()
	}

	novelty := k.representationNoveltyLocked(actualVector)
	state := k.BrainState.PredictionState
	if state == nil {
		state = map[string]float64{}
		k.BrainState.PredictionState = state
	}

	samples := uint64(state["information_samples"])
	previousEMA := clamp01(state["error_ema"])
	nextSamples := samples + 1

	// Early observations adapt quickly; later observations retain a longer
	// history. The bounded denominator also makes replay numerically stable.
	window := float64(nextSamples)
	if window > 16 {
		window = 16
	}
	alpha := 1.0 / window
	errorEMA := previousEMA + alpha*(predictionError-previousEMA)

	previousMean := state["error_mean"]
	previousM2 := state["error_m2"]
	delta := predictionError - previousMean
	mean := previousMean + delta/float64(nextSamples)
	delta2 := predictionError - mean
	m2 := previousM2 + delta*delta2
	variance := 0.0
	if nextSamples > 1 {
		variance = m2 / float64(nextSamples)
	}
	if variance < 0 {
		variance = 0
	}

	policy := k.BrainState.LearningPolicyState
	uncertaintyWeight := clamp01(policy.UncertaintySensitivity)
	noveltyWeight := clamp01(policy.NoveltySensitivity)
	pressureWeight := clamp01(policy.CuriosityPressure)
	weightSum := uncertaintyWeight + noveltyWeight + pressureWeight
	if weightSum <= 0 {
		weightSum = 1
	}

	instability := clamp01(sqrtNonNegative(variance))
	uncertainty := clamp01(
		(errorEMA*uncertaintyWeight+
			novelty*noveltyWeight+
			instability*pressureWeight) / weightSum,
	)
	informationValue := clamp01(
		(predictionError*uncertaintyWeight+
			novelty*noveltyWeight+
			instability*pressureWeight) / weightSum,
	)

	state["error_ema"] = errorEMA
	state["error_mean"] = mean
	state["error_m2"] = m2
	state["error_variance"] = variance
	state["novelty"] = novelty
	state["uncertainty"] = uncertainty
	state["information_value"] = informationValue
	state["information_samples"] = float64(nextSamples)

	// Inquiry consumes this numeric boundary later. It does not create the
	// boundary and does not turn it into a verbal question.
	k.BrainState.InquiryState.Uncertainty = uncertainty
	k.BrainState.InquiryState.ExpectedInformationGain = informationValue
	k.BrainState.InquiryState.LastUpdated = now

	return InformationDynamics{
		ErrorEMA: errorEMA,
		ErrorVariance: variance,
		Novelty: novelty,
		Uncertainty: uncertainty,
		InformationValue: informationValue,
		Samples: nextSamples,
	}
}

func (k *KnowledgeBase) informationDynamicsLocked() InformationDynamics {
	if k == nil {
		return InformationDynamics{}
	}
	state := k.BrainState.PredictionState
	if state == nil {
		return InformationDynamics{}
	}
	samples := uint64(state["information_samples"])
	return InformationDynamics{
		ErrorEMA: clamp01(state["error_ema"]),
		ErrorVariance: clamp01(state["error_variance"]),
		Novelty: clamp01(state["novelty"]),
		Uncertainty: clamp01(state["uncertainty"]),
		InformationValue: clamp01(state["information_value"]),
		Samples: samples,
	}
}

// InformationDynamics returns the last persisted numeric information state.
func (k *KnowledgeBase) InformationDynamics() InformationDynamics {
	if k == nil {
		return InformationDynamics{}
	}
	k.mu.RLock()
	defer k.mu.RUnlock()
	return k.informationDynamicsLocked()
}

func (k *KnowledgeBase) representationNoveltyLocked(vector NeuralVector) float64 {
	bestSimilarity := -1.0
	k.projectionMu.RLock()
	defer k.projectionMu.RUnlock()

	for _, population := range k.ProjectionPopulations {
		if len(population.Prototype) != len(vector.Values) {
			continue
		}
		similarity := NewNeuralVector(population.Prototype).Similarity(vector)
		if similarity > bestSimilarity {
			bestSimilarity = similarity
		}
	}

	if bestSimilarity < -0.999999 {
		return 1
	}
	return clamp01((1 - clamp(bestSimilarity, -1, 1)) / 2)
}

func sqrtNonNegative(value float64) float64 {
	if value <= 0 {
		return 0
	}
	// Newton iteration avoids introducing a second dependency for one scalar
	// operation and is deterministic for the same IEEE-754 input.
	x := value
	for i := 0; i < 12; i++ {
		x = 0.5 * (x + value/x)
	}
	return x
}
