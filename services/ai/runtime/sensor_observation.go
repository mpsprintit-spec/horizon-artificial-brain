package runtime

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// SensorObservation is the modality-neutral contract at the physical sensor
// boundary. Numeric values are normalized to [-1, 1] by the sensor adapter;
// labels and provenance remain metadata and never become neural identity.
type SensorObservation struct {
	Modality           string    `json:"modality"`
	Source             string    `json:"source"`
	Timestamp          time.Time `json:"timestamp"`
	SampleID           string    `json:"sample_id"`
	ChannelStates      []float64 `json:"channel_states"`
	SpatialCoordinates []float64 `json:"spatial_coordinates,omitempty"`
	TemporalDelta      []float64 `json:"temporal_delta,omitempty"`
	Confidence         float64   `json:"confidence"`
	Provenance         SensorProvenance `json:"provenance"`
}

// SensorProvenance identifies how a measurement was obtained without
// assigning semantic meaning to its neural representation.
type SensorProvenance struct {
	Adapter          string   `json:"adapter,omitempty"`
	CalibrationID    string   `json:"calibration_id,omitempty"`
	ParentSampleIDs  []string `json:"parent_sample_ids,omitempty"`
}

var ErrInvalidSensorObservation = errors.New("invalid sensor observation")

// Validate rejects ambiguous or non-finite physical input before it reaches
// the canonical brain. Numeric feature arrays are expected to be normalized.
// Clone returns an ownership-safe copy of the observation and all nested slices.
func (s SensorObservation) Clone() SensorObservation {
	out := s
	out.ChannelStates = append([]float64(nil), s.ChannelStates...)
	out.SpatialCoordinates = append([]float64(nil), s.SpatialCoordinates...)
	out.TemporalDelta = append([]float64(nil), s.TemporalDelta...)
	out.Provenance.ParentSampleIDs = append([]string(nil), s.Provenance.ParentSampleIDs...)
	return out
}

func (s SensorObservation) Validate() error {
	if strings.TrimSpace(s.Modality) == "" || strings.TrimSpace(s.Source) == "" || strings.TrimSpace(s.SampleID) == "" || s.Timestamp.IsZero() {
		return fmt.Errorf("%w: modality, source, sample_id and timestamp are required", ErrInvalidSensorObservation)
	}
	if len(s.ChannelStates)+len(s.SpatialCoordinates)+len(s.TemporalDelta) == 0 {
		return fmt.Errorf("%w: no numeric sensor state", ErrInvalidSensorObservation)
	}
	if len(s.ChannelStates) > 64 || len(s.SpatialCoordinates) > 64 || len(s.TemporalDelta) > 64 {
		return fmt.Errorf("%w: each numeric feature group is limited to 64 values", ErrInvalidSensorObservation)
	}
	if math.IsNaN(s.Confidence) || math.IsInf(s.Confidence, 0) || s.Confidence < 0 || s.Confidence > 1 {
		return fmt.Errorf("%w: confidence must be finite and within [0,1]", ErrInvalidSensorObservation)
	}
	for name, values := range map[string][]float64{
		"channel_states": s.ChannelStates,
		"spatial_coordinates": s.SpatialCoordinates,
		"temporal_delta": s.TemporalDelta,
	} {
		for i, value := range values {
			if math.IsNaN(value) || math.IsInf(value, 0) || value < -1 || value > 1 {
				return fmt.Errorf("%w: %s[%d] must be finite and within [-1,1]", ErrInvalidSensorObservation, name, i)
			}
		}
	}
	return nil
}

// NeuralVector produces a deterministic numeric pattern without converting
// sensor input to a token, hash, or semantic label. Section markers preserve
// the distinction between channel, spatial, and temporal coordinates.
func (s SensorObservation) NeuralVector() (knowledge.NeuralVector, error) {
	if err := s.Validate(); err != nil {
		return knowledge.NeuralVector{}, err
	}
	values := []float64{1, float64(len(s.ChannelStates)) / 64}
	values = append(values, s.ChannelStates...)
	values = append(values, -1, float64(len(s.SpatialCoordinates))/64)
	values = append(values, s.SpatialCoordinates...)
	values = append(values, 0.5, float64(len(s.TemporalDelta))/64)
	values = append(values, s.TemporalDelta...)
	values = append(values, s.Confidence)
	return knowledge.NewNeuralVector(values), nil
}

// ProcessSensorObservation projects structured sensor state directly into the
// canonical neural substrate. It deliberately bypasses lexical grounding:
// numeric sensor channels must not be collapsed into a synthetic token.
func (o *CognitiveOrchestrator) ProcessSensorObservation(sensor SensorObservation) (CognitiveInterpretation, error) {
	if o == nil || o.Runtime == nil || o.Runtime.brain == nil {
		return CognitiveInterpretation{}, errors.New("cognitive orchestrator is not initialized")
	}
	sensor = sensor.Clone()
	if err := sensor.Validate(); err != nil {
		return CognitiveInterpretation{}, err
	}
	vector, err := sensor.NeuralVector()
	if err != nil {
		return CognitiveInterpretation{}, err
	}

	type processedSensor struct {
		output     CognitiveOutput
		ids        []knowledge.NodeID
		similarity float64
	}
	// The runtime lock protects the entire canonical sensor mutation:
	// timestamp/duplicate validation, projection, cognition and temporal
	// reinforcement. Independent Process/Think/Learn calls cannot interleave.
	// Durable crash recovery remains a separate concern: an event-log failure
	// after projection can still require recovery from a known checkpoint.
	result, err := func() (processedSensor, error) {
		r := o.Runtime
		r.mu.Lock()
		defer r.mu.Unlock()

		if !r.lastObservationAt.IsZero() && sensor.Timestamp.Before(r.lastObservationAt) {
			return processedSensor{}, fmt.Errorf("%w: timestamp %s precedes last observation %s", ErrInvalidSensorObservation, sensor.Timestamp.UTC().Format(time.RFC3339Nano), r.lastObservationAt.UTC().Format(time.RFC3339Nano))
		}
		if _, exists := r.processedEvents[sensor.SampleID]; exists {
			return processedSensor{}, fmt.Errorf("duplicate sensor sample ID %q", sensor.SampleID)
		}

		population, err := r.brain.ProjectVectorPopulationAt(vector, GroundingThreshold, 4, sensor.Timestamp)
		if err != nil {
			return processedSensor{}, err
		}
		if len(population.Units) == 0 {
			return processedSensor{}, errors.New("sensor projection produced no neural population")
		}
		ids := make([]knowledge.NodeID, 0, len(population.Units))
		for _, unit := range population.Units {
			ids = append(ids, unit.NodeID)
		}
		event := Event{
			ID: sensor.SampleID, Source: sensor.Source, Modality: sensor.Modality,
			Timestamp: sensor.Timestamp.UTC(), Cycles: 1,
			StimulusNodeIDs: ids, SensorObservation: &sensor,
		}
		output, err := r.cognitiveProcessLocked(event)
		if err != nil {
			return processedSensor{}, err
		}
		if err := r.learnObservedTransitionLocked(ids, event.Timestamp); err != nil {
			return processedSensor{}, err
		}
		return processedSensor{output: output, ids: ids, similarity: population.Units[0].Activation}, nil
	}()
	if err != nil {
		return CognitiveInterpretation{}, err
	}

	interpretation, err := o.Runtime.InterpretCognitive(result.output, Observation{
		Source: sensor.Source, Modality: sensor.Modality,
	})
	if err != nil {
		return CognitiveInterpretation{}, err
	}
	interpretation.GroundedRepresentations = []knowledge.GroundedRepresentation{{
		NodeID: result.ids[0], Population: append([]knowledge.NodeID(nil), result.ids...),
		Similarity: result.similarity, Status: knowledge.GroundingCandidate,
		Source: sensor.Source, Modality: sensor.Modality, Timestamp: sensor.Timestamp.UTC(),
	}}
	return interpretation, nil
}
