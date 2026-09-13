package activation

import (
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// CompleteFromCue reconstructs learned temporal activity from a partial
// internal cue and feeds the reconstructed units back into the same recurrent
// activation state. It is deliberately a substrate operation: no semantic
// answer or rule is selected by this method.
func (e *Engine) CompleteFromCue(cue []knowledge.PatternStep, context []knowledge.ContextFrame, cycles int, now time.Time) ([]knowledge.PatternSynapse, Result, error) {
	if e == nil || e.Memory == nil {
		return nil, Result{}, knowledge.ErrNilBrain
	}
	matches := e.Memory.Patterns.CompleteTrace(cue, context)
	if len(matches) == 0 {
		return nil, Result{Activations: map[knowledge.NodeID]float64{}, Confidence: map[knowledge.NodeID]float64{}}, nil
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if cycles < 1 {
		cycles = 1
	}

	state := map[knowledge.NodeID]float64{}
	confidence := map[knowledge.NodeID]float64{}
	for _, match := range matches {
		strength := clamp01(match.Weight * match.Confidence)
		if strength == 0 {
			strength = 0.05
		}
		for _, step := range match.Sequence {
			level := clamp01(strength * step.Activation)
			if level > state[step.NodeID] {
				state[step.NodeID] = level
			}
			confidence[step.NodeID] = max(confidence[step.NodeID], strength)
		}
	}

	state, confidence = e.advance(state, confidence, now, cycles)
	result := e.converge(state, confidence, now)

	e.mu.Lock()
	e.internalState = cloneState(result.Activations)
	e.internalConfidence = cloneState(result.Confidence)
	e.lastPrediction = map[knowledge.NodeID]float64{}
	e.lastPredictionConf = map[knowledge.NodeID]float64{}
	e.mu.Unlock()
	return matches, result, nil
}
