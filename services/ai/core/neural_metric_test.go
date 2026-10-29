package core

import (
	"bytes"
	"fmt"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/runtime"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMetricFacadeUsesActiveBrainAndRecovery(t *testing.T) {
	at := time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC)
	rt := runtime.NewBrainRuntime(knowledge.NewBrain())
	rt.SetClock(runtime.NewFixedClock(at))
	owner, e := runtime.NewRuntimeSupervisor(rt)
	if e != nil {
		t.Fatal(e)
	}
	h := &HorizonEngine{supervisor: owner}
	cfg := knowledge.DefaultCompositionConfig()
	cfg.Version = knowledge.NeuralCompositionMetricV6
	cfg.Initialization = "random"
	dir := t.TempDir()
	if e = h.InitializeNeuralComposition(cfg, filepath.Join(dir, "backup.json"), at); e != nil {
		t.Fatal(e)
	}
	domain := runtime.NumericalDomain{ID: 81, StateDimensions: 1, ActionDimensions: 1, StepNS: int64(20 * time.Millisecond)}
	tol := 1e-5
	for phase := 0; phase < 5; phase++ {
		lesson := runtime.NumericalTransitionLesson{ID: fmt.Sprintf("facade-%d", phase), Domain: domain, Output: 0, RequiredPrecision: &tol}
		for set, xs := range [][]float64{{-.6, -.2, .2, .6}, {-.7, -.3, .3, .7}} {
			for i, x := range xs {
				for j, a := range xs {
					before := at.Add(-time.Second)
					row := runtime.NumericalTransition{Before: runtime.NumericalFrame{Values: []float64{x}, At: before}, After: runtime.NumericalFrame{Values: []float64{.7*x + .2*a}, At: before.Add(20 * time.Millisecond)}, Action: []float64{a}, EpisodeID: fmt.Sprintf("facade-%d-%d-%d-%d", phase, set, i, j)}
					if set == 0 {
						lesson.Fit = append(lesson.Fit, row)
					} else {
						lesson.Check = append(lesson.Check, row)
					}
				}
			}
		}
		if _, receipt, e := h.LearnNumericalTransition(lesson, at.Add(time.Duration(phase)*time.Millisecond)); e != nil || receipt == nil {
			t.Fatalf("active learning %v", e)
		}
	}
	if e = owner.WithMutation(func(r *runtime.BrainRuntime) error {
		r.SetClock(runtime.NewFixedClock(at.Add(time.Second)))
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	req := runtime.NumericalImaginationRequest{Domain: domain, Start: runtime.NumericalFrame{Values: []float64{.31}, At: at}, Actions: [][]float64{{.1}, {-.2}, {.3}}, Tolerances: []float64{.001}}
	path := filepath.Join(dir, "saved.json")
	if e = h.SaveActiveBrain(path); e != nil {
		t.Fatal(e)
	}
	before, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	out, e := h.ImagineNumericalTrajectory(req)
	if e != nil || out.StopReason != "complete" || len(out.Steps) != 3 {
		t.Fatalf("managed imagination %v %+v", e, out)
	}
	actual := .31
	for i, step := range out.Steps {
		actual = .7*actual + .2*req.Actions[i][0]
		if math.Abs(step.State.Values[0]-actual) > .001 {
			t.Fatal("active owner forecast wrong")
		}
	}
	if h.Gateway != nil {
		t.Fatal("imagination created an execution gateway")
	}
	d, e := h.CompositionDiagnosis()
	if e != nil || d == nil {
		t.Fatal("active diagnosis unavailable")
	}
	if e = h.SaveActiveBrain(path); e != nil {
		t.Fatal(e)
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("managed imagination changed memory")
	}
	rt2 := runtime.NewBrainRuntime(knowledge.NewBrain())
	owner2, e := runtime.NewRuntimeSupervisor(rt2)
	if e != nil {
		t.Fatal(e)
	}
	h2 := &HorizonEngine{supervisor: owner2}
	if e = h2.LoadActiveBrain(path); e != nil {
		t.Fatal(e)
	}
	restored, e := h2.ImagineNumericalTrajectory(req)
	if e != nil || len(restored.Steps) != 3 {
		t.Fatal("restored managed world model unavailable")
	}
	for i := range out.Steps {
		if restored.Steps[i].State.Values[0] != out.Steps[i].State.Values[0] {
			t.Fatal("managed recovery forecast differs")
		}
	}
}
