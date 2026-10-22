package knowledge

import "time"

// SensoryFieldBinding binds reused neural factors to a local measurement.
// A group names a sensor patch or time window, not an inferred object.
type SensoryFieldBinding struct {
	FieldID          string    `json:"field_id"`
	GroupID          string    `json:"group_id"`
	Kind             string    `json:"kind"`
	Values           []float64 `json:"values"`
	Coordinates      []float64 `json:"coordinates,omitempty"`
	TemporalOffsetNS int64     `json:"temporal_offset_ns"`
	TemporalScaleNS  int64     `json:"temporal_scale_ns"`
	Population       []NodeID  `json:"population"`
}

// SensoryExperienceTrace is bounded causal memory within the same Brain.
// It stores measurements and neural references, never JPEG/WAV/media files.
type SensoryExperienceTrace struct {
	SampleID        string                `json:"sample_id"`
	Source          string                `json:"source"`
	Modality        string                `json:"modality"`
	Timestamp       time.Time             `json:"timestamp"`
	Confidence      float64               `json:"confidence"`
	Adapter         string                `json:"adapter,omitempty"`
	CalibrationID   string                `json:"calibration_id,omitempty"`
	ParentSampleIDs []string              `json:"parent_sample_ids,omitempty"`
	Fields          []SensoryFieldBinding `json:"fields"`
	PredictionError float64               `json:"prediction_error"`
}

func CloneSensoryTrace(trace SensoryExperienceTrace) SensoryExperienceTrace {
	out := trace
	out.ParentSampleIDs = append([]string(nil), trace.ParentSampleIDs...)
	out.Fields = make([]SensoryFieldBinding, len(trace.Fields))
	for i, field := range trace.Fields {
		out.Fields[i] = field
		out.Fields[i].Values = append([]float64(nil), field.Values...)
		out.Fields[i].Coordinates = append([]float64(nil), field.Coordinates...)
		out.Fields[i].Population = append([]NodeID(nil), field.Population...)
	}
	return out
}

func (k *KnowledgeBase) LastSensoryTrace(source, modality string) (SensoryExperienceTrace, bool) {
	if k == nil {
		return SensoryExperienceTrace{}, false
	}
	k.mu.RLock()
	defer k.mu.RUnlock()
	for i := len(k.BrainState.SensoryTraces) - 1; i >= 0; i-- {
		trace := k.BrainState.SensoryTraces[i]
		if trace.Source == source && trace.Modality == modality {
			return CloneSensoryTrace(trace), true
		}
	}
	return SensoryExperienceTrace{}, false
}

func (k *KnowledgeBase) RecordSensoryTrace(trace SensoryExperienceTrace) {
	if k == nil {
		return
	}
	frozen := CloneSensoryTrace(trace)
	k.mu.Lock()
	defer k.mu.Unlock()
	k.BrainState.SensoryTraces = append(k.BrainState.SensoryTraces, frozen)
	if len(k.BrainState.SensoryTraces) > maxExperienceTraces {
		start := len(k.BrainState.SensoryTraces) - maxExperienceTraces
		k.BrainState.SensoryTraces = append([]SensoryExperienceTrace(nil), k.BrainState.SensoryTraces[start:]...)
	}
}
