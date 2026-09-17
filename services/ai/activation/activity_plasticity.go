package activation

import (
	"math"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

const (
	activityPlasticityLearningRate = 0.08
	activityPlasticityDecayRate    = 0.015
	activityPlasticityMinWeight    = 0.02
	activityPlasticityMaxWeight    = 0.95
	activityPlasticityMinActive    = 0.05

	// Eligibility decays continuously with elapsed time. A one-second
	// half-life keeps recent co-activity relevant while allowing traces to
	// fade during longer idle periods without deleting the synapse.
	activityPlasticityEligibilityHalfLife = time.Second
)

func decayEligibility(eligibility float64, lastUpdate, now time.Time) float64 {
	eligibility = clamp01(eligibility)
	if eligibility <= 0 || lastUpdate.IsZero() || !now.After(lastUpdate) {
		return eligibility
	}
	elapsed := now.Sub(lastUpdate)
	factor := math.Exp(-math.Ln2 * elapsed.Seconds() / activityPlasticityEligibilityHalfLife.Seconds())
	return clamp01(eligibility * factor)
}

// ApplyActivityPlasticity adapts existing recurrent synapses from actual
// pre/post neural activity. It does not inspect tokens, RelationKind, or any
// semantic rule. Co-active pathways are strengthened through a time-decaying
// eligibility trace; unused pathways decay without immediate deletion.
func (e *Engine) ApplyActivityPlasticity(preState, postState map[knowledge.NodeID]float64, now time.Time) {
	if e == nil || e.Memory == nil {
		return
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	for _, source := range e.Memory.Registry.Nodes() {
		pre := clamp01(preState[source.ID])
		for _, synapse := range source.OutboundAll() {
			if synapse == nil {
				continue
			}

			target := clamp01(postState[synapse.TargetID])
			coactivity := pre * target
			eligibility := decayEligibility(synapse.Dynamic.Eligibility, synapse.Dynamic.LastEligibilityUpdate, now)

			// Eligibility is a continuous temporal trace. New co-activity is
			// accumulated after elapsed-time decay, replacing call-count decay.
			eligibility = clamp01(eligibility*0.85 + coactivity*0.15)
			synapse.Dynamic.Eligibility = eligibility
			synapse.Dynamic.LastEligibilityUpdate = now

			if synapse.Inhibitory {
				if coactivity >= activityPlasticityMinActive {
					delta := activityPlasticityLearningRate * coactivity * eligibility
					synapse.Dynamic.Weight = clamp(synapse.Dynamic.Weight+delta, activityPlasticityMinWeight, activityPlasticityMaxWeight)
					synapse.Dynamic.LastModification = now
				}
				synapse.Weight = synapse.Dynamic.Weight
				continue
			}

			if coactivity >= activityPlasticityMinActive {
				delta := activityPlasticityLearningRate * coactivity * eligibility
				synapse.Dynamic.Weight = clamp(synapse.Dynamic.Weight+delta, activityPlasticityMinWeight, activityPlasticityMaxWeight)
				synapse.Dynamic.Confidence = clamp01(synapse.Dynamic.Confidence + delta*0.5)
				synapse.Dynamic.Frequency++
				synapse.Dynamic.LastActivation = now
				synapse.Dynamic.LastModification = now
			} else {
				decay := activityPlasticityDecayRate * (1 - eligibility)
				synapse.Dynamic.Weight = clamp(synapse.Dynamic.Weight-decay, activityPlasticityMinWeight, activityPlasticityMaxWeight)
				synapse.Dynamic.Confidence = clamp01(synapse.Dynamic.Confidence - decay*0.25)
			}

			synapse.Weight = synapse.Dynamic.Weight
			synapse.Confidence = synapse.Dynamic.Confidence
		}
	}
}
