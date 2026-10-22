package runtime

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// SensorField is one local physical measurement. Kind describes its numeric
// coordinate system (for example RGB), not a semantic object/category label.
// Offsets are relative to the observation timestamp; browser audio windows
// can therefore retain their timing without claiming uninterrupted capture.
type SensorField struct {
	FieldID          string    `json:"field_id"`
	GroupID          string    `json:"group_id"`
	Kind             string    `json:"kind"`
	Values           []float64 `json:"values"`
	Coordinates      []float64 `json:"coordinates,omitempty"`
	TemporalOffsetNS int64     `json:"temporal_offset_ns,omitempty"`
	TemporalScaleNS  int64     `json:"temporal_scale_ns,omitempty"`
}

func (s SensorObservation) validateFields() error {
	if len(s.Fields) == 0 {
		return nil
	}
	if len(s.Fields) > 128 {
		return fmt.Errorf("%w: at most 128 local fields", ErrInvalidSensorObservation)
	}
	if len(s.ChannelStates)+len(s.SpatialCoordinates)+len(s.TemporalDelta) != 0 {
		return fmt.Errorf("%w: fields cannot be mixed with a legacy flat observation", ErrInvalidSensorObservation)
	}
	seen := make(map[string]struct{}, len(s.Fields))
	for _, field := range s.Fields {
		if strings.TrimSpace(field.FieldID) == "" || strings.TrimSpace(field.Kind) == "" || strings.TrimSpace(field.GroupID) == "" || len(field.FieldID) > 128 || len(field.Kind) > 128 || len(field.GroupID) > 128 {
			return fmt.Errorf("%w: bounded field_id, kind and group_id are required", ErrInvalidSensorObservation)
		}
		if _, exists := seen[field.FieldID]; exists {
			return fmt.Errorf("%w: duplicate field_id %q", ErrInvalidSensorObservation, field.FieldID)
		}
		seen[field.FieldID] = struct{}{}
		if len(field.Values) == 0 || len(field.Values) > 64 || len(field.Coordinates) > 4 {
			return fmt.Errorf("%w: field values require 1..64 dimensions and at most four coordinates", ErrInvalidSensorObservation)
		}
		if field.TemporalScaleNS < 0 || field.TemporalScaleNS > int64(time.Minute) || field.TemporalOffsetNS > 0 || field.TemporalOffsetNS < -int64(time.Minute) {
			return fmt.Errorf("%w: field timing must describe at most one minute of captured history", ErrInvalidSensorObservation)
		}
		if field.TemporalOffsetNS+field.TemporalScaleNS > 0 {
			return fmt.Errorf("%w: field capture window extends beyond observation time", ErrInvalidSensorObservation)
		}
		for _, values := range [][]float64{field.Values, field.Coordinates} {
			for _, value := range values {
				if math.IsNaN(value) || math.IsInf(value, 0) || value < -1 || value > 1 {
					return fmt.Errorf("%w: field values must be finite within [-1,1]", ErrInvalidSensorObservation)
				}
			}
		}
	}
	return nil
}

// ProcessSensorObservation is a typed, journaled entry point for adapters.
func (r *BrainRuntime) ProcessSensorObservation(sensor SensorObservation) (CognitiveInterpretation, error) {
	return NewCognitiveOrchestrator(r).ProcessSensorObservation(sensor)
}

// Adjacent duration measurements share a receptive temporal scale, rather
// than allocating new populations for nanosecond-level capture jitter.
// Round to the nearest power of two in nanoseconds. Exact capture timing
// remains on each binding and is used by temporal learning and replay.
func sensorFieldScope(modality, kind string, scale int64) knowledge.ProjectionScope {
	if scale > 0 {
		lower := int64(1)
		for lower <= scale/2 {
			lower *= 2
		}
		if scale-lower >= 2*lower-scale {
			lower *= 2
		}
		scale = lower
	}
	return knowledge.ProjectionScope{Modality: modality, Field: kind, TemporalScaleNS: scale}
}

func (r *BrainRuntime) projectSensorFieldsLocked(sensor SensorObservation) ([]knowledge.SensoryFieldBinding, []knowledge.NodeID, error) {
	bindings := make([]knowledge.SensoryFieldBinding, 0, len(sensor.Fields))
	ids := make([]knowledge.NodeID, 0, len(sensor.Fields)*4)
	seen := make(map[knowledge.NodeID]struct{})
	for _, field := range sensor.Fields {
		population, err := r.brain.ProjectSensoryFieldAt(knowledge.NewNeuralVector(field.Values), sensorFieldScope(sensor.Modality, field.Kind, field.TemporalScaleNS), GroundingThreshold, 4, sensor.Timestamp)
		if err != nil {
			return nil, nil, err
		}
		binding := knowledge.SensoryFieldBinding{
			FieldID: field.FieldID, GroupID: field.GroupID, Kind: field.Kind,
			Values: append([]float64(nil), field.Values...), Coordinates: append([]float64(nil), field.Coordinates...),
			TemporalScaleNS: field.TemporalScaleNS, TemporalOffsetNS: field.TemporalOffsetNS,
			Population: make([]knowledge.NodeID, 0, len(population.Units)),
		}
		for _, unit := range population.Units {
			binding.Population = append(binding.Population, unit.NodeID)
			if _, exists := seen[unit.NodeID]; !exists {
				seen[unit.NodeID] = struct{}{}
				ids = append(ids, unit.NodeID)
			}
		}
		bindings = append(bindings, binding)
	}
	return bindings, ids, nil
}

// Each field learns only its own predecessor in the same source/group and
// coordinate system. Flattening whole scenes here would create all-to-all
// temporal links and destroy the distinction between local bindings.
func (r *BrainRuntime) learnSensorFieldsLocked(sensor SensorObservation, bindings []knowledge.SensoryFieldBinding, ids []knowledge.NodeID, predictionError float64) error {
	previous, found := r.brain.LastSensoryTrace(sensor.Source, sensor.Modality)
	if found {
		byID := make(map[string]knowledge.SensoryFieldBinding, len(previous.Fields))
		for _, field := range previous.Fields {
			byID[field.FieldID] = field
		}
		for _, field := range bindings {
			old, exists := byID[field.FieldID]
			if !exists || old.Kind != field.Kind || old.GroupID != field.GroupID || sensorFieldScope(sensor.Modality, old.Kind, old.TemporalScaleNS) != sensorFieldScope(sensor.Modality, field.Kind, field.TemporalScaleNS) {
				continue
			}
			before := previous.Timestamp.Add(time.Duration(old.TemporalOffsetNS))
			after := sensor.Timestamp.Add(time.Duration(field.TemporalOffsetNS))
			if after.Before(before) {
				continue
			}
			if err := r.brain.ReinforceTransitionPopulation(old.Population, field.Population, before, after, 0.25); err != nil {
				return err
			}
		}
	}
	// Ordered windows within one acoustic capture also produce temporal
	// evidence. Simultaneous spatial patches have equal offsets and do not
	// acquire invented left-to-right temporal relationships.
	lastWindow := make(map[knowledge.ProjectionScope]knowledge.SensoryFieldBinding)
	windows := append([]knowledge.SensoryFieldBinding(nil), bindings...)
	sort.SliceStable(windows, func(i, j int) bool { return windows[i].TemporalOffsetNS < windows[j].TemporalOffsetNS })
	for _, field := range windows {
		scope := sensorFieldScope(sensor.Modality, field.Kind, field.TemporalScaleNS)
		if old, exists := lastWindow[scope]; exists && field.TemporalOffsetNS > old.TemporalOffsetNS {
			if err := r.brain.ReinforceTransitionPopulation(old.Population, field.Population, sensor.Timestamp.Add(time.Duration(old.TemporalOffsetNS)), sensor.Timestamp.Add(time.Duration(field.TemporalOffsetNS)), 0.25); err != nil {
				return err
			}
		}
		lastWindow[scope] = field
	}
	r.brain.RecordSensoryTrace(knowledge.SensoryExperienceTrace{
		SampleID: sensor.SampleID, Source: sensor.Source, Modality: sensor.Modality, Timestamp: sensor.Timestamp.UTC(),
		Confidence: sensor.Confidence, Adapter: sensor.Provenance.Adapter, CalibrationID: sensor.Provenance.CalibrationID,
		ParentSampleIDs: append([]string(nil), sensor.Provenance.ParentSampleIDs...), Fields: bindings, PredictionError: predictionError,
	})
	r.lastObservationPopulation = append([]knowledge.NodeID(nil), ids...)
	r.lastObservationAt = sensor.Timestamp.UTC()
	return nil
}
