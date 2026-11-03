package knowledge

import (
	"errors"
	"math"
)

// Output intervention only. These fields are absent from ordinary inference,
// never training labels, never a candidate policy, and never persistent state.
// All calculations use the same canonical N parameters and native operators;
// overrides are temporary local copies, leaving the Brain and Adam unchanged.
type Phase3InterventionV5 struct {
	OfflineOnly      bool        `json:"offline_only"`
	WorldHidden      []float64   `json:"world_hidden,omitempty"`
	ReadoutHidden    []float64   `json:"readout_hidden,omitempty"`
	Joint            []float64   `json:"joint,omitempty"`
	Competence       [][]float64 `json:"competence,omitempty"`
	Reward           [][]float64 `json:"reward,omitempty"`
	ActorPolicy      []float64   `json:"actor_policy,omitempty"`
	BMPSWeights      *[4]float64 `json:"bmps_weights,omitempty"`
	ParameterIndices []int       `json:"parameter_indices,omitempty"`
	ParameterValues  []float64   `json:"parameter_values,omitempty"`
}

func phase3VectorV5(x []float64, n int, probability bool) error {
	if len(x) != n {
		return errors.New("causal intervention shape mismatch")
	}
	sum := 0.
	for _, v := range x {
		if !finiteNumber(v) || probability && v < 0 {
			return errors.New("nonfinite or negative intervention")
		}
		sum += v
	}
	if probability && math.Abs(sum-1) > 1e-9 {
		return errors.New("causal probability must normalize")
	}
	return nil
}
func phase3MatrixV5(x [][]float64, rows, cols int) error {
	if len(x) != rows {
		return errors.New("causal readout row mismatch")
	}
	for _, r := range x {
		if err := phase3VectorV5(r, cols, false); err != nil {
			return err
		}
	}
	return nil
}
func (d Phase3InterventionV5) applyParameters(w []float64) error {
	if len(d.ParameterIndices) != len(d.ParameterValues) {
		return errors.New("causal parameter override mismatch")
	}
	seen := map[int]bool{}
	for i, p := range d.ParameterIndices {
		if p < 0 || p >= len(w) || seen[p] || !finiteNumber(d.ParameterValues[i]) {
			return errors.New("invalid causal parameter index/value")
		}
		seen[p] = true
		w[p] = d.ParameterValues[i]
	}
	return nil
}
