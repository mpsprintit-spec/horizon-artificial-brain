package runtime

import (
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"math"
	"testing"
)

func TestMetricProductPreservesMagnitude(t *testing.T) {
	r, _, _, _ := mNew(t, knowledge.NeuralCompositionMetricV6)
	fit := nGrid([]float64{-.8, -.4, .4, .8}, []float64{-.7, -.3, .3, .7})
	check := nGrid([]float64{-.9, -.2, .2, .9}, []float64{-.6, -.1, .1, .6})
	for i := 0; i < 6; i++ {
		q := i % 2
		f := func(p nXY) float64 {
			if q == 0 {
				return p.x
			}
			return p.y
		}
		_ = mTeach(t, r, nLesson(q, i, fit, check, f), 1e-4)
	}
	ds := []knowledge.NDiagnosis{}
	product := func(p nXY) float64 { return p.x * p.y }
	for i := 6; i < 14; i++ {
		ds = append(ds, mTeach(t, r, nLesson(2, i, fit, check, product), 1e-4))
	}
	correct := 0
	maxerr := 0.
	for _, p := range nGrid([]float64{-.43, -.21, .17, .61}, []float64{-.49, .37}) {
		o := mRead(t, r, 2, p, 1e-4)
		err := math.Abs(o.NResolution.Value - product(p))
		maxerr = math.Max(maxerr, err)
		if o.Accepted && err <= 1e-4 {
			correct++
		}
	}
	if correct != 8 {
		t.Fatalf("nonlinear metric composition %d/8 maxerror %g", correct, maxerr)
	}
	found := false
	for _, f := range r.brain.BrainState.NResolution.Features {
		found = found || f.Kind == 3
	}
	if !found {
		t.Fatal("numeric product not represented in canonical graph")
	}
	mSave(t, "metric_product.json", map[string]any{"diagnoses": ds, "correct": correct, "total": 8, "max_error": maxerr, "learned_n": len(r.brain.BrainState.NResolution.Features)})
}
