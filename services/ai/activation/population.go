package activation

import (
	"errors"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

var ErrEmptyPopulation = errors.New("population is empty")

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

	population, err := e.Memory.ProjectVectorPopulationAt(vector, e.Threshold, 4, now)
	if err != nil {
		return Result{}, err
	}
	return e.ActivatePopulation(population, cycles, now)
}

// ActivatePopulation seeds the existing internal state with a distributed
// neural population and then advances the normal recurrent dynamics. The
// population is an activation pattern, not a second memory store.
func (e *Engine) ActivatePopulation(population knowledge.ProjectionPopulation, cycles int, now time.Time) (Result, error) {
	if e == nil || e.Memory == nil {
		return Result{}, knowledge.ErrNilBrain
	}
	if len(population.Units) == 0 {
		return Result{}, ErrEmptyPopulation
	}
	if cycles < 1 {
		cycles = 1
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	state := make(map[knowledge.NodeID]float64, len(population.Units))
	confidence := make(map[knowledge.NodeID]float64, len(population.Units))
	for _, unit := range population.Units {
		if e.Memory.Registry.GetByID(unit.NodeID) == nil {
			continue
		}
		level := clamp01(unit.Activation)
		if level <= 0 {
			continue
		}
		state[unit.NodeID] = level
		confidence[unit.NodeID] = level
	}
	if len(state) == 0 {
		return Result{}, ErrEmptyPopulation
	}

	state, confidence = e.advance(state, confidence, now, cycles)
	result := e.converge(state, confidence, now)

	e.mu.Lock()
	e.internalState = cloneState(result.Activations)
	e.internalConfidence = cloneState(result.Confidence)
	e.mu.Unlock()
	return result, nil
}

// LearnPopulationTransition strengthens recurrent connections between two
// distributed states. The connection has no semantic label: its meaning is
// determined by repeated temporal experience in the shared substrate.
func (e *Engine) LearnPopulationTransition(previous, current knowledge.ProjectionPopulation, weight, confidence float64, now time.Time) error {
	if e == nil || e.Memory == nil {
		return knowledge.ErrNilBrain
	}
	if len(previous.Units) == 0 || len(current.Units) == 0 {
		return ErrEmptyPopulation
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	weight = clamp01(weight)
	confidence = clamp01(confidence)
	if weight == 0 {
		weight = 1
	}
	if confidence == 0 {
		confidence = 1
	}

	for _, sourceUnit := range previous.Units {
		source := e.Memory.Registry.GetByID(sourceUnit.NodeID)
		if source == nil {
			continue
		}
		for _, targetUnit := range current.Units {
			target := e.Memory.Registry.GetByID(targetUnit.NodeID)
			if target == nil || source.ID == target.ID {
				continue
			}
			e.Memory.Connect(source, target, weight, confidence, false)
		}
	}
	return nil
}
