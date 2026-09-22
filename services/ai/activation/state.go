package activation

import "github.com/project-horizon/horizon-core/services/ai/knowledge"

// StateSnapshot captures the engine's internal recurrent state. It is runtime
// state, not a second memory substrate, and is persisted only so a checkpoint
// can resume the same ongoing neural process.
type StateSnapshot struct {
	InternalState      map[knowledge.NodeID]float64 `json:"internal_state,omitempty"`
	InternalConfidence map[knowledge.NodeID]float64 `json:"internal_confidence,omitempty"`
	LastPrediction     map[knowledge.NodeID]float64 `json:"last_prediction,omitempty"`
	LastPredictionConf map[knowledge.NodeID]float64 `json:"last_prediction_confidence,omitempty"`
}

func (e *Engine) SnapshotState() StateSnapshot {
	if e == nil {
		return StateSnapshot{}
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return StateSnapshot{
		InternalState:      cloneState(e.internalState),
		InternalConfidence: cloneState(e.internalConfidence),
		LastPrediction:     cloneState(e.lastPrediction),
		LastPredictionConf: cloneState(e.lastPredictionConf),
	}
}

func (e *Engine) RestoreState(snapshot StateSnapshot) {
	if e == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.internalState = cloneState(snapshot.InternalState)
	e.internalConfidence = cloneState(snapshot.InternalConfidence)
	e.lastPrediction = cloneState(snapshot.LastPrediction)
	e.lastPredictionConf = cloneState(snapshot.LastPredictionConf)
}


// PredictionSnapshot returns the current expected neural state at the exact
// boundary before an external event. The returned maps are defensive copies.
func (e *Engine) PredictionSnapshot() Prediction {
	if e == nil {
		return Prediction{State: map[knowledge.NodeID]float64{}, Confidence: map[knowledge.NodeID]float64{}}
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return Prediction{State: cloneState(e.lastPrediction), Confidence: cloneState(e.lastPredictionConf)}
}

// PredictionError compares a captured prediction with the resulting neural
// state without mutating the substrate.
func (e *Engine) PredictionError(prediction Prediction, actual map[knowledge.NodeID]float64) float64 {
	if e == nil {
		return 0
	}
	return stateDifference(prediction.State, actual)
}
