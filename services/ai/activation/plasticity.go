package activation

import (
	"math"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

const (
	predictionLearningRate = 0.12
	predictionMinWeight    = 0.05
	predictionMaxWeight    = 0.95
)

func decayPredictionEligibility(eligibility float64, lastUpdate, now time.Time) float64 {
	eligibility = clamp01(eligibility)
	if eligibility <= 0 || lastUpdate.IsZero() || !now.After(lastUpdate) {
		return eligibility
	}
	const halfLife = time.Second
	factor := math.Exp(-math.Ln2 * now.Sub(lastUpdate).Seconds() / halfLife)
	return clamp01(eligibility * factor)
}

// ApplyPredictionErrorPlasticity adapts existing neural connections from the
// mismatch between an internally predicted state and the state that actually
// occurred. It operates only on substrate dynamics; no semantic relation or
// cognitive rule is consulted. Eligibility is evaluated at the supplied
// timestamp using the same elapsed-time decay model as activity plasticity.
func (e *Engine) ApplyPredictionErrorPlasticity(predicted, actual map[knowledge.NodeID]float64, errorSignal float64, now time.Time) {
	if e == nil || e.Memory == nil || errorSignal <= 0 {
		return
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	errorSignal = clamp01(errorSignal)

	for _, source := range e.Memory.Registry.Nodes() {
		for _, synapse := range source.OutboundAll() {
			if synapse == nil || synapse.Inhibitory {
				continue
			}
			targetActual := actual[synapse.TargetID]
			targetPredicted := predicted[synapse.TargetID]
			direction := targetActual - targetPredicted
			if abs(direction) < 0.001 {
				continue
			}

			eligibility := decayPredictionEligibility(synapse.Dynamic.Eligibility, synapse.Dynamic.LastEligibilityUpdate, now)
			if eligibility <= 0 {
				eligibility = clamp01(source.Activation * synapse.Activation)
			}
			if eligibility <= 0 {
				continue
			}

			synapse.Dynamic.Eligibility = eligibility
			synapse.Dynamic.LastEligibilityUpdate = now
			magnitude := predictionLearningRate * errorSignal * clamp01(abs(direction)) * eligibility
			if direction > 0 {
				synapse.Dynamic.Weight = clamp(synapse.Dynamic.Weight+magnitude, predictionMinWeight, predictionMaxWeight)
				synapse.Dynamic.Confidence = clamp01(synapse.Dynamic.Confidence + magnitude*0.5)
			} else {
				synapse.Dynamic.Weight = clamp(synapse.Dynamic.Weight-magnitude, predictionMinWeight, predictionMaxWeight)
				synapse.Dynamic.Confidence = clamp01(synapse.Dynamic.Confidence - magnitude*0.5)
			}
			synapse.Dynamic.LastModification = now

			synapse.Weight = synapse.Dynamic.Weight
			synapse.Confidence = synapse.Dynamic.Confidence
		}
	}

	// Prediction error also informs structural competition. This is a bounded
	// reconsolidation signal, not a semantic contradiction detector: competing
	// traces remain in the same Brain and retain their evidence history.
	e.Memory.Patterns.ReconsolidateFromState(predicted, actual, errorSignal)
}
