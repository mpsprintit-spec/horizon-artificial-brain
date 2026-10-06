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
// measure. Information value is higher when the learned action has a
// differentiated outcome distribution: a predictable single outcome leaves
// less uncertainty to resolve, while multiple plausible outcomes create more
// opportunity for observation to reduce uncertainty. This is an
// uncertainty-weighted proxy for expected information gain, not a Bayesian
// posterior calculation. No semantic question is generated.
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

	r.mu.Lock()
	targetSet := r.inquiryTargetSetLocked(action)
	r.mu.Unlock()
	return r.predictInquiryInformationValueWithTargets(targetSet, uncertainty, at)
}

func (r *BrainRuntime) inquiryTargetSetLocked(action InquiryAction) map[knowledge.NodeID]struct{} {
	targetSet := make(map[knowledge.NodeID]struct{})
	intent := "inquiry:" + string(action)
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
	return targetSet
}

func (r *BrainRuntime) predictInquiryInformationValueWithTargets(targetSet map[knowledge.NodeID]struct{}, uncertainty float64, at time.Time) (float64, bool, error) {
	if len(targetSet) == 0 {
		return uncertainty, false, nil
	}

	targets := make([]knowledge.NodeID, 0, len(targetSet))
	for id := range targetSet {
		targets = append(targets, id)
	}
	sort.Slice(targets, func(i, j int) bool { return targets[i] < targets[j] })

	r.brain.RLock()
	prediction := r.activation.PredictOutcomeFromNodes(targets, at, 1)

	// Restrict the outcome distribution to neural units that are actually
	// reachable through the learned action -> outcome channels. The recurrent
	// predictor may contain unrelated active state; allowing that state into
	// the entropy would turn unrelated activity into apparent information
	// value for this action.
	outcomeSet := make(map[knowledge.NodeID]struct{})
	for _, targetID := range targets {
		target := r.brain.Registry.GetByID(targetID)
		if target == nil {
			continue
		}
		for _, synapse := range target.OutboundAll() {
			if synapse.Inhibitory {
				continue
			}
			if prediction.State[synapse.TargetID] <= 0 {
				continue
			}
			outcomeSet[synapse.TargetID] = struct{}{}
		}
	}
	if len(outcomeSet) == 0 {
		return 0, true, nil
	}

	// Use the learned target -> outcome channel statistics for the outcome
	// distribution. Recurrent activation determines reachability above; the
	// empirical channel frequency/weight determines how strongly each outcome
	// is represented by prior experience.
	channelState := make(map[knowledge.NodeID]float64)
	for targetID := range targetSet {
		target := r.brain.Registry.GetByID(targetID)
		if target == nil { continue }
		for _, synapse := range target.OutboundAll() {
			if synapse == nil || synapse.Inhibitory { continue }
			if prediction.State[synapse.TargetID] <= 0 { continue }
			strength := synapse.Weight
			if strength <= 0 { strength = synapse.Dynamic.Weight }
			frequency := float64(synapse.Frequency)
			if frequency <= 0 { frequency = float64(synapse.Dynamic.Frequency) }
			confidence := synapse.Confidence
			if confidence <= 0 { confidence = synapse.Dynamic.Confidence }
			if frequency <= 0 { frequency = 1 }
			if confidence <= 0 { confidence = 1 }
			channelState[synapse.TargetID] += clamp01(strength) * frequency * clamp01(confidence)
		}
	}
	r.brain.RUnlock()
	uncertaintyOfOutcome := normalizedStateEntropy(channelState)
	yield, reliability, _, learned := r.brain.InquiryInformationExperience(string(action))
	if !learned {
		return 0, true, nil
	}
	actionYield := yield * (0.25 + 0.75*reliability)
	informationValue := uncertainty * uncertaintyOfOutcome * actionYield
	return clamp01(informationValue), true, nil
}

func normalizedStateEntropy(state map[knowledge.NodeID]float64) float64 {
	positive := make([]float64, 0, len(state))
	for _, value := range state {
		if value > 0 { positive = append(positive, value) }
	}
	if len(positive) < 2 { return 0 }
	total := 0.0
	for _, value := range positive { total += value }
	if total <= 0 { return 0 }
	entropy := 0.0
	for _, value := range positive {
		p := value / total
		entropy -= p * math.Log(p)
	}
	normalizer := math.Log(float64(len(positive)))
	if normalizer <= 0 { return 0 }
	return clamp01(entropy / normalizer)
}

func normalizedPredictionEntropyAllowed(state map[knowledge.NodeID]float64, allowed map[knowledge.NodeID]struct{}) float64 {
	if len(allowed) < 2 {
		return 0
	}

	total := 0.0
	outcomeCount := 0
	for id := range allowed {
		value := state[id]
		if value <= 0 {
			continue
		}
		total += value
		outcomeCount++
	}
	if total <= 0 || outcomeCount < 2 {
		return 0
	}

	entropy := 0.0
	for id := range allowed {
		value := state[id]
		if value <= 0 {
			continue
		}
		p := value / total
		entropy -= p * math.Log(p)
	}
	normalizer := math.Log(float64(outcomeCount))
	if normalizer <= 0 {
		return 0
	}
	return clamp01(entropy / normalizer)
}

func normalizedPredictionEntropyExcluding(state map[knowledge.NodeID]float64, excluded map[knowledge.NodeID]struct{}) float64 {
	if len(state) < 2 {
		return 0
	}

	total := 0.0
	outcomeCount := 0
	for id, value := range state {
		if _, skip := excluded[id]; skip {
			continue
		}
		if value > 0 {
			total += value
			outcomeCount++
		}
	}
	if total <= 0 || outcomeCount < 2 {
		return 0
	}

	entropy := 0.0
	for id, value := range state {
		if _, skip := excluded[id]; skip || value <= 0 {
			continue
		}
		p := value / total
		entropy -= p * math.Log(p)
	}
	normalizer := math.Log(float64(outcomeCount))
	if normalizer <= 0 {
		return 0
	}
	return clamp01(entropy / normalizer)
}

