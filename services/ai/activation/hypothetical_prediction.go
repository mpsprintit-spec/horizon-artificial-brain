package activation

import (
	"sort"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// CurrentState returns a copy of the recurrent neural state. It is a
// read-only observation boundary; callers cannot mutate the engine state.
func (e *Engine) CurrentState() map[knowledge.NodeID]float64 {
	if e == nil {
		return map[knowledge.NodeID]float64{}
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return cloneState(e.internalState)
}

// PredictOutcomeFromNodes performs a hypothetical neural transition from the
// current recurrent state with the supplied action-target nodes injected.
// It never writes activation state, plasticity, memory, or event-log state.
// Learned pattern completion is included because it is part of Horizon's
// existing predictive substrate.
func (e *Engine) PredictOutcomeFromNodes(targets []knowledge.NodeID, now time.Time, cycles int) Prediction {
	if e == nil || e.Memory == nil {
		return Prediction{}
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if cycles < 1 {
		cycles = 1
	}

	e.mu.RLock()
	state := cloneState(e.internalState)
	confidence := cloneState(e.internalConfidence)
	e.mu.RUnlock()

	for _, id := range targets {
		if e.Memory.Registry.GetByID(id) == nil {
			continue
		}
		state[id] = max(state[id], 1)
		confidence[id] = max(confidence[id], 0.5)
	}
	state = normalize(state)
	predicted, predictedConfidence := e.advance(state, confidence, now, cycles)
	predicted, predictedConfidence = e.applyPatternPrediction(predicted, predictedConfidence, state)

	// Keep the representation deterministic for callers that compare snapshots.
	_ = sortedNodeIDs(predicted)
	return Prediction{State: cloneState(predicted), Confidence: cloneState(predictedConfidence)}
}
