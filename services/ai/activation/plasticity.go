package activation

import (
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

const (
	predictionLearningRate = 0.12
	predictionMinWeight    = 0.05
	predictionMaxWeight    = 0.95
)

// ApplyPredictionErrorPlasticity adapts existing neural connections from the
// mismatch between an internally predicted state and the state that actually
// occurred. It operates only on substrate dynamics; no semantic relation or
// cognitive rule is consulted.
func (e *Engine) ApplyPredictionErrorPlasticity(predicted, actual map[knowledge.NodeID]float64, errorSignal float64, now time.Time) {
	if e == nil || e.Memory == nil || errorSignal <= 0 {
		return
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

			eligibility := synapse.Dynamic.Eligibility
			if eligibility <= 0 {
				eligibility = clamp01(source.Activation * synapse.Activation)
			}
			if eligibility <= 0 {
				continue
			}

			magnitude := predictionLearningRate * errorSignal * clamp01(abs(direction)) * eligibility
			if direction > 0 {
				synapse.Dynamic.Weight = clamp(synapse.Dynamic.Weight+magnitude, predictionMinWeight, predictionMaxWeight)
				synapse.Dynamic.Confidence = clamp01(synapse.Dynamic.Confidence + magnitude*0.5)
			} else {
				synapse.Dynamic.Weight = clamp(synapse.Dynamic.Weight-magnitude, predictionMinWeight, predictionMaxWeight)
				synapse.Dynamic.Confidence = clamp01(synapse.Dynamic.Confidence - magnitude*0.5)
			}
			synapse.Dynamic.LastModification = now

			// Keep legacy fields synchronized while consumers migrate to Dynamic.
			synapse.Weight = synapse.Dynamic.Weight
			synapse.Confidence = synapse.Dynamic.Confidence
		}
	}
}
