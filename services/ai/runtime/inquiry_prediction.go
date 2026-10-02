package runtime

import (
	"errors"
	"math"
	"sort"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// PredictInquiryInformationValue estimates information value from Horizon's
// learned action/outcome model. The prediction is hypothetical: it does not
// authorize, execute, or mutate the action or neural state.
//
// A learned action is represented by prior action bindings. Their target
// populations are injected into the existing activation predictor, which then
// applies the existing learned temporal/pattern prediction. The resulting
// distributed state is converted into a bounded entropy-like uncertainty
// measure. No semantic question is generated.
func (r *BrainRuntime) PredictInquiryInformationValue(action InquiryAction, uncertainty float64, at time.Time) (float64, bool, error) {
	if r == nil || r.activation == nil {
		return 0, false, errors.New("brain runtime is not initialized")
	}
	if action == "" {
		return 0, false, errors.New("inquiry action is required")
	}
	uncertainty = clamp01(uncertainty)
	if at.IsZero() {
		at = r.now()
	}

	targetSet := make(map[knowledge.NodeID]struct{})
	intent := "inquiry:" + string(action)

	r.mu.Lock()
	for _, binding := range r.actions {
		if binding.Intent != intent {
			continue
		}
		for _, id := range binding.TargetNodeIDs {
			targetSet[id] = struct{}{}
		}
		for _, synapse := range binding.Synapses {
			targetSet[synapse.TargetNodeID] = struct{}{}
		}
	}
	r.mu.Unlock()

	if len(targetSet) == 0 {
		return uncertainty, false, nil
	}

	targets := make([]knowledge.NodeID, 0, len(targetSet))
	for id := range targetSet {
		targets = append(targets, id)
	}
	sort.Slice(targets, func(i, j int) bool { return targets[i] < targets[j] })

	prediction := r.activation.PredictOutcomeFromNodes(targets, at, 1)
	value := normalizedPredictionEntropy(prediction.State)
	return clamp01(uncertainty * value), true, nil
}

func normalizedPredictionEntropy(state map[knowledge.NodeID]float64) float64 {
	if len(state) < 2 {
		return 0
	}

	total := 0.0
	for _, value := range state {
		if value > 0 {
			total += value
		}
	}
	if total <= 0 {
		return 0
	}

	entropy := 0.0
	for _, value := range state {
		if value <= 0 {
			continue
		}
		p := value / total
		entropy -= p * math.Log(p)
	}
	normalizer := math.Log(float64(len(state)))
	if normalizer <= 0 {
		return 0
	}
	return clamp01(entropy / normalizer)
}
