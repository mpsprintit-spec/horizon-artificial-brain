package runtime

import "time"

// inquiryPreferenceLearningRate controls how strongly a new consequence
// updates the action-specific experience signal. The signal is bounded and
// remains a runtime preference, not an authorization mechanism.
const inquiryPreferenceLearningRate = 0.20

func (r *BrainRuntime) RecordInquiryConsequence(action InquiryAction, valence float64) {
	_, _ = r.RecordInquiryConsequenceEvent(InquiryConsequenceEvent{Action: action, Valence: valence}, time.Time{})
}

// InquiryPriorExperience converts accumulated action valence into the
// existing PriorExperience dimension. It never creates a new decision score.
func (r *BrainRuntime) InquiryPriorExperience(action InquiryAction, baseline float64) float64 {
	baseline = clamp01(baseline)
	if r == nil || action == "" {
		return baseline
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	valence := r.inquiryValence[action]
	return clamp01(baseline + 0.25*valence)
}

func clampSigned(v float64) float64 {
	if v < -1 { return -1 }
	if v > 1 { return 1 }
	return v
}

// BuildAdaptiveInquiryAgenda applies learned consequence preference to the
// existing inquiry candidates, then delegates scoring to the canonical
// inquiry policy. It does not bypass safety or authorization.
func (r *BrainRuntime) BuildAdaptiveInquiryAgenda(interpretation CognitiveInterpretation, atTime time.Time) (InquiryAgenda, error) {
	candidates := DefaultInquiryCandidates(interpretation.Answer.Uncertainty.Level)
	for i := range candidates {
		candidates[i].PriorExperience = r.InquiryPriorExperience(candidates[i].Action, candidates[i].PriorExperience)
	}
	return BuildInquiryAgenda(
		interpretation.State.BrainIdentity,
		interpretation.State.Sequence,
		atTime,
		candidates,
		DefaultInquiryPolicy(),
	)
}
