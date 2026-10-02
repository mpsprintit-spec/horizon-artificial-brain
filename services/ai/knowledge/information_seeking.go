package knowledge

import (
	"sort"
)

// InquiryCandidate is a substrate-level candidate for acquiring experience.
// It contains no question text and no semantic command. The producer supplies
// the predicted outcome distribution of an available candidate experience.
type InquiryCandidate struct {
	ActionID       string
	PredictedStates []map[NodeID]float64
	OutcomeProbabilities []float64
	Novelty        float64
	Repeatability  float64
	Sequence       uint64
}

// InformationSeekingScore is the numeric pressure for selecting a candidate
// experience. Higher values indicate more expected reduction of uncertainty.
func InformationSeekingScore(c InquiryCandidate, currentUncertainty float64, explorationBias float64, repeatObservationBias float64) float64 {
	currentUncertainty = clamp01(currentUncertainty)
	explorationBias = clamp01(explorationBias)
	repeatObservationBias = clamp01(repeatObservationBias)
	if len(c.PredictedStates) == 0 {
		return 0
	}

	// When alternatives are available, expected disagreement between predicted
	// outcomes is an epistemic value signal. It is deliberately numeric: no
	// semantic question or curiosity label is introduced.
	mean := 0.0
	for _, p := range c.OutcomeProbabilities {
		if p > 0 {
			mean += p
		}
	}
	if mean <= 0 {
		mean = float64(len(c.PredictedStates))
	}
	spread := 0.0
	for i, state := range c.PredictedStates {
		weight := 1.0 / float64(len(c.PredictedStates))
		if i < len(c.OutcomeProbabilities) && c.OutcomeProbabilities[i] > 0 {
			weight = c.OutcomeProbabilities[i] / mean
		}
		spread += weight * stateDistanceFromMean(state, c.PredictedStates)
	}
	spread = clamp01(spread)
	novelty := clamp01(c.Novelty)
	repeatPenalty := clamp01(c.Repeatability) * repeatObservationBias
	return clamp01(currentUncertainty * (0.5*spread + 0.5*novelty) * (1+explorationBias) - repeatPenalty)
}

func stateDistanceFromMean(state map[NodeID]float64, states []map[NodeID]float64) float64 {
	if len(states) < 2 {
		return 0
	}
	ids := make(map[NodeID]struct{})
	for _, s := range states {
		for id := range s {
			ids[id] = struct{}{}
		}
	}
	ordered := make([]int, 0, len(ids))
	for id := range ids {
		ordered = append(ordered, int(id))
	}
	sort.Ints(ordered)

	distance := 0.0
	for _, raw := range ordered {
		id := NodeID(raw)
		mean := 0.0
		for _, s := range states {
			mean += clamp01(s[id])
		}
		mean /= float64(len(states))
		d := clamp01(state[id]) - mean
		if d < 0 {
			d = -d
		}
		distance += d
	}
	return clamp01(distance / float64(len(ordered)))
}

// SelectInformationSeekingCandidate deterministically selects the candidate
// with the greatest information value. Ties are resolved by ActionID and then
// Sequence, so replay does not depend on map or goroutine ordering.
func SelectInformationSeekingCandidate(candidates []InquiryCandidate, currentUncertainty, explorationBias, repeatObservationBias float64) (InquiryCandidate, float64, bool) {
	if len(candidates) == 0 {
		return InquiryCandidate{}, 0, false
	}
	best := InquiryCandidate{}
	bestScore := -1.0
	for _, candidate := range candidates {
		score := InformationSeekingScore(candidate, currentUncertainty, explorationBias, repeatObservationBias)
		if score > bestScore || (score == bestScore && (candidate.ActionID < best.ActionID || (candidate.ActionID == best.ActionID && candidate.Sequence < best.Sequence))) {
			best = candidate
			bestScore = score
		}
	}
	return best, clamp01(bestScore), true
}

// The selected candidate is persisted through KnowledgeBase.RecordInquirySelection
// in knowledge.go. Keeping persistence there avoids two competing inquiry
// state writers and preserves the existing sequence/uncertainty fields.
