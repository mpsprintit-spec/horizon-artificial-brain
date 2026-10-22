package activation

import (
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// ThinkWithPredictionAt is the deterministic form of the internal recurrent
// transition. The supplied timestamp is used for temporal decay/plasticity so
// event-log replay can reproduce the same transition.
func (e *Engine) ThinkWithPredictionAt(cycles int, now time.Time) ThoughtResult {
	return e.ThinkWithContextAt(cycles, nil, now)
}

// ThinkWithContextAt performs one recurrent transition while adding an
// endogenous context signal to the existing neural state. The context is
// additive and therefore cannot replace the current recurrent trajectory.
func (e *Engine) ThinkWithContextAt(cycles int, context map[knowledge.NodeID]float64, now time.Time) ThoughtResult {
	return e.ThinkWithContextDynamicsAt(cycles, context, now, knowledge.LegacyDynamics)
}

func (e *Engine) ThinkWithContextDynamicsAt(cycles int, context map[knowledge.NodeID]float64, now time.Time, dynamics knowledge.DynamicsEncoding) ThoughtResult {
	if cycles < 1 {
		cycles = 1
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	e.mu.RLock()
	state := cloneState(e.internalState)
	confidence := cloneState(e.internalConfidence)
	previousPrediction := cloneState(e.lastPrediction)
	trace := e.lastTrace.Clone()
	recurrentContext := e.contextState.Clone()
	e.mu.RUnlock()

	for id, boost := range context {
		if boost <= 0 { continue }
		state[id] += boost
		confidence[id] = max(confidence[id], boost)
	}
	state = normalize(state)
	confidence = normalize(confidence)

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
	if dynamics == knowledge.RecurrentDynamics {
  nextPrediction,nextPredictionConfidence=e.applyRecurrentPatternPrediction(nextPrediction,nextPredictionConfidence,trace,recurrentContext,now,true)
	} else if dynamics == knowledge.OrderedDynamics {nextPrediction,nextPredictionConfidence=e.applyOrderedPatternPrediction(nextPrediction,nextPredictionConfidence,trace)}

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
