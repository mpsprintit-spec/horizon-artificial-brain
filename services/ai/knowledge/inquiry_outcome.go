package knowledge

import (
	"sort"
	"time"
)

// InquiryOutcome is the numeric observation produced after an information-seeking
// action. It contains no semantic question or textual interpretation.
type InquiryOutcome struct {
	State map[NodeID]float64
	At    time.Time
}

// EvaluateInquiryOutcome compares the observed state with the candidate's
// predicted outcome distribution. The result is a bounded numeric surprise
// signal and prediction error that can close the inquiry-learning loop.
func EvaluateInquiryOutcome(candidate InquiryCandidate, actual InquiryOutcome) (observedInformationGain, predictionError float64) {
	if len(candidate.PredictedStates) == 0 || len(actual.State) == 0 {
		return 0, 0
	}
	predicted := weightedStateMean(candidate.PredictedStates, candidate.OutcomeProbabilities)
	error := stateDistance(actual.State, predicted)
	observedInformationGain = error

	// A candidate with a wider predicted distribution should tolerate more
	// variation. Prediction error therefore measures the observed distance
	// relative to the candidate's predicted spread.
	spread := 0.0
	for _, state := range candidate.PredictedStates {
		spread += stateDistance(state, predicted)
	}
	spread /= float64(len(candidate.PredictedStates))
	if spread > 0 {
		predictionError = clamp01(error / (error + spread))
	} else {
		predictionError = error
	}
	return clamp01(observedInformationGain), clamp01(predictionError)
}

func weightedStateMean(states []map[NodeID]float64, probabilities []float64) map[NodeID]float64 {
	ids := make(map[NodeID]struct{})
	for _, state := range states {
		for id := range state {
			ids[id] = struct{}{}
		}
	}
	ordered := make([]int, 0, len(ids))
	for id := range ids {
		ordered = append(ordered, int(id))
	}
	sort.Ints(ordered)

	mean := make(map[NodeID]float64, len(ordered))
	weightTotal := 0.0
	for i := range states {
		weight := 1.0
		if i < len(probabilities) && probabilities[i] > 0 {
			weight = probabilities[i]
		}
		weightTotal += weight
		for _, raw := range ordered {
			id := NodeID(raw)
			mean[id] += weight * states[i][id]
		}
	}
	if weightTotal == 0 {
		return mean
	}
	for _, raw := range ordered {
		id := NodeID(raw)
		mean[id] = clamp01(mean[id] / weightTotal)
	}
	return mean
}

func stateDistance(a, b map[NodeID]float64) float64 {
	ids := make(map[NodeID]struct{})
	for id := range a { ids[id] = struct{}{} }
	for id := range b { ids[id] = struct{}{} }
	if len(ids) == 0 { return 0 }

	ordered := make([]int, 0, len(ids))
	for id := range ids { ordered = append(ordered, int(id)) }
	sort.Ints(ordered)

	total := 0.0
	for _, raw := range ordered {
		id := NodeID(raw)
		d := a[id] - b[id]
		if d < 0 { d = -d }
		total += d
	}
	return clamp01(total / float64(len(ordered)))
}
