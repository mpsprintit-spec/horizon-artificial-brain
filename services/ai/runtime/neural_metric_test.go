package runtime

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func mNew(t *testing.T, version string) (*BrainRuntime, *EventLog, string, string) {
	t.Helper()
	r := NewBrainRuntime(knowledge.NewBrain())
	r.SetClock(NewFixedClock(nBase))
	dir := t.TempDir()
	log, e := OpenEventLog(filepath.Join(dir, "events.jsonl"))
	compositionCheck(t, e)
	t.Cleanup(func() { _ = log.Close() })
	r.SetEventLog(log)
	w, e := r.PlanNeuralWrittenGrounding()
	compositionCheck(t, e)
	compositionCheck(t, r.ApplyNeuralWrittenGrounding(w, filepath.Join(dir, "before-written.json"), nBase))
	c := knowledge.DefaultCompositionConfig()
	c.Version = version
	c.Initialization = "random"
	p, e := r.PlanNeuralComposition(c)
	compositionCheck(t, e)
	compositionCheck(t, r.ApplyNeuralComposition(p, filepath.Join(dir, "before-composition.json"), nBase))
	start := filepath.Join(dir, "start.json")
	compositionCheck(t, r.Checkpoint(start))
	return r, log, dir, start
}
func mTeach(t *testing.T, r *BrainRuntime, e knowledge.CompositionExposure, tol float64) knowledge.NDiagnosis {
	t.Helper()
	e.Motor = []float64{0, 0, 0, 0}
	e.After = nScene(e.After.ObservedAt)
	if tol > 0 {
		e.Input.RequiredPrecision = &tol
	}
	r.SetClock(NewFixedClock(e.After.ObservedAt))
	_, receipt, err := r.LearnCompositionWithDiagnosis(e, e.After.ObservedAt)
	compositionCheck(t, err)
	if receipt == nil {
		t.Fatal("no numerical receipt")
	}
	return receipt.Diagnosis
}
func mRead(t *testing.T, r *BrainRuntime, q int, p nXY, tol float64) knowledge.CompositionReadout {
	t.Helper()
	in := nInputFixture(q, p, nBase.Add(24*time.Hour))
	if tol > 0 {
		in.RequiredPrecision = &tol
	}
	v, e := r.ReadComposition(in, in.Scene.ObservedAt)
	compositionCheck(t, e)
	return v
}
func mSave(t *testing.T, name string, v any) {
	t.Helper()
	root := os.Getenv("HORIZON_M_RESULTS")
	if root == "" {
		return
	}
	compositionCheck(t, os.MkdirAll(root, 0700))
	b, e := json.MarshalIndent(v, "", "  ")
	compositionCheck(t, e)
	compositionCheck(t, os.WriteFile(filepath.Join(root, name), append(b, '\n'), 0600))
}
func TestMetricInformationReuseAndPrecision(t *testing.T) {
	r, _, _, _ := mNew(t, knowledge.NeuralCompositionMetricV6)
	fit := nGrid([]float64{-.8, -.4, .4, .8}, []float64{-.7, .7})
	check := nGrid([]float64{-.9, -.2, .2, .9}, []float64{-.5, .5})
	sign := func(p nXY) float64 { return nSign(p.x) }
	metric := func(p nXY) float64 { return p.x }
	d0 := mTeach(t, r, nLesson(0, 0, fit, check, sign), 0)
	if !d0.Committed || len(r.brain.BrainState.NResolution.Features) != 1 {
		t.Fatalf("foundation %+v", d0)
	}
	ds := []knowledge.NDiagnosis{d0}
	for i := 1; i <= 6; i++ {
		ds = append(ds, mTeach(t, r, nLesson(1, i, fit, check, metric), 1e-6))
	}
	a, b := mRead(t, r, 1, nXY{.100012, -.9}, 1e-6), mRead(t, r, 1, nXY{.200014, .9}, 1e-6)
	if !a.Accepted || !b.Accepted || math.Abs(a.NResolution.Value-.100012) > 1e-6 || math.Abs(b.NResolution.Value-.200014) > 1e-6 || len(r.brain.BrainState.NResolution.Features) != 1 {
		t.Fatalf("metric readout did not reuse precise N: a=%+v b=%+v diagnoses=%+v", a, b, ds)
	}
	if !a.NResolution.MetricReadout || a.NResolution.IsCalibratedProbability {
		t.Fatal("metric provenance missing")
	}
	// Confidence for an absolute precision request is never a magnitude/probability.
	if a.Confidence != 0 {
		t.Fatal("invented metric confidence")
	}
	old := mRead(t, r, 0, nXY{-.1, .4}, 0)
	if !old.Accepted || old.NResolution.Value >= 0 {
		t.Fatal("old directional meaning lost")
	}
	noisy := nInputFixture(1, nXY{.1, .5}, nBase.Add(24*time.Hour))
	tol := 1e-6
	noisy.RequiredPrecision = &tol
	noisy.MeasurementError = [][]float64{{.01, 0, 0, 0, 0, 0}}
	uncertain, e := r.ReadComposition(noisy, noisy.Scene.ObservedAt)
	compositionCheck(t, e)
	if uncertain.Accepted || *uncertain.NResolution.EstimatedAbsError < .009 {
		t.Fatal("supplied measurement uncertainty discarded")
	}
	before := continuationBytes(t, r)
	_ = mRead(t, r, 1, nXY{.3, -.3}, 1e-6)
	if !bytes.Equal(before, continuationBytes(t, r)) {
		t.Fatal("preview mutated")
	}
	in := nInputFixture(1, nXY{.1, .3}, nBase.Add(25*time.Hour))
	in.RequiredPrecision = &tol
	in.MeasurementError = [][]float64{{.001, 0, 0, 0, 0, 0}}
	_, e = r.ObserveCompositionInput(CompositionPerception{ID: "metric-sensor", Source: "numeric-sensor", Input: in}, in.Scene.ObservedAt)
	compositionCheck(t, e)
	r.SetClock(NewFixedClock(in.Scene.ObservedAt))
	thought, e := r.CognitiveThink(1)
	compositionCheck(t, e)
	if thought.Composition == nil || thought.Composition.NResolution.RequestedPrecision == nil || thought.Composition.Accepted {
		t.Fatal("native thinking lost the precision contract")
	}
	mSave(t, "metric_reuse.json", map[string]any{"diagnoses": ds, "a": a, "b": b, "old_direction": old, "uncertain": uncertain, "learned_n": 1, "native_think": thought.Composition})
}
func TestMetricFineContrastAndConflictDiagnosis(t *testing.T) {
	r, _, _, _ := mNew(t, knowledge.NeuralCompositionMetricV6)
	sx := func(p nXY) float64 { return nSign(p.x) }
	_ = mTeach(t, r, nLesson(0, 0, nGrid([]float64{-.8, -.4, .4, .8}, []float64{-.7, .7}), nGrid([]float64{-.9, -.2, .2, .9}, []float64{-.5, .5}), sx), 0)
	d := mTeach(t, r, nLesson(0, 1, nGrid([]float64{-1.5e-12, -1e-12, 1e-12, 1.5e-12}, []float64{-.7, .7}), nGrid([]float64{-1.2e-12, -.8e-12, .8e-12, 1.2e-12}, []float64{-.5, .5}), sx), 0)
	pos, neg := mRead(t, r, 0, nXY{1e-12, .3}, 0), mRead(t, r, 0, nXY{-1e-12, .3}, 0)
	if !pos.Accepted || !neg.Accepted || pos.NResolution.Value <= 0 || neg.NResolution.Value >= 0 {
		t.Fatalf("fine contrast failed: %+v %+v diagnosis %+v", pos, neg, d)
	}
	// Same actual encoding with contradictory outcomes remains unresolved.
	n0 := len(r.brain.Registry.Nodes())
	exp := nLesson(0, 2, nGrid([]float64{-.9, -.2, .2, .9}, []float64{-.5, .5}), nGrid([]float64{-.9, -.2, .2, .9}, []float64{-.5, .5}), func(p nXY) float64 { return -sx(p) })
	conflict := mTeach(t, r, exp, 0)
	if conflict.Committed || conflict.Cause != "conflicting-observations" || len(conflict.Hypotheses) != 3 || len(conflict.Probes) < 1 || len(r.brain.Registry.Nodes()) != n0 {
		t.Fatalf("conflict falsely resolved %+v", conflict)
	}
	inspect, e := r.CompositionDiagnosis()
	compositionCheck(t, e)
	inspect.Hypotheses[0] = "caller mutation"
	inspect.Probes[0].Evidence = "caller mutation"
	again, e := r.CompositionDiagnosis()
	compositionCheck(t, e)
	if again.Hypotheses[0] == "caller mutation" || again.Probes[0].Evidence == "caller mutation" {
		t.Fatal("diagnosis aliases canonical state")
	}
	mSave(t, "fine_and_conflict.json", map[string]any{"fine": d, "positive": pos, "negative": neg, "conflict": conflict})
}
func TestMetricContextualRetention(t *testing.T) {
	r, _, _, _ := mNew(t, knowledge.NeuralCompositionMetricV6)
	fit := nGrid([]float64{-.8, -.4, .4, .8}, []float64{-.7, .7})
	check := nGrid([]float64{-.9, -.2, .2, .9}, []float64{-.5, .5})
	sx := func(p nXY) float64 { return nSign(p.x) }
	sy := func(p nXY) float64 { return nSign(p.y) }
	_ = mTeach(t, r, nLesson(0, 0, fit, check, sx), 0)
	_ = mTeach(t, r, nLesson(1, 1, fit, check, sy), 0)
	_ = mTeach(t, r, nLesson(2, 2, nGrid([]float64{-.8, -.4, .4, .8}, []float64{-.7}), nGrid([]float64{-.9, -.2, .2, .9}, []float64{-.5}), sx), 0)
	d := mTeach(t, r, nLesson(2, 3, nGrid([]float64{-.8, -.4, .4, .8}, []float64{.7}), nGrid([]float64{-.9, -.2, .2, .9}, []float64{.5}), func(p nXY) float64 { return -sx(p) }), 0)
	correct := 0
	for _, p := range nGrid([]float64{-.43, -.21, .17, .61}, []float64{-.49, .37}) {
		o := mRead(t, r, 2, p, 0)
		if o.Accepted && nSign(o.NResolution.Value) == -nSign(p.x*p.y) {
			correct++
		}
	}
	if correct != 8 {
		for _, p := range nGrid([]float64{-.43, .61}, []float64{-.49, .37}) {
			o := mRead(t, r, 2, p, 0)
			t.Logf("context %+v -> %+v", p, o)
		}
		t.Fatalf("conditional retention %d/8 diagnosis %+v", correct, d)
	}
	// Retention and new growth are decided by function, not geometry alone.
	if d.Grown == 0 || d.Cause != "missing-relation" {
		t.Fatalf("new conditional meaning not composed %+v", d)
	}
	mSave(t, "context_retention.json", map[string]any{"diagnosis": d, "correct": correct, "total": 8, "learned_n": len(r.brain.BrainState.NResolution.Features)})
}
func mPhysicalStep(state, action []float64) []float64 {
	v := state[1] + .02*(action[0]-.05*state[1])
	return []float64{state[0] + .02*v, v}
}
func mWorldLesson(phase, output int, rng *rand.Rand) NumericalTransitionLesson {
	d := NumericalDomain{ID: 73, StateDimensions: 2, ActionDimensions: 1, StepNS: int64(20 * time.Millisecond)}
	tol := 1e-7
	l := NumericalTransitionLesson{ID: fmt.Sprintf("plant-%d", phase), Domain: d, Output: output, RequiredPrecision: &tol}
	for set := 0; set < 2; set++ {
		for i := 0; i < 32; i++ {
			at := nBase.Add(time.Duration(phase+1) * time.Hour).Add(-time.Duration(100-i) * time.Second)
			s := []float64{rng.Float64()*1.2 - .6, rng.Float64()*.8 - .4}
			a := []float64{rng.Float64()*1.4 - .7}
			r := NumericalTransition{Before: NumericalFrame{Values: s, At: at}, After: NumericalFrame{Values: mPhysicalStep(s, a), At: at.Add(20 * time.Millisecond)}, Action: a, EpisodeID: fmt.Sprintf("world-%d-%d-%d", phase, set, i)}
			if set == 0 {
				l.Fit = append(l.Fit, r)
			} else {
				l.Check = append(l.Check, r)
			}
		}
	}
	return l
}
func TestMetricImaginedFramesCannotBecomeObservedLabels(t *testing.T) {
	l := mWorldLesson(0, 0, rand.New(rand.NewSource(7)))
	l.Check[0].After.Hypothetical = true
	if _, e := CompileNumericalTransitionTarget(l, nBase.Add(time.Hour)); e == nil {
		t.Fatal("imagined after-state accepted as evidence")
	}
}
func TestMetricNativeTransitionRolloutRecovery(t *testing.T) {
	r, log, dir, start := mNew(t, knowledge.NeuralCompositionMetricV6)
	ds := []any{}
	for phase := 0; phase < 8; phase++ {
		for output := 0; output < 2; output++ {
			rng := rand.New(rand.NewSource(int64(7 + phase)))
			l := mWorldLesson(phase, output, rng)
			at := nBase.Add(time.Duration(phase+1) * time.Hour)
			r.SetClock(NewFixedClock(at))
			_, receipt, e := r.LearnNumericalTransition(l, at)
			compositionCheck(t, e)
			ds = append(ds, receipt)
		}
	}
	req := NumericalImaginationRequest{Domain: NumericalDomain{ID: 73, StateDimensions: 2, ActionDimensions: 1, StepNS: int64(20 * time.Millisecond)}, Start: NumericalFrame{Values: []float64{.15, .25}, At: nBase.Add(24 * time.Hour)}, Tolerances: []float64{.001, .001}}
	for i := 0; i < 16; i++ {
		req.Actions = append(req.Actions, []float64{.3 * math.Sin(float64(i)*.41)})
	}
	checkpoint := filepath.Join(dir, "metric-world.json")
	compositionCheck(t, r.Checkpoint(checkpoint))
	before, e := os.ReadFile(checkpoint)
	compositionCheck(t, e)
	seq := r.LastSequence()
	out, e := r.ImagineNumericalTrajectory(req)
	compositionCheck(t, e)
	if out.StopReason != "complete" || len(out.Steps) != 16 {
		t.Fatalf("rollout incomplete %+v lessons %+v", out, ds)
	}
	actual := slices.Clone(req.Start.Values)
	maxError := 0.
	for i, step := range out.Steps {
		actual = mPhysicalStep(actual, req.Actions[i])
		for j, x := range actual {
			maxError = math.Max(maxError, math.Abs(x-step.State.Values[j]))
		}
	}
	if maxError > .001 {
		t.Fatalf("rollout error %g", maxError)
	}
	after := filepath.Join(dir, "metric-world-after.json")
	compositionCheck(t, r.Checkpoint(after))
	b, e := os.ReadFile(after)
	compositionCheck(t, e)
	if !bytes.Equal(before, b) || r.LastSequence() != seq {
		t.Fatal("imagination learned or mutated state")
	}
	restored := NewBrainRuntime(knowledge.NewBrain())
	compositionCheck(t, restored.RestoreCheckpoint(checkpoint))
	restored.SetClock(NewFixedClock(nBase.Add(8 * time.Hour)))
	replayed, e := RebuildRuntimeFromCheckpointAndJournal(log, start)
	compositionCheck(t, e)
	for _, other := range []*BrainRuntime{restored, replayed} {
		a, e := other.ImagineNumericalTrajectory(req)
		compositionCheck(t, e)
		bb, _ := json.Marshal(out)
		cc, _ := json.Marshal(a)
		if !bytes.Equal(bb, cc) {
			t.Fatal("native prediction changed after restore or full replay")
		}
	}
	// A transition model that has never been taught supplies no guessed trajectory.
	unknown := req
	unknown.Domain.ID = 74
	u, e := r.ImagineNumericalTrajectory(unknown)
	compositionCheck(t, e)
	if len(u.Steps) != 0 || u.StopReason != "untrained-transition" {
		t.Fatal("unknown transition fabricated")
	}
	// Strict precision can stop a rollout while exposing its reason.
	narrow := req
	narrow.Tolerances = []float64{1e-16, 1e-16}
	stopped, e := r.ImagineNumericalTrajectory(narrow)
	compositionCheck(t, e)
	if stopped.StopReason != "precision-not-met" {
		t.Fatal("precision contract ignored")
	}
	// Remove the learned velocity readout on a disposable restored Brain.
	// This deliberate intervention invalidates its empirical calibration.
	qids := restored.brain.BrainState.NResolution.Heads[1].Query
	for _, f := range restored.brain.BrainState.NResolution.Features {
		node := restored.brain.Registry.GetByID(f.Node)
		for _, q := range qids {
			for _, edge := range node.Synapses[q] {
				edge.Dynamic.Weight = 0
			}
		}
	}
	damaged, e := restored.ImagineNumericalTrajectory(req)
	compositionCheck(t, e)
	ablationError := 0.
	truth := slices.Clone(req.Start.Values)
	for i, step := range damaged.Steps {
		truth = mPhysicalStep(truth, req.Actions[i])
		for j, x := range truth {
			ablationError = math.Max(ablationError, math.Abs(x-step.State.Values[j]))
		}
	}
	if ablationError < .01 {
		t.Fatal("learned synaptic transition had no causal contribution")
	}
	compositionCheck(t, log.Close())
	root := os.Getenv("HORIZON_M_RESULTS")
	if root != "" {
		compositionCheck(t, os.MkdirAll(root, 0700))
		for _, f := range []string{"metric-world.json", "events.jsonl", "start.json"} {
			b, e := os.ReadFile(filepath.Join(dir, f))
			compositionCheck(t, e)
			compositionCheck(t, os.WriteFile(filepath.Join(root, f), b, 0600))
		}
	}
	mSave(t, "world_rollout.json", map[string]any{"ablation_error": ablationError, "ablation": damaged, "receipts": ds, "request": req, "trajectory": out, "max_absolute_error": maxError, "full_restore_and_replay_equal": true, "readonly": true, "untrained": u, "precision_stop": stopped, "learned_n": len(r.brain.BrainState.NResolution.Features)})
}
