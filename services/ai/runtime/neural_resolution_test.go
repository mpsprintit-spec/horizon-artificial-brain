package runtime

import (
	"encoding/json"
	"fmt"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

var nBase = time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC)

type nXY struct{ x, y float64 }

func nScene(at time.Time) knowledge.CompositionScene {
	return knowledge.CompositionScene{ObservedAt: at, Point: []float64{0, 0, 1}, Patches: []knowledge.CompositionPatch{{Values: []float64{.4, -.2, .6, -1, 0}, XY: []float64{0, 0}}}}
}
func nInputFixture(q int, p nXY, at time.Time) knowledge.CompositionInput {
	v := make([]float64, 16)
	v[q%12] = 1
	v[12] = -1
	if q >= 12 {
		v[12] = 1
	}
	v[15] = .125
	return knowledge.CompositionInput{Words: [][]float64{v}, Scene: nScene(at), Sequence: [][]float64{{p.x, p.y, 0, 0, 0, 0, 1}}}
}
func nNewRuntime(t *testing.T) (*BrainRuntime, *EventLog, string, string) {
	t.Helper()
	r := NewBrainRuntime(knowledge.NewBrain())
	r.SetClock(NewFixedClock(nBase))
	dir := t.TempDir()
	log, e := OpenEventLog(filepath.Join(dir, "events.jsonl"))
	compositionCheck(t, e)
	r.SetEventLog(log)
	t.Cleanup(func() { _ = log.Close() })
	w, e := r.PlanNeuralWrittenGrounding()
	compositionCheck(t, e)
	compositionCheck(t, r.ApplyNeuralWrittenGrounding(w, filepath.Join(dir, "before-written.json"), nBase))
	cfg := knowledge.DefaultCompositionConfig()
	cfg.Version = knowledge.NeuralCompositionResolutionV5
	cfg.Initialization = "random"
	p, e := r.PlanNeuralComposition(cfg)
	compositionCheck(t, e)
	compositionCheck(t, r.ApplyNeuralComposition(p, filepath.Join(dir, "before-composition.json"), nBase))
	start := filepath.Join(dir, "start.json")
	compositionCheck(t, r.Checkpoint(start))
	return r, log, dir, start
}
func nGrid(mags, nuisance []float64) []nXY {
	out := []nXY{}
	for _, x := range mags {
		for _, y := range nuisance {
			out = append(out, nXY{x, y})
		}
	}
	return out
}
func nLesson(q, phase int, fit, check []nXY, truth func(nXY) float64) knowledge.CompositionExposure {
	at := nBase.Add(time.Duration(phase+1) * time.Hour)
	batch := &knowledge.NResolutionLesson{}
	for set, rows := range [][]nXY{fit, check} {
		for i, p := range rows {
			input := nInputFixture(q, p, at.Add(-time.Duration(len(rows)-i)*time.Millisecond))
			o := knowledge.NObservation{Input: input, Outcome: truth(p), ObservedAt: input.Scene.ObservedAt.Add(time.Nanosecond), EpisodeID: fmt.Sprintf("phase-%d-set-%d-episode-%d", phase, set, i)}
			if set == 0 {
				batch.Fit = append(batch.Fit, o)
			} else {
				batch.Check = append(batch.Check, o)
			}
		}
	}
	e := knowledge.CompositionExposure{ID: fmt.Sprintf("n-lesson-%d", phase), Source: "controlled-numerical-experience", EpisodeID: fmt.Sprint(phase), IndependenceGroup: fmt.Sprintf("phase-%d", phase), CausalGroup: fmt.Sprint(phase), Reliability: 1, Input: batch.Fit[0].Input, Resolution: batch, Motor: []float64{0, 0, truth(fit[0]), 1}, After: nScene(at)}
	e.After.Patches[0].Values[3] = truth(fit[0])
	return e
}
func nTeach(t *testing.T, r *BrainRuntime, e knowledge.CompositionExposure) knowledge.NDiagnosis {
	t.Helper()
	r.SetClock(NewFixedClock(e.After.ObservedAt))
	_, err := r.LearnComposition(e, e.After.ObservedAt)
	compositionCheck(t, err)
	return r.brain.BrainState.NResolution.Last
}
func nAnswer(t *testing.T, r *BrainRuntime, q int, p nXY) knowledge.CompositionReadout {
	t.Helper()
	input := nInputFixture(q, p, nBase.Add(24*time.Hour))
	out, e := r.ReadComposition(input, input.Scene.ObservedAt)
	compositionCheck(t, e)
	return out
}
func nSign(x float64) float64 {
	if x > 0 {
		return 1
	}
	return -1
}
func nSaveResult(t *testing.T, name string, v any) {
	t.Helper()
	root := os.Getenv("HORIZON_N_RESULTS")
	if root == "" {
		return
	}
	compositionCheck(t, os.MkdirAll(root, 0700))
	b, e := json.MarshalIndent(v, "", "  ")
	compositionCheck(t, e)
	compositionCheck(t, os.WriteFile(filepath.Join(root, name), append(b, '\n'), 0600))
}

func TestNResolutionGrowthIdentityAndComposition(t *testing.T) {
	r, log, dir, start := nNewRuntime(t)
	fit := nGrid([]float64{-.7, -.5, .5, .7}, []float64{-.8, .8})
	check := nGrid([]float64{-.9, -.3, .3, .9}, []float64{-.6, .6})
	xTruth := func(p nXY) float64 { return nSign(p.x) }
	d0 := nTeach(t, r, nLesson(0, 0, fit, check, xTruth))
	if !d0.Committed || d0.Cause != "missing-direction" || len(r.brain.BrainState.NResolution.Features) != 1 {
		t.Fatalf("first understanding: %+v", d0)
	}
	initial := nAnswer(t, r, 0, nXY{.0001, .8})
	if initial.Accepted || initial.NResolution.Units[0].Code != 0 {
		t.Fatalf("coarse N should leave tiny distinction unresolved: %+v", initial)
	}
	fine := nGrid([]float64{-.00015, -.0001, .0001, .00015}, []float64{-.8, .8})
	fineCheck := nGrid([]float64{-.00012, -.00008, .00008, .00012}, []float64{-.6, .6})
	d1 := nTeach(t, r, nLesson(0, 1, fine, fineCheck, xTruth))
	if !d1.Committed || d1.Cause != "resolution" || d1.Grown != 0 || d1.ResolutionAfter >= d1.ResolutionBefore || len(r.brain.BrainState.NResolution.Features) != 1 {
		t.Fatalf("precision should improve without another direction: %+v", d1)
	}
	pos, neg := nAnswer(t, r, 0, nXY{.0001, .8}), nAnswer(t, r, 0, nXY{-.0001, .8})
	if !pos.Accepted || !neg.Accepted || pos.NResolution.Value <= 0 || neg.NResolution.Value >= 0 || pos.NResolution.Units[0].Code == neg.NResolution.Units[0].Code {
		t.Fatal("important tiny contrast was merged")
	}
	a, b := nAnswer(t, r, 0, nXY{.3, -.9}), nAnswer(t, r, 0, nXY{.3, .9})
	if !reflect.DeepEqual(a.NResolution.Units, b.NResolution.Units) || a.NResolution.Value != b.NResolution.Value {
		t.Fatal("irrelevant appearance changed the learned N")
	}
	d2 := nTeach(t, r, nLesson(1, 2, fit, check, xTruth))
	if !d2.Committed || d2.Cause != "readout" || len(r.brain.BrainState.NResolution.Features) != 1 {
		t.Fatalf("another query for the same information duplicated N: %+v", d2)
	}
	// The previously irrelevant coordinate becomes relevant under another query.
	yTruth := func(p nXY) float64 { return nSign(p.y) }
	d3 := nTeach(t, r, nLesson(2, 3, fit, check, yTruth))
	if !d3.Committed || d3.Cause != "missing-direction" || len(r.brain.BrainState.NResolution.Features) != 2 {
		t.Fatalf("new independent information: %+v", d3)
	}
	relation := func(p nXY) float64 { return nSign(p.x * p.y) }
	d4 := nTeach(t, r, nLesson(3, 4, fit, check, relation))
	if !d4.Committed || d4.Cause != "missing-relation" || len(r.brain.BrainState.NResolution.Features) != 3 {
		t.Fatalf("composed understanding: %+v", d4)
	}
	held := nGrid([]float64{-.43, -.21, .17, .61}, []float64{-.49, .37})
	correct := [4]int{}
	for _, p := range held {
		for q, f := range []func(nXY) float64{xTruth, xTruth, yTruth, relation} {
			out := nAnswer(t, r, q, p)
			if out.Accepted && nSign(out.NResolution.Value) == f(p) {
				correct[q]++
			}
		}
	}
	for q, v := range correct {
		if v != len(held) {
			t.Fatalf("held-out q%d: %d/%d", q, v, len(held))
		}
	}
	before := continuationBytes(t, r)
	_ = nAnswer(t, r, 3, nXY{.11, -.31})
	if !reflect.DeepEqual(before, continuationBytes(t, r)) {
		t.Fatal("reading allocated or changed neural state")
	}
	input := nInputFixture(3, nXY{.11, -.31}, nBase.Add(6*time.Hour))
	r.SetClock(NewFixedClock(input.Scene.ObservedAt))
	nodesBefore := len(r.brain.Registry.Nodes())
	_, e := r.ObserveCompositionInput(CompositionPerception{ID: "actual-new-frame", Source: "controlled-numeric-sensor", Input: input}, input.Scene.ObservedAt)
	compositionCheck(t, e)
	if len(r.brain.Registry.Nodes()) != nodesBefore {
		t.Fatal("perception grew neurons")
	}
	r.mu.Lock()
	continued, e := r.compositionReadoutLocked(input.Scene.ObservedAt)
	r.mu.Unlock()
	compositionCheck(t, e)
	direct, e := r.ReadComposition(input, input.Scene.ObservedAt)
	compositionCheck(t, e)
	if !reflect.DeepEqual(*continued, direct) {
		t.Fatal("ordinary continuation did not use the same sequence")
	}
	thought, e := r.CognitiveThink(1)
	compositionCheck(t, e)
	if thought.Composition == nil || !reflect.DeepEqual(thought.Composition.NResolution, direct.NResolution) {
		t.Fatal("ordinary thinking lost numerical sequence understanding")
	}
	if thought.Activations[r.brain.BrainState.NResolution.Features[0].Node] <= 0 {
		t.Fatal("learned N state never reached the native thought activation")
	}
	afterThought, e := r.ReadComposition(input, input.Scene.ObservedAt)
	compositionCheck(t, e)
	if !reflect.DeepEqual(afterThought.NResolution, direct.NResolution) {
		t.Fatal("a thought cycle silently changed diagnosed N parameters")
	}
	direct = afterThought
	final := filepath.Join(dir, "final.json")
	compositionCheck(t, r.Checkpoint(final))
	expected := continuationBytes(t, r)
	restored := NewBrainRuntime(knowledge.NewBrain())
	compositionCheck(t, restored.RestoreCheckpoint(final))
	restored.SetClock(NewFixedClock(input.Scene.ObservedAt))
	replayed, e := RebuildRuntimeFromCheckpointAndJournal(log, start)
	compositionCheck(t, e)
	replayed.SetClock(NewFixedClock(input.Scene.ObservedAt))
	for _, v := range []*BrainRuntime{restored, replayed} {
		if !reflect.DeepEqual(expected, continuationBytes(t, v)) {
			t.Fatal("full N checkpoint/replay differs")
		}
		out, e := v.ReadComposition(input, input.Scene.ObservedAt)
		compositionCheck(t, e)
		if !reflect.DeepEqual(out, direct) {
			t.Fatal("N readout changed after recovery")
		}
	}
	// Numeric contribution ablation on a disposable restored Brain.
	rel := restored.brain.BrainState.NResolution.Features[2].Node
	query := restored.brain.BrainState.NResolution.Heads[3].Query
	source := restored.brain.Registry.GetByID(rel)
	for _, id := range query {
		for _, edge := range source.Synapses[id] {
			edge.Dynamic.Weight = 0
		}
	}
	abl := nAnswer(t, restored, 3, nXY{.11, -.31})
	if abl.Accepted || math.Abs(abl.NResolution.Value) > 1e-6 {
		t.Fatal("new relational neuron had no causal contribution")
	}
	compositionCheck(t, log.Close())
	root := os.Getenv("HORIZON_N_RESULTS")
	if root != "" {
		compositionCheck(t, os.MkdirAll(root, 0700))
		for _, f := range []string{"start.json", "final.json", "events.jsonl"} {
			b, e := os.ReadFile(filepath.Join(dir, f))
			compositionCheck(t, e)
			compositionCheck(t, os.WriteFile(filepath.Join(root, f), b, 0600))
		}
	}
	nSaveResult(t, "mechanism_results.json", map[string]any{"diagnoses": []knowledge.NDiagnosis{d0, d1, d2, d3, d4}, "fine_positive": pos, "fine_negative": neg, "appearance_a": a.NResolution, "appearance_b": b.NResolution, "heldout_cases_per_question": len(held), "heldout_correct": correct, "learned_feature_units": 3, "total_neural_units": len(r.brain.Registry.Nodes()), "native_continuation_equal": true, "native_restore_equal": true, "native_full_journal_replay_equal": true, "readonly_no_growth": true, "relation_ablation_value": abl.NResolution.Value, "is_general_understanding_proof": false})
	t.Logf("resolution %.9g -> %.9g; 3 learned N circuits; held-out %v/%d; full restore/replay and contribution ablation passed", d1.ResolutionBefore, d1.ResolutionAfter, correct, len(held))
}

func TestNResolutionObservedZeroIsDifferentFromMissingInput(t *testing.T) {
	r, _, _, _ := nNewRuntime(t)
	at := nBase.Add(time.Hour)
	lesson := &knowledge.NResolutionLesson{}
	for set := 0; set < 2; set++ {
		for i := 0; i < 8; i++ {
			in := nInputFixture(0, nXY{}, at.Add(-time.Second))
			present := i%2 == 0
			y := 1.
			if !present {
				in.Sequence[0][6] = 0
				y = -1
			}
			o := knowledge.NObservation{Input: in, Outcome: y, ObservedAt: in.Scene.ObservedAt, EpisodeID: fmt.Sprintf("presence-%d-%d", set, i)}
			if set == 0 {
				lesson.Fit = append(lesson.Fit, o)
			} else {
				lesson.Check = append(lesson.Check, o)
			}
		}
	}
	e := knowledge.CompositionExposure{ID: "presence-lesson", Source: "observed-numeric-presence", EpisodeID: "presence", IndependenceGroup: "presence-intervention", CausalGroup: "presence", Reliability: 1, Input: lesson.Fit[0].Input, Resolution: lesson, Motor: []float64{0, 0, 1, 1}, After: nScene(at)}
	e.After.Patches[0].Values[3] = 1
	d := nTeach(t, r, e)
	if !d.Committed || d.Grown == 0 {
		t.Fatalf("missingness collapsed with a measured zero: %+v", d)
	}
	a := nInputFixture(0, nXY{}, at)
	b := nInputFixture(0, nXY{}, at)
	b.Sequence[0][6] = 0
	pa, err := r.ReadComposition(a, at)
	compositionCheck(t, err)
	pb, err := r.ReadComposition(b, at)
	compositionCheck(t, err)
	if !pa.Accepted || !pb.Accepted || pa.NResolution.Value <= 0 || pb.NResolution.Value >= 0 {
		t.Fatal("learned presence distinction was lost")
	}
	nSaveResult(t, "presence_resolution_results.json", map[string]any{"diagnosis": d, "observed_zero_value": pa.NResolution.Value, "missing_value": pb.NResolution.Value, "different_numeric_states": true})
}

func TestNResolutionSameAxisCanNeedAnotherMeaningBoundary(t *testing.T) {
	r, _, _, _ := nNewRuntime(t)
	fit := nGrid([]float64{-.7, -.5, .5, .7}, []float64{-.8, .8})
	check := nGrid([]float64{-.9, -.3, .3, .9}, []float64{-.6, .6})
	truth := func(p nXY) float64 { return nSign(p.x) }
	nTeach(t, r, nLesson(0, 0, fit, check, truth))
	old := r.brain.BrainState.NResolution.Features[0].Node
	f := nGrid([]float64{.25, .35, .65, .75}, []float64{-.8, .8})
	c := nGrid([]float64{.15, .45, .55, .85}, []float64{-.6, .6})
	shifted := func(p nXY) float64 { return nSign(p.x - .5) }
	d := nTeach(t, r, nLesson(1, 1, f, c, shifted))
	if !d.Committed || d.Cause != "missing-boundary" || len(r.brain.BrainState.NResolution.Features) != 2 {
		t.Fatalf("geometric overlap incorrectly prevented a new distinction: %+v", d)
	}
	for _, p := range []nXY{{.4, .2}, {.6, -.2}} {
		a, b := nAnswer(t, r, 0, p), nAnswer(t, r, 1, p)
		if nSign(a.NResolution.Value) != truth(p) || nSign(b.NResolution.Value) != shifted(p) {
			t.Fatal("same appearance with different query/context was merged")
		}
		if a.NResolution.Units[0].Node != old || b.NResolution.Units[0].Node != old {
			t.Fatal("old numerical identity was replaced")
		}
	}
	inverse := func(p nXY) float64 { return -nSign(p.x) }
	reuse := nTeach(t, r, nLesson(2, 2, fit, check, inverse))
	if !reuse.Committed || reuse.Cause != "readout" || len(r.brain.BrainState.NResolution.Features) != 2 {
		t.Fatalf("opposite reading duplicated existing information: %+v", reuse)
	}
	newExamples := nGrid([]float64{.19, .38, .62, .91}, []float64{-.47, .29})
	correct := [3]int{}
	for _, p := range newExamples {
		for q, f := range []func(nXY) float64{truth, shifted, inverse} {
			out := nAnswer(t, r, q, p)
			if out.Accepted && nSign(out.NResolution.Value) == f(p) {
				correct[q]++
			}
		}
	}
	for _, v := range correct {
		if v != len(newExamples) {
			t.Fatal("conditional distinction failed on new examples")
		}
	}
	nSaveResult(t, "conditional_identity_results.json", map[string]any{"new_boundary": d, "inverse_readout_reuse": reuse, "old_n_identity_preserved": old, "heldout_correct": correct, "heldout_per_question": len(newExamples), "raw_axis_overlap_did_not_decide_identity": true})
}

func TestNResolutionBalancedRetentionAndTransformStress(t *testing.T) {
	r, _, _, _ := nNewRuntime(t)
	fit := nGrid([]float64{-.7, -.5, .5, .7}, []float64{-.8, .8})
	check := nGrid([]float64{-.9, -.3, .3, .9}, []float64{-.6, .6})
	truth := func(p nXY) float64 { return nSign(p.x) }
	for q := 0; q < 20; q++ {
		nTeach(t, r, nLesson(q, q, fit, check, truth))
	}
	s := r.brain.BrainState.NResolution
	if len(s.Features) != 1 || len(s.Witnesses) > 128 {
		t.Fatal("repeated knowledge grew circuits or exceeded witness budget")
	}
	coverage := map[knowledge.NodeID]int{}
	for _, w := range s.Witnesses {
		coverage[w.Query[0]]++
	}
	if len(coverage) != 20 {
		t.Fatal("bounded retention lost a whole query")
	}
	correct, pairs := 0, 512
	for i := 0; i < pairs; i++ {
		x := .08 + .84*float64((i*73)%509)/509
		if i%2 == 0 {
			x = -x
		}
		y := float64((i*137)%511)/255.5 - 1
		a, b := nAnswer(t, r, i%20, nXY{x, y}), nAnswer(t, r, i%20, nXY{x, -y})
		if a.Accepted && b.Accepted && nSign(a.NResolution.Value) == nSign(x) && a.NResolution.Value == b.NResolution.Value && reflect.DeepEqual(a.NResolution.Units, b.NResolution.Units) {
			correct++
		}
	}
	if correct != pairs {
		t.Fatalf("irrelevant transformations changed interpretation: %d/%d", correct, pairs)
	}
	nSaveResult(t, "retention_stress_results.json", map[string]any{"heads": 20, "learned_n_units": len(s.Features), "retained_witnesses": len(s.Witnesses), "covered_queries": len(coverage), "unseen_transformation_pairs": pairs, "correct_and_identical_n_pairs": correct, "observations_are_synthetic": true})
}

func TestNResolutionRejectsUnsupportedCorrectionsAndFutureEvidence(t *testing.T) {
	r, _, _, _ := nNewRuntime(t)
	fit := nGrid([]float64{-.7, -.5, .5, .7}, []float64{-.8, .8})
	check := nGrid([]float64{-.9, -.3, .3, .9}, []float64{-.6, .6})
	truth := func(p nXY) float64 { return nSign(p.x) }
	nTeach(t, r, nLesson(0, 0, fit, check, truth))
	features := len(r.brain.BrainState.NResolution.Features)
	nodes := len(r.brain.Registry.Nodes())
	// A proposed changed response on new positive inputs would damage retained
	// observations. Current witness labels are not erased to make it pass.
	changed := func(p nXY) float64 { return -1 }
	f := nGrid([]float64{.41, .45, .49, .53}, []float64{-.8, .8})
	c := nGrid([]float64{.43, .47, .51, .55}, []float64{-.6, .6})
	d := nTeach(t, r, nLesson(0, 1, f, c, changed))
	if d.Committed {
		t.Fatalf("harmful correction committed: %+v", d)
	}
	reg := 0
	for _, v := range d.Trials {
		reg += v.RetentionRegressions
	}
	if reg == 0 {
		t.Fatal("retention rejection not exercised")
	}
	if len(r.brain.BrainState.NResolution.Features) != features || len(r.brain.Registry.Nodes()) != nodes {
		t.Fatal("rejected correction grew neurons")
	}
	conflict := nLesson(0, 2, fit, check, truth)
	conflict.Resolution.Fit[1].Input = conflict.Resolution.Fit[0].Input
	conflict.Resolution.Fit[1].Outcome = -conflict.Resolution.Fit[0].Outcome
	dc := nTeach(t, r, conflict)
	if dc.Cause != "conflicting-observations" || dc.Grown != 0 || len(r.brain.Registry.Nodes()) != nodes {
		t.Fatalf("conflicting observation invented knowledge: %+v", dc)
	}
	out := nAnswer(t, r, 0, nXY{.5, .8})
	if out.Accepted {
		t.Fatal("known conflicting evidence still yielded an accepted answer")
	}
	invalid := nLesson(1, 3, fit, check, truth)
	invalid.Resolution.Check[0].ObservedAt = invalid.After.ObservedAt.Add(time.Hour)
	r.SetClock(NewFixedClock(invalid.After.ObservedAt))
	before := continuationBytes(t, r)
	_, e := r.LearnComposition(invalid, invalid.After.ObservedAt)
	if e == nil || !reflect.DeepEqual(before, continuationBytes(t, r)) {
		t.Fatal("future evidence was accepted or mutated Brain")
	}
	invalid = nLesson(1, 3, fit, check, truth)
	invalid.Resolution.Check[0].EpisodeID = invalid.Resolution.Fit[0].EpisodeID
	_, e = r.LearnComposition(invalid, invalid.After.ObservedAt)
	if e == nil || !reflect.DeepEqual(before, continuationBytes(t, r)) {
		t.Fatal("non-independent checks were accepted or changed Brain")
	}
	nSaveResult(t, "rejection_results.json", map[string]any{"harmful_correction": d, "conflicting_evidence": dc, "retention_regression_trials": reg, "unchanged_neuron_count": nodes, "future_evidence_rejected_before_mutation": true, "shared_episode_check_rejected_before_mutation": true, "conflicting_readout_accepted": out.Accepted})
}

func TestNResolutionRejectsCorruptTopologyBeforeRestore(t *testing.T) {
	r, _, dir, _ := nNewRuntime(t)
	fit := nGrid([]float64{-.7, -.5, .5, .7}, []float64{-.8, .8})
	check := nGrid([]float64{-.9, -.3, .3, .9}, []float64{-.6, .6})
	nTeach(t, r, nLesson(0, 0, fit, check, func(p nXY) float64 { return nSign(p.x) }))
	path := filepath.Join(dir, "valid.json")
	compositionCheck(t, r.Checkpoint(path))
	bytes, e := os.ReadFile(path)
	compositionCheck(t, e)
	var original BrainSnapshot
	compositionCheck(t, json.Unmarshal(bytes, &original))
	for _, mutation := range []string{"zero_resolution", "duplicate_feature", "remove_readout"} {
		var raw map[string]any
		compositionCheck(t, json.Unmarshal(original.Brain, &raw))
		state := raw["n_resolution"].(map[string]any)
		features := state["features"].([]any)
		first := features[0].(map[string]any)["node"].(float64)
		switch mutation {
		case "duplicate_feature":
			state["features"] = append(features, features[0])
		case "zero_resolution":
			for _, x := range raw["synapses"].([]any) {
				v := x.(map[string]any)
				if v["target_id"] == first {
					v["dynamic"].(map[string]any)["weight"] = float64(0)
				}
			}
		case "remove_readout":
			kept := []any{}
			for _, x := range raw["synapses"].([]any) {
				v := x.(map[string]any)
				if v["source_id"] != first {
					kept = append(kept, x)
				}
			}
			raw["synapses"] = kept
		}
		snapshot := original
		snapshot.Brain, e = json.Marshal(raw)
		compositionCheck(t, e)
		snapshot.Checksum, e = checkpointChecksumForSchema(snapshot)
		compositionCheck(t, e)
		bad := filepath.Join(dir, mutation+".json")
		b, e := json.Marshal(snapshot)
		compositionCheck(t, e)
		compositionCheck(t, os.WriteFile(bad, b, 0600))
		target := NewBrainRuntime(knowledge.NewBrain())
		before := continuationBytes(t, target)
		if target.RestoreCheckpoint(bad) == nil {
			t.Fatalf("accepted %s", mutation)
		}
		if !reflect.DeepEqual(before, continuationBytes(t, target)) {
			t.Fatalf("%s mutated restore target", mutation)
		}
	}
	nSaveResult(t, "restore_validation_results.json", map[string]any{"recomputed_checksum_does_not_bypass_topology_checks": true, "rejected_mutations": []string{"zero_resolution", "duplicate_feature", "remove_readout"}, "all_restore_targets_unchanged": true})
}
