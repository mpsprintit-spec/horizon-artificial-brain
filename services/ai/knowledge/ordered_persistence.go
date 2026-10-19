package knowledge

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
)

// ValidateOrderedState checks versioned continuation before a restore mutates
// its destination. Historical patterns retain their original decoding rules.
func ValidateOrderedState(data []byte, trace *ObservationTrace, allowOrdered bool) error {
	var raw struct {
		Neurons []struct {
			ID NodeID `json:"id"`
		} `json:"neural_units"`
		Patterns       []*PatternSynapse `json:"temporal_patterns"`
		LegacyPatterns []*PatternSynapse `json:"patterns"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	known := make(map[NodeID]bool, len(raw.Neurons))
	for _, neuron := range raw.Neurons {
		known[neuron.ID] = true
	}
	if trace != nil && !allowOrdered {
		return errors.New("legacy checkpoint cannot carry ordered observation state")
	}
	if err := trace.Validate(func(id NodeID) bool { return known[id] }); err != nil {
		return err
	}
	for _, pattern := range append(raw.Patterns, raw.LegacyPatterns...) {
		if pattern == nil {
			continue
		}
		if err := pattern.Dynamics.Validate(); err != nil {
			return err
		}
		if pattern.Dynamics == LegacyDynamics {
			if pattern.Observation != nil {
				return errors.New("ordered pattern context requires versioned dynamics")
			}
			continue
		}
		if pattern.Dynamics != OrderedDynamics {
			return errors.New("unsupported pattern dynamics")
		}
		if !allowOrdered {
			return errors.New("legacy checkpoint cannot carry ordered patterns")
		}
		if len(pattern.Sequence) == 0 || pattern.Sequence[len(pattern.Sequence)-1].NodeID != pattern.Result || len(pattern.Context) > 0 {
			return errors.New("invalid ordered pattern result or context")
		}
		if pattern.Observation == nil || len(pattern.Observation.Stimulus) == 0 {
			return errors.New("ordered pattern is missing population frames")
		}
		if err := pattern.Observation.Validate(func(id NodeID) bool { return known[id] }); err != nil {
			return err
		}
		if !slices.Contains(pattern.Observation.Stimulus[len(pattern.Observation.Stimulus)-1], pattern.Result) {
			return errors.New("ordered pattern result is outside its outcome population")
		}
		expected := TraceSteps(pattern.Observation.Canonical().Stimulus)
		expected[len(expected)-1].NodeID = pattern.Result
		if sequenceKey(expected) != sequenceKey(pattern.Sequence) {
			return errors.New("ordered pattern sequence and population frames disagree")
		}
		for i, step := range pattern.Sequence {
			if step.Position != i || !known[step.NodeID] {
				return fmt.Errorf("ordered pattern references invalid position or neuron %d", step.NodeID)
			}
		}
	}
	return nil
}
