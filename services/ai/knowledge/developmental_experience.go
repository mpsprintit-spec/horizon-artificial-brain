package knowledge

import (
	"encoding/json"
	"time"
)

type DevelopmentalStateFrame struct {
	ActiveUnits []NodeID `json:"active_units,omitempty"`
	ActivePopulations []NodeID `json:"active_populations,omitempty"`
	Representation []float64 `json:"representation,omitempty"`
	Confidence float64 `json:"confidence"`
}

type DevelopmentalTransition struct {
	ChangeRepresentation []float64 `json:"change_representation,omitempty"`
	TemporalDeltasNanos []int64 `json:"temporal_deltas_nanos,omitempty"`
	Confidence float64 `json:"confidence"`
}

type SymbolExposure struct {
	Symbol string `json:"symbol,omitempty"`
	Modality string `json:"modality"`
	Representation []float64 `json:"representation,omitempty"`
	RelativePosition string `json:"relative_position,omitempty"`
	SequencePosition int `json:"sequence_position"`
	Confidence float64 `json:"confidence"`
	SemanticFrame json.RawMessage `json:"semantic_frame,omitempty"`
}

type VisualExposure struct {
	Representation []float64 `json:"representation,omitempty"`
	RelativePosition string `json:"relative_position,omitempty"`
	SequencePosition int `json:"sequence_position"`
	Confidence float64 `json:"confidence"`
}

type DevelopmentalSynapticDelta struct {
	SourceNodeID NodeID `json:"source_node_id"`
	TargetNodeID NodeID `json:"target_node_id"`
	Delta float64 `json:"delta"`
}

type DevelopmentalLearningTrace struct {
	ActiveUnits []NodeID `json:"active_units,omitempty"`
	ActivePopulations []NodeID `json:"active_populations,omitempty"`
	TemporalTrace []time.Time `json:"temporal_trace,omitempty"`
	Prediction []float64 `json:"prediction,omitempty"`
	Outcome []float64 `json:"outcome,omitempty"`
	PredictionError float64 `json:"prediction_error"`
	RepresentationDelta []float64 `json:"representation_delta,omitempty"`
	SynapticDeltas []DevelopmentalSynapticDelta `json:"synaptic_deltas,omitempty"`
	Plasticity float64 `json:"plasticity"`
	Confidence float64 `json:"confidence"`
}

type DevelopmentalExperience struct {
	ID string `json:"id"`
	BeforeState DevelopmentalStateFrame `json:"before_state"`
	Transition DevelopmentalTransition `json:"transition"`
	AfterState DevelopmentalStateFrame `json:"after_state"`
	SymbolExposures []SymbolExposure `json:"symbol_exposures,omitempty"`
	LearningTrace DevelopmentalLearningTrace `json:"learning_trace"`
	Provenance BootstrapProvenance `json:"provenance"`
}
