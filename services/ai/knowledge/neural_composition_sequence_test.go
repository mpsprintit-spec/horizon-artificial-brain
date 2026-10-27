package knowledge

import (
	"math"
	"testing"
)

func sequenceTestFixture() (CompositionInput, sequenceWeights) {
	in := CompositionInput{Sequence: [][]float64{{.1, .2, .05, -.3, .04, .2, 1}, {-.2, .07, .11, .4, -.02, -.1, 1}, {.3, .2, .1, -.2, .1, .4, 1}}}
	var w sequenceWeights
	for h := 0; h < compositionHidden; h++ {
		w.Q[h] = .13 * math.Sin(float64(h+1))
		w.K[h] = .2 * math.Cos(float64(h+1))
		w.HB[h] = .03 * float64(h-5)
		for j := range w.W[h] {
			w.W[h][j] = .11 * math.Cos(float64(h*11+j))
		}
		for j := range w.R[h] {
			w.R[h][j] = .07 * math.Sin(float64(h*13+j+1))
		}
	}
	for o := 0; o < compositionOutputs; o++ {
		w.B[o] = .02 * float64(o)
		for h := 0; h < compositionHidden; h++ {
			w.H[o][h] = .17 * math.Sin(float64(o*13+h+1))
		}
	}
	return in, w
}
func TestSequenceCompositionFiniteDifference(t *testing.T) {
	in, w := sequenceTestFixture()
	dz := [compositionOutputs]float64{.1, -.2, .3, -.4, .5, -.6}
	g := sequenceGradient(in, w, dz)
	loss := func(a sequenceWeights) float64 {
		v := sequenceForward(in, a)
		sum := 0.
		for o, d := range dz {
			sum += d * v.Z[o]
		}
		return sum
	}
	check := func(name string, analytic float64, mutate func(*sequenceWeights, float64)) {
		hi, lo := w, w
		mutate(&hi, 1e-6)
		mutate(&lo, -1e-6)
		numeric := (loss(hi) - loss(lo)) / 2e-6
		if math.Abs(numeric-analytic) > 2e-8 {
			t.Fatalf("%s analytic %.12g numeric %.12g", name, analytic, numeric)
		}
	}
	for h := 0; h < compositionHidden; h++ {
		check("query", g.Q[h], func(v *sequenceWeights, d float64) { v.Q[h] += d })
		check("position key", g.K[h], func(v *sequenceWeights, d float64) { v.K[h] += d })
		check("hidden bias", g.HB[h], func(v *sequenceWeights, d float64) { v.HB[h] += d })
		for j := range w.W[h] {
			check("measurement/time/presence", g.W[h][j], func(v *sequenceWeights, d float64) { v.W[h][j] += d })
		}
		for j := range w.R[h] {
			check("recurrence", g.R[h][j], func(v *sequenceWeights, d float64) { v.R[h][j] += d })
		}
	}
	for o := 0; o < compositionOutputs; o++ {
		check("bias", g.B[o], func(v *sequenceWeights, d float64) { v.B[o] += d })
		for h := 0; h < compositionHidden; h++ {
			check("readout", g.H[o][h], func(v *sequenceWeights, d float64) { v.H[o][h] += d })
		}
	}
}
func TestSequenceCompositionOrderAndVersion(t *testing.T) {
	in, w := sequenceTestFixture()
	a := sequenceForward(in, w)
	in.Sequence[0], in.Sequence[2] = in.Sequence[2], in.Sequence[0]
	b := sequenceForward(in, w)
	if a.Z == b.Z {
		t.Fatal("ordered histories merged")
	}
	if validateCompositionMode(DefaultCompositionConfig(), in) == nil {
		t.Fatal("legacy contract silently accepted sequence")
	}
	if DefaultCompositionConfig().Version != NeuralCompositionV1 {
		t.Fatal("legacy default changed")
	}
	if validateCompositionSequence([][]float64{{1, 2, 3}}) == nil {
		t.Fatal("invalid sequence accepted")
	}
}

func TestSequenceFocusTeachingGradient(t *testing.T) {
	in, w := sequenceTestFixture()
	focus := []float64{0, 0, 1}
	var dz [compositionOutputs]float64
	g := sequenceGradient(in, w, dz, sequenceSupervision{focus, .7})
	loss := func(v sequenceWeights) float64 {
		out := sequenceForward(in, v)
		return -.7 * math.Log(out.Attention[2])
	}
	for h := 0; h < compositionHidden; h++ {
		hi, lo := w, w
		hi.K[h] += 1e-6
		lo.K[h] -= 1e-6
		numeric := (loss(hi) - loss(lo)) / 2e-6
		if math.Abs(numeric-g.K[h]) > 2e-8 {
			t.Fatal("focus gradient mismatch")
		}
	}
	if validateSequenceFocus(CompositionExposure{Input: in, SequenceFocus: focus, SequenceFocusWeight: 1}) != nil {
		t.Fatal("valid focus rejected")
	}
	if validateSequenceFocus(CompositionExposure{Input: in, SequenceFocus: []float64{1, 1, 1}, SequenceFocusWeight: 1}) == nil {
		t.Fatal("invalid focus accepted")
	}
}
