package activation

import (
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// ThinkWithPredictionAt is the deterministic form of the recurrent thought
// transition. The explicit timestamp makes replay and checkpoint continuation
// independent of wall-clock time while using the same neural dynamics as
// ThinkWithPrediction.
func (e *Engine) ThinkWithPredictionAt(cycles int, now time.Time) ThoughtResult {
	if cycles < 1 {
		cycles = 1
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	e.mu.RLock()
	state := cloneState(e.internalState)
	confidence := cloneState(e.internalConfidence)
	previousPrediction := cloneState(e.lastPrediction)
	e.mu.RUnlock()

	if len(state) == 0 {
		return ThoughtResult{Result: Result{
			Activations: map[knowledge.NodeID]float64{},
			Confidence:  map[knowledge.NodeID]float64{},
		}}
	}

	actualState, actualConfidence := e.advance(state, confidence, now, cycles)
	actual := e.converge(actualState, actualConfidence, now)
	e.ApplyActivityPlasticity(state, actual.Activations, now)

	error := 0.0
	if len(previousPrediction) > 0 {
		error = stateDifference(previousPrediction, actual.Activations)
		e.ApplyPredictionErrorPlasticity(previousPrediction, actual.Activations, error, now)
	}

	nextPrediction, nextPredictionConfidence := e.advance(actual.Activations, actual.Confidence, now, cycles)

	e.mu.Lock()
	e.internalState = cloneState(actual.Activations)
	e.internalConfidence = cloneState(actual.Confidence)
	e.lastPrediction = cloneState(nextPrediction)
	e.lastPredictionConf = cloneState(nextPredictionConfidence)
	e.mu.Unlock()

	return ThoughtResult{
		Result:          actual,
		Prediction:      Prediction{State: nextPrediction, Confidence: nextPredictionConfidence},
		PredictionError: error,
	}
}
