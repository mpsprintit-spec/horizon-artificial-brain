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
