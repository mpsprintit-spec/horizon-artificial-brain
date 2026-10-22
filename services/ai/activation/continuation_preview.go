package activation

import (
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// PreviewLearnedPopulation projects an explicit hypothetical prefix through
// the existing recurrent substrate. Callers must serialize access to Memory
// and bound/validate the prefix (the runtime continuation API does both).
// The prefix is never observed: no plasticity, context update, extra evidence,
// clock advance, or replacement of the real-observation forecast occurs.
func (e *Engine) PreviewLearnedPopulation(prefix *knowledge.ObservationTrace, at time.Time) *knowledge.PredictionAssessment {
	if e == nil || e.Memory == nil || prefix == nil || len(prefix.Stimulus) == 0 || at.IsZero() {
		return knowledge.UnknownPredictionAssessment()
	}
	prefix = prefix.Canonical()
	e.mu.RLock()
	state, confidence := cloneState(e.internalState), cloneState(e.internalConfidence)
	history := e.contextState.Clone().before(at, false)
	e.mu.RUnlock()

	step := 0
	for _, stream := range [][][]knowledge.NodeID{prefix.Context, prefix.Data, prefix.Stimulus} {
		for _, frame := range stream {
			if step > 0 {
				state, confidence = e.advanceHypotheticalWithSynapseOrder(state, confidence, at, 1, true)
			}
			for _, id := range frame {
				if node := e.Memory.Registry.GetByID(id); node != nil {
					state[id] = max(state[id], 1)
					confidence[id] = max(confidence[id], .5*clamp01(node.Importance))
				}
			}
			step++
		}
	}
	state = normalize(state)
	state, confidence = e.advanceHypotheticalWithSynapseOrder(state, confidence, at, 1, true)
	predicted, predictedConfidence := e.advanceHypotheticalWithSynapseOrder(state, confidence, at, 1, true)
	channels := e.Memory.Patterns.PredictRecurrent(prefix, history, at)
	predicted, _ = e.applyPatternChannels(predicted, predictedConfidence, channels)
	return e.Memory.AssessObservedPrediction(prefix.Stimulus[len(prefix.Stimulus)-1], channels.Credit, predicted)
}
