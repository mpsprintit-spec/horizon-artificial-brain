package activation

import (
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// ActivateVector presents a numeric experience to the same recurrent dynamics
// used by the rest of the brain. Projection only seeds a sparse population;
// subsequent state evolution is performed by the shared activation substrate.
func (e *Engine) ActivateVector(vector knowledge.NeuralVector, cycles int, now time.Time) (Result, error) {
	if e == nil || e.Memory == nil {
		return Result{}, knowledge.ErrNilBrain
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if cycles < 1 {
		cycles = 1
	}

	population, err := e.Memory.ProjectVectorPopulation(vector, e.Threshold, 4)
	if err != nil {
		return Result{}, err
	}

	state := make(map[knowledge.NodeID]float64, len(population.Units))
	confidence := make(map[knowledge.NodeID]float64, len(population.Units))
	for _, unit := range population.Units {
		state[unit.NodeID] = unit.Activation
		confidence[unit.NodeID] = unit.Activation
	}
	state = normalize(state)
	confidence = normalize(confidence)

	state, confidence = e.advance(state, confidence, now, cycles)
	result := e.converge(state, confidence, now)

	e.mu.Lock()
	e.internalState = cloneState(result.Activations)
	e.internalConfidence = cloneState(result.Confidence)
	e.lastPrediction = map[knowledge.NodeID]float64{}
	e.lastPredictionConf = map[knowledge.NodeID]float64{}
	e.mu.Unlock()

	return result, nil
}
