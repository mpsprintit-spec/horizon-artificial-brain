package activation

import (
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
	predicted, predictedConfidence := e.advanceHypothetical(state, confidence, now, cycles)
	predicted, predictedConfidence = e.applyPatternPrediction(predicted, predictedConfidence, state)

	// Keep the representation deterministic for callers that compare snapshots.
	_ = sortedNodeIDs(predicted)
	return Prediction{State: cloneState(predicted), Confidence: cloneState(predictedConfidence)}
}

func (e *Engine) advanceHypothetical(state, confidence map[knowledge.NodeID]float64, now time.Time, cycles int) (map[knowledge.NodeID]float64, map[knowledge.NodeID]float64) {
	if e == nil || e.Memory == nil {
		return cloneState(state), cloneState(confidence)
	}
	state = cloneState(state)
	confidence = cloneState(confidence)
	for i := 0; i < cycles; i++ {
		next := map[knowledge.NodeID]float64{}
		nextConfidence := map[knowledge.NodeID]float64{}
		for _, id := range sortedNodeIDs(state) {
			n := e.Memory.Registry.GetByID(id)
			if n == nil {
				continue
			}
			next[id] = n.RestingActivation
		}
		for _, id := range sortedNodeIDs(state) {
			level := state[id]
			n := e.Memory.Registry.GetByID(id)
			if n == nil {
				continue
			}
			next[id] += level * (1 - e.Decay)
			nextConfidence[id] = max(nextConfidence[id], confidence[id])
			synapses := n.OutboundAll()
			sort.Slice(synapses, func(i, j int) bool {
				return synapses[i].TargetID < synapses[j].TargetID
			})
			for _, s := range synapses {
				age := temporalPenalty(now, s.LastActivation)
				pulse := level * s.Weight * s.Confidence * e.SpreadRate * age
				if s.Inhibitory {
					next[s.TargetID] -= pulse * e.Inhibition
				} else {
					next[s.TargetID] += pulse
				}
				nextConfidence[s.TargetID] = max(nextConfidence[s.TargetID], confidence[id]*s.Confidence*age)
			}
		}
		state = normalize(next)
		confidence = normalize(nextConfidence)
	}
	return state, confidence
}
