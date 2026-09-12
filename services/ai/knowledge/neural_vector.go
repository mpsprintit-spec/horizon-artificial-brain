package knowledge

// NeuralVector is the numeric representation presented by an experience
// source before it enters the shared brain dynamics. It deliberately carries
// no text token, modality name, semantic relation, or answer.
type NeuralVector struct {
	Values []float64
}

// NewNeuralVector copies and bounds a numeric activation pattern. Values are
// intentionally not assigned semantic meaning by the substrate.
func NewNeuralVector(values []float64) NeuralVector {
	out := make([]float64, len(values))
	for i, value := range values {
		out[i] = clamp(value, -1, 1)
	}
	return NeuralVector{Values: out}
}

// Similarity measures geometric agreement between two numeric experience
// patterns. Different modalities can use the same mechanism as long as their
// encoders produce compatible internal coordinates.
func (v NeuralVector) Similarity(other NeuralVector) float64 {
	if len(v.Values) == 0 || len(v.Values) != len(other.Values) {
		return 0
	}
	var distance float64
	for i, value := range v.Values {
		distance += abs(value - other.Values[i])
	}
	return clamp01(1 - distance/(2*float64(len(v.Values))))
}

func (v NeuralVector) Empty() bool { return len(v.Values) == 0 }
