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

// ApplyActivityPlasticity preserves the historical call-based inactive decay
// for old journals and trusted legacy callers. Serving supplies its recorded
// dynamics explicitly through ApplyActivityPlasticityWithDynamics.
func (e *Engine) ApplyActivityPlasticity(preState, postState map[knowledge.NodeID]float64, now time.Time) {
	e.applyActivityPlasticity(preState, postState, now, false)
}

// ApplyActivityPlasticityWithDynamics separates passive retention from actual
// co-activity. An unrelated/empty experience supplies no evidence against an
// inactive connection. Prediction error and consequences still adapt eligible
// paths in either direction; this does not freeze previously learned weights.
func (e *Engine) ApplyActivityPlasticityWithDynamics(preState, postState map[knowledge.NodeID]float64, now time.Time, dynamics knowledge.DynamicsEncoding) {
	e.applyActivityPlasticity(preState, postState, now, dynamics.HasLocalActivity())
}

func (e *Engine) applyActivityPlasticity(preState, postState map[knowledge.NodeID]float64, now time.Time, local bool) {
	if e == nil || e.Memory == nil {
		return
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	e.Memory.Lock()
	defer e.Memory.Unlock()

	for _, source := range e.Memory.Registry.Nodes() {
		pre := clamp01(preState[source.ID])
		for _, synapse := range source.OutboundAll() {
			if synapse == nil {
				continue
			}

			target := clamp01(postState[synapse.TargetID])
			coactivity := pre * target
			eligibility := decayEligibility(synapse.Dynamic.Eligibility, synapse.Dynamic.LastEligibilityUpdate, now)

			if local {
				// Only elapsed time forgets eligibility. Actual co-activity
				// adds bounded evidence; empty calls cannot multiply it away.
				eligibility = clamp01(eligibility + .15*coactivity*(1-eligibility))
			} else {
				eligibility = clamp01(eligibility*0.85 + coactivity*0.15)
			}
			synapse.Dynamic.Eligibility = eligibility
			if !local || synapse.Dynamic.LastEligibilityUpdate.IsZero() || now.After(synapse.Dynamic.LastEligibilityUpdate) {
				synapse.Dynamic.LastEligibilityUpdate = now
			}

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
			} else if !local {
				decay := activityPlasticityDecayRate * (1 - eligibility)
				synapse.Dynamic.Weight = clamp(synapse.Dynamic.Weight-decay, activityPlasticityMinWeight, activityPlasticityMaxWeight)
				synapse.Dynamic.Confidence = clamp01(synapse.Dynamic.Confidence - decay*0.25)
			}

			synapse.Weight = synapse.Dynamic.Weight
			synapse.Confidence = synapse.Dynamic.Confidence
		}
	}
}
