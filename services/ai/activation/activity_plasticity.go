package activation

import (
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

const (
	activityPlasticityLearningRate = 0.08
	activityPlasticityDecayRate    = 0.015
	activityPlasticityMinWeight   = 0.02
	activityPlasticityMaxWeight   = 0.95
	activityPlasticityMinActive   = 0.05
)

// ApplyActivityPlasticity adapts existing recurrent synapses from actual
// pre/post neural activity. It does not inspect tokens, RelationKind, or any
// semantic rule. Co-active pathways are gradually strengthened through their
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
			eligibility := synapse.Dynamic.Eligibility

			// Eligibility is a temporal trace: recent co-activity leaves a
			// decaying trace that can support a later plasticity event.
			eligibility = clamp01(eligibility*0.85 + coactivity*0.15)
			synapse.Dynamic.Eligibility = eligibility

			if synapse.Inhibitory {
				// Inhibitory pathways retain adaptive eligibility but use a
				// bounded activity-dependent update in the opposite direction.
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
				// Decay is deliberately slow and never means immediate deletion.
				decay := activityPlasticityDecayRate * (1 - eligibility)
				synapse.Dynamic.Weight = clamp(synapse.Dynamic.Weight-decay, activityPlasticityMinWeight, activityPlasticityMaxWeight)
				synapse.Dynamic.Confidence = clamp01(synapse.Dynamic.Confidence - decay*0.25)
			}

			synapse.Weight = synapse.Dynamic.Weight
			synapse.Confidence = synapse.Dynamic.Confidence
		}
	}
}
