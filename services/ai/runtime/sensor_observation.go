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
	if o == nil || o.Runtime == nil {
		return CognitiveInterpretation{}, errors.New("cognitive orchestrator is not initialized")
	}
	o.Runtime.sensorMu.Lock()
	defer o.Runtime.sensorMu.Unlock()
	if o.Runtime.HasProcessedEvent(sensor.SampleID) {
		return CognitiveInterpretation{}, fmt.Errorf("duplicate sensor sample ID %q", sensor.SampleID)
	}
	vector, err := sensor.NeuralVector()
	if err != nil {
		return CognitiveInterpretation{}, err
	}
	population, err := o.Runtime.brain.ProjectVectorPopulationAt(vector, GroundingThreshold, 4, sensor.Timestamp)
	if err != nil {
		return CognitiveInterpretation{}, err
	}
	if len(population.Units) == 0 {
		return CognitiveInterpretation{}, errors.New("sensor projection produced no neural population")
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
	output, err := o.Runtime.CognitiveProcess(event)
	if err != nil {
		return CognitiveInterpretation{}, err
	}
	if err := o.Runtime.LearnObservedTransition(ids, event.Timestamp); err != nil {
		return CognitiveInterpretation{}, err
	}
	interpretation, err := o.Runtime.InterpretCognitive(output, Observation{
		Source: sensor.Source, Modality: sensor.Modality,
	})
	if err != nil {
		return CognitiveInterpretation{}, err
	}
	interpretation.GroundedRepresentations = []knowledge.GroundedRepresentation{{
		NodeID: ids[0], Population: append([]knowledge.NodeID(nil), ids...),
		Similarity: population.Units[0].Activation, Status: knowledge.GroundingCandidate,
		Source: sensor.Source, Modality: sensor.Modality, Timestamp: event.Timestamp,
	}}
	return interpretation, nil
}
