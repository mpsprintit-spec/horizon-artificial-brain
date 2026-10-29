package runtime

import (
	"bytes"
	"encoding/json"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMetricInvalidEvidenceFailsBeforeMutation(t *testing.T) {
	for _, version := range []string{knowledge.NeuralCompositionResolutionV5, knowledge.NeuralCompositionMetricV6} {
		r, _, _, _ := mNew(t, version)
		fit := nGrid([]float64{-.8, -.4, .4, .8}, []float64{-.7, .7})
		check := nGrid([]float64{-.9, -.2, .2, .9}, []float64{-.5, .5})
		base := nLesson(0, 0, fit, check, func(p nXY) float64 { return nSign(p.x) })
		base.Motor = []float64{0, 0, 0, 0}
		base.After = nScene(base.After.ObservedAt)
		for _, fault := range []string{"negative-goal", "nan-measurement", "future", "episode-overlap", "legacy-precision"} {
			e, err := freezeComposition(base)
			compositionCheck(t, err)
			at := e.After.ObservedAt
			switch fault {
			case "negative-goal":
				x := -1.
				e.Input.RequiredPrecision = &x
			case "nan-measurement":
				e.Input.MeasurementError = [][]float64{{math.NaN(), 0, 0, 0, 0, 0}}
			case "future":
				at = at.Add(-time.Hour)
			case "episode-overlap":
				e.Resolution.Check[0].EpisodeID = e.Resolution.Fit[0].EpisodeID
			case "legacy-precision":
				if version == knowledge.NeuralCompositionMetricV6 {
					continue
				}
				e.Resolution.Check[0].OutcomeError = .01
			}
			before := continuationBytes(t, r)
			seq := r.LastSequence()
			_, _, err = r.LearnCompositionWithDiagnosis(e, at)
			if err == nil || seq != r.LastSequence() || !bytes.Equal(before, continuationBytes(t, r)) {
				t.Fatalf("%s/%s accepted or mutated", version, fault)
			}
		}
	}
}
func TestMetricCanonicalRestoreRejectsWrongContractAndOrphans(t *testing.T) {
	r, _, dir, _ := mNew(t, knowledge.NeuralCompositionMetricV6)
	fit := nGrid([]float64{-.8, -.4, .4, .8}, []float64{-.7, .7})
	check := nGrid([]float64{-.9, -.2, .2, .9}, []float64{-.5, .5})
	_ = mTeach(t, r, nLesson(0, 0, fit, check, func(p nXY) float64 { return p.x }), .001)
	path := filepath.Join(dir, "metric.json")
	compositionCheck(t, r.Checkpoint(path))
	raw, e := os.ReadFile(path)
	compositionCheck(t, e)
	var snapshot map[string]json.RawMessage
	compositionCheck(t, json.Unmarshal(raw, &snapshot))
	brain := snapshot["brain"]
	compositionCheck(t, knowledge.ValidateCompositionState(brain, true, knowledge.NeuralCompositionMetricV6))
	if e := knowledge.ValidateCompositionState(brain, true, knowledge.NeuralCompositionResolutionV5); e == nil {
		t.Fatal("metric state accepted under v5")
	}
	var altered map[string]any
	compositionCheck(t, json.Unmarshal(brain, &altered))
	state := altered["n_resolution"].(map[string]any)
	state["controls"] = []any{}
	b, e := json.Marshal(altered)
	compositionCheck(t, e)
	if e = knowledge.ValidateCompositionState(b, true, knowledge.NeuralCompositionMetricV6); e == nil {
		t.Fatal("orphan or missing error control accepted")
	}
	if _, _, e = r.LearnCompositionWithDiagnosis(knowledge.CompositionExposure{}, time.Time{}); e == nil {
		t.Fatal("empty lesson accepted")
	}
}
func TestMetricObservedOutcomeUncertaintyIsRetained(t *testing.T) {
	r, _, _, _ := mNew(t, knowledge.NeuralCompositionMetricV6)
	fit := nGrid([]float64{-.8, -.4, .4, .8}, []float64{-.7, .7})
	check := nGrid([]float64{-.9, -.2, .2, .9}, []float64{-.5, .5})
	e := nLesson(0, 0, fit, check, func(p nXY) float64 { return p.x })
	for i := range e.Resolution.Check {
		e.Resolution.Check[i].OutcomeError = .01
	}
	d := mTeach(t, r, e, .001)
	o := mRead(t, r, 0, nXY{.4, .2}, .001)
	if !d.Committed || o.Accepted || o.NResolution.ObservedCheckMaxError == nil || *o.NResolution.ObservedCheckMaxError < .01 {
		t.Fatal("observed outcome uncertainty was hidden")
	}
}
func TestMetricWorldRejectsMalformedRequests(t *testing.T) {
	r, _, _, _ := mNew(t, knowledge.NeuralCompositionMetricV6)
	d := NumericalDomain{ID: 9, StateDimensions: 2, ActionDimensions: 1, StepNS: int64(20 * time.Millisecond)}
	req := NumericalImaginationRequest{Domain: d, Start: NumericalFrame{Values: []float64{0, 0}, At: nBase}, Actions: make([][]float64, MaxNumericalRolloutSteps+1), Tolerances: []float64{.001, .001}}
	before := continuationBytes(t, r)
	if _, e := r.ImagineNumericalTrajectory(req); e == nil {
		t.Fatal("oversized rollout accepted")
	}
	if !bytes.Equal(before, continuationBytes(t, r)) {
		t.Fatal("bad rollout changed brain")
	}
}
