package knowledge

import (
	"math"
	"slices"
)

func SupportedCompositionVersion(version string) bool {
	return version == NeuralCompositionV1 || version == NeuralCompositionQueryV2 || version == NeuralCompositionSequenceV4 || version == NeuralCompositionResolutionV5
}

// The final observed frame conditions each earlier frame. Its learned
// canonical embedding supplies a bounded channel gate 1+tanh(q), not a
// hardcoded vocabulary, target or market rule. No trainable scalar lives here.
func compositionQueryActivations(e [][]float64) [][]float64 {
	if len(e) < 2 {
		return e
	}
	q := len(e) - 1
	a := make([][]float64, len(e))
	a[q] = e[q]
	for i := 0; i < q; i++ {
		a[i] = make([]float64, len(e[i]))
		for h, v := range e[i] {
			a[i][h] = v * (1 + math.Tanh(e[q][h]))
		}
	}
	return a
}

func compositionForwardVersion(input CompositionInput, w compositionWeights, version string) ([compositionOutputs]float64, []float64) {
	if version == NeuralCompositionQueryV2 {
		w.E = compositionQueryActivations(w.E)
	}
	return compositionForward(input, w)
}

// Gradients are with respect to the original embeddings, including both roles
// if one canonical identity occurs as a feature and as the conditioning frame.
func compositionQueryGradient(w compositionWeights, dz [compositionOutputs]float64) compositionWeights {
	n := len(w.E)
	a := compositionQueryActivations(w.E)
	g := compositionWeights{E: make([][]float64, n)}
	da := make([][]float64, n)
	for i := range w.E {
		g.E[i] = make([]float64, compositionHidden)
		da[i] = make([]float64, compositionHidden)
	}
	for out := 0; out < compositionOutputs; out++ {
		g.B[out] = dz[out]
		for i, v := range a {
			for h, x := range v {
				g.H[out][h] += dz[out] * w.P[out][i] * x
				g.P[out][i] += dz[out] * w.H[out][h] * x
				da[i][h] += dz[out] * w.P[out][i] * w.H[out][h]
			}
		}
	}
	if n < 2 {
		g.E = da
		return g
	}
	q := n - 1
	for h := 0; h < compositionHidden; h++ {
		t := math.Tanh(w.E[q][h])
		g.E[q][h] = da[q][h]
		for i := 0; i < q; i++ {
			g.E[i][h] = da[i][h] * (1 + t)
			g.E[q][h] += da[i][h] * w.E[i][h] * (1 - t*t)
		}
	}
	return g
}

func compositionQueryStep(old compositionWeights, ids [][]NodeID, dz [compositionOutputs]float64, reliability float64) compositionWeights {
	g := compositionQueryGradient(old, dz)
	next := old
	next.E = make([][]float64, len(old.E))
	for i, v := range old.E {
		next.E[i] = slices.Clone(v)
	}
	for out := 0; out < compositionOutputs; out++ {
		next.B[out] = compositionStep(old.B[out], g.B[out], reliability)
		for h := 0; h < compositionHidden; h++ {
			next.H[out][h] = compositionStep(old.H[out][h], g.H[out][h], reliability)
		}
		for i := range old.E {
			next.P[out][i] = compositionStep(old.P[out][i], g.P[out][i], reliability)
		}
	}
	for i, p := range ids {
		first := true
		for j := 0; j < i; j++ {
			if slices.Equal(p, ids[j]) {
				first = false
				break
			}
		}
		if !first {
			continue
		}
		for h, v := range old.E[i] {
			grad := 0.
			for j, q := range ids {
				if slices.Equal(p, q) {
					grad += g.E[j][h]
				}
			}
			value := compositionStep(v, grad, reliability)
			for j, q := range ids {
				if slices.Equal(p, q) {
					next.E[j][h] = value
				}
			}
		}
	}
	return next
}
