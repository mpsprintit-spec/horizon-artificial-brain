package knowledge

import (
	"math"
	"reflect"
	"testing"
)

func queryTestWeights() compositionWeights {
	w := compositionWeights{E: make([][]float64, 3)}
	for i := range w.E {
		w.E[i] = make([]float64, compositionHidden)
		for h := range w.E[i] {
			w.E[i][h] = .21 * math.Sin(float64(i*17+h+1))
		}
	}
	for o := 0; o < compositionOutputs; o++ {
		w.B[o] = .03 * float64(o)
		for h := 0; h < compositionHidden; h++ {
			w.H[o][h] = .17 * math.Cos(float64(o*19+h+1))
		}
		for i := range w.E {
			w.P[o][i] = .5 + .1*float64(i)
		}
	}
	return w
}
func cloneQueryWeights(w compositionWeights) compositionWeights {
	r := w
	r.E = make([][]float64, len(w.E))
	for i, v := range w.E {
		r.E[i] = append([]float64(nil), v...)
	}
	return r
}
func queryLinearLoss(w compositionWeights, dz [compositionOutputs]float64) float64 {
	input := CompositionInput{Scene: CompositionScene{Point: []float64{0, 0, 1}, Patches: []CompositionPatch{{Values: []float64{.4, -.2, .6, -1, 0}, XY: []float64{0, 0}}}}}
	z, _ := compositionForwardVersion(input, w, NeuralCompositionQueryV2)
	r := 0.
	for o, v := range dz {
		r += v * z[o]
	}
	return r
}
func TestQueryCompositionGradients(t *testing.T) {
	w := queryTestWeights()
	dz := [compositionOutputs]float64{.12, -.2, .3, .09, -.7, .4}
	g := compositionQueryGradient(w, dz)
	eps := 1e-6
	compare := func(name string, analytic float64, mutate func(*compositionWeights, float64)) {
		hi, lo := cloneQueryWeights(w), cloneQueryWeights(w)
		mutate(&hi, eps)
		mutate(&lo, -eps)
		numeric := (queryLinearLoss(hi, dz) - queryLinearLoss(lo, dz)) / (2 * eps)
		if math.Abs(numeric-analytic) > 2e-8 {
			t.Fatalf("%s derivative analytic %.12g numeric %.12g", name, analytic, numeric)
		}
	}
	for i := range w.E {
		for h := 0; h < compositionHidden; h++ {
			compare("embedding", g.E[i][h], func(v *compositionWeights, d float64) { v.E[i][h] += d })
		}
	}
	for o := 0; o < compositionOutputs; o++ {
		compare("bias", g.B[o], func(v *compositionWeights, d float64) { v.B[o] += d })
		for h := 0; h < compositionHidden; h++ {
			compare("hidden", g.H[o][h], func(v *compositionWeights, d float64) { v.H[o][h] += d })
		}
		for i := range w.E {
			compare("position", g.P[o][i], func(v *compositionWeights, d float64) { v.P[o][i] += d })
		}
	}
}
func TestQueryCompositionCanonicalAliasGradient(t *testing.T) {
	w := queryTestWeights()
	w.E[2] = append([]float64(nil), w.E[0]...)
	before := cloneQueryWeights(w)
	dz := [compositionOutputs]float64{0, 0, 0, 0, .7, .1}
	g := compositionQueryGradient(w, dz)
	ids := [][]NodeID{{11}, {12}, {11}}
	next := compositionQueryStep(w, ids, dz, .25)
	for h, v := range w.E[0] {
		expected := compositionStep(v, g.E[0][h]+g.E[2][h], .25)
		if next.E[0][h] != expected || next.E[2][h] != expected {
			t.Fatal("shared canonical identity was not updated once with both role gradients")
		}
	}
	if !reflect.DeepEqual(w, before) {
		t.Fatal("detached old weights were mutated")
	}
	for h := 0; h < compositionHidden; h++ {
		hi, lo := cloneQueryWeights(w), cloneQueryWeights(w)
		eps := 1e-6
		hi.E[0][h] += eps
		hi.E[2][h] += eps
		lo.E[0][h] -= eps
		lo.E[2][h] -= eps
		numeric := (queryLinearLoss(hi, dz) - queryLinearLoss(lo, dz)) / (2 * eps)
		if math.Abs(numeric-g.E[0][h]-g.E[2][h]) > 2e-8 {
			t.Fatal("alias derivative wrong")
		}
	}
}
func TestQueryCompositionCanChangeFeatureOrder(t *testing.T) {
	w := compositionWeights{E: [][]float64{make([]float64, compositionHidden), make([]float64, compositionHidden)}}
	w.H[4][0] = 1
	w.H[4][1] = 1
	w.P[4][0] = 1
	w.E[0][0] = 1
	w.E[0][1] = -1
	w.E[1][0] = 2
	w.E[1][1] = -2
	input := CompositionInput{Scene: CompositionScene{Point: []float64{0, 0, 1}, Patches: []CompositionPatch{{Values: []float64{.4, -.2, .6, -1, 0}, XY: []float64{0, 0}}}}}
	a, _ := compositionForwardVersion(input, w, NeuralCompositionQueryV2)
	w.E[1][0] = -2
	w.E[1][1] = 2
	b, _ := compositionForwardVersion(input, w, NeuralCompositionQueryV2)
	if !(a[4] > 0 && b[4] < 0) {
		t.Fatal("question did not change feature interpretation")
	}
	old, _ := compositionForward(input, w)
	legacy, _ := compositionForwardVersion(input, w, NeuralCompositionV1)
	if old != legacy {
		t.Fatal("legacy forward changed")
	}
	if DefaultCompositionConfig().Version != NeuralCompositionV1 || SupportedCompositionVersion("future-unknown") {
		t.Fatal("opt-in/version validation changed")
	}
}
