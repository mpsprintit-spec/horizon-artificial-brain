package knowledge

import (
	"errors"
	"fmt"
	"slices"
)

type DynamicsEncoding string

const (
	LegacyDynamics              DynamicsEncoding = ""
	OrderedDynamics             DynamicsEncoding = "ordered-observation-v1"
	RecurrentDynamics           DynamicsEncoding = "recurrent-context-v1"
	PhysicalMemoryDynamics      DynamicsEncoding = "physical-memory-v1"
	LocalActivityDynamics       DynamicsEncoding = "local-activity-v1"
	ObservedFeedbackDynamics    DynamicsEncoding = "observed-feedback-v1"
	EvidenceUncertaintyDynamics DynamicsEncoding = "evidence-uncertainty-v1"
	CurrentDynamics                              = EvidenceUncertaintyDynamics
)

func (d DynamicsEncoding) Validate() error {
	if d != LegacyDynamics && d != OrderedDynamics && d != RecurrentDynamics && d != PhysicalMemoryDynamics && d != LocalActivityDynamics && d != ObservedFeedbackDynamics && d != EvidenceUncertaintyDynamics {
		return fmt.Errorf("unsupported neural dynamics %q", d)
	}
	return nil
}

// ObservationTrace preserves ordered simultaneous population frames. Members
// inside one frame form a set; repeated frames and stream boundaries matter.
// Streams are adapter structure, never grammatical or semantic role labels.
type ObservationTrace struct {
	Stimulus [][]NodeID `json:"stimulus,omitempty"`
	Context  [][]NodeID `json:"context,omitempty"`
	Data     [][]NodeID `json:"data,omitempty"`
}

func ClonePopulationFrames(frames [][]NodeID) [][]NodeID {
	if len(frames) == 0 {
		return nil
	}
	out := make([][]NodeID, len(frames))
	for i, frame := range frames {
		out[i] = slices.Clone(frame)
	}
	return out
}

func (t *ObservationTrace) Clone() *ObservationTrace {
	if t == nil {
		return nil
	}
	return &ObservationTrace{Stimulus: ClonePopulationFrames(t.Stimulus), Context: ClonePopulationFrames(t.Context), Data: ClonePopulationFrames(t.Data)}
}

func (t *ObservationTrace) Canonical() *ObservationTrace {
	out := t.Clone()
	if out == nil {
		return nil
	}
	for _, stream := range [][][]NodeID{out.Stimulus, out.Context, out.Data} {
		for i := range stream {
			slices.Sort(stream[i])
			stream[i] = slices.Compact(stream[i])
		}
	}
	return out
}

func (t *ObservationTrace) Validate(known func(NodeID) bool) error {
	if t == nil {
		return nil
	}
	for _, stream := range [][][]NodeID{t.Stimulus, t.Context, t.Data} {
		for _, frame := range stream {
			if len(frame) == 0 {
				return errors.New("observation trace contains an empty population frame")
			}
			for _, id := range frame {
				if id <= 0 || known != nil && !known(id) {
					return fmt.Errorf("observation trace references unknown neuron %d", id)
				}
			}
		}
	}
	return nil
}

// TraceSteps is the compatibility anchor projection; ordered patterns also
// retain full population frames so shared anchors cannot erase identity.
func TraceSteps(frames [][]NodeID) []PatternStep {
	steps := make([]PatternStep, 0, len(frames))
	for i, frame := range frames {
		if len(frame) > 0 {
			steps = append(steps, PatternStep{NodeID: frame[0], Position: i, Activation: 1})
		}
	}
	return steps
}

func (d DynamicsEncoding) HasOrderedObservations() bool {
	return d == OrderedDynamics || d.HasRecurrentContext()
}

func (d DynamicsEncoding) HasRecurrentContext() bool {
	return d == RecurrentDynamics || d.HasPhysicalMemory()
}

func (d DynamicsEncoding) HasPhysicalMemory() bool {
	return d == PhysicalMemoryDynamics || d.HasLocalActivity()
}

func (d DynamicsEncoding) HasLocalActivity() bool {
	return d == LocalActivityDynamics || d.HasObservedFeedback()
}

func (d DynamicsEncoding) HasObservedFeedback() bool {
	return d == ObservedFeedbackDynamics || d == EvidenceUncertaintyDynamics
}
