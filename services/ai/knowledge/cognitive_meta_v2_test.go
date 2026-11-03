package knowledge

import (
	"math"
	"reflect"
	"testing"
	"time"
)

func metaTestConfig() CognitiveModelConfig {
	c := cognitiveTestConfig()
	c.Version = CognitiveModelV2
	c.Inputs = 12
	c.Hidden = 2
	c.Actions = 4
	c.Horizon = 3
	c.WorldWeight = 0
	c.NeuralMeta = &NeuralMetaConfig{Classes: 2, Computations: 1, Outcomes: 2, ContextOffset: 0, ContextSize: 1, ResultOffset: 1, OperationOffset: 3, ActionOffset: 4, HypothesisOffset: 8, PlanMarker: 10, StepIndex: 11, JointWeight: .7, RewardWeight: .3, PlanDepth: 1}
	return c
}
func metaTestInput() CognitiveInput {
	return CognitiveInput{Features: []float64{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, Mask: []bool{true, true, true, true}, Reset: true, Draw: .3, NeuralMeta: &NeuralMetaInput{Mode: "bmps", Costs: []float64{.1}, Remaining: 2}}
}
func TestNativeMetaV2JointPlanBPTT(t *testing.T) {
	k := NewBrain()
	c := metaTestConfig()
	if e := k.InitializeCognitiveModel(c, time.Unix(1800000000, 0)); e != nil {
		t.Fatal(e)
	}
	s := k.BrainState.CognitiveModel
	w, _ := k.CognitiveParameters()
	y := 1
	x := metaTestInput()
	x.NeuralMeta.Mode = "recurrent"
	next := metaNextFeatures(c, x.Features, 0, 1, false)
	batch := CognitiveTrainingBatch{LearnPolicy: true, Episodes: []CognitiveTrainingEpisode{{VerifiedTarget: &y, VerifiedOperations: []VerifiedMentalResult{{0, 1}}, SuccessAt: 1, Frames: []CognitiveTrainingFrame{{Features: x.Features, Mask: []bool{true, false, false, false}, NeuralMeta: x.NeuralMeta, Action: 0, Reward: -.1}, {Features: next, Mask: []bool{false, true, true, true}, NeuralMeta: x.NeuralMeta, Action: 2, Reward: 1}}}}}
	target, e := makeCognitiveTargets(c, s.Layout, w, batch, false)
	if e != nil {
		t.Fatal(e)
	}
	_, g := cognitiveBatchObjectiveV2(s, w, batch, target)
	epsilon := 1e-5
	maximum := 0.0
	for i := range w {
		old := w[i]
		w[i] = old + epsilon
		a, _ := cognitiveBatchObjectiveV2(s, w, batch, target)
		w[i] = old - epsilon
		b, _ := cognitiveBatchObjectiveV2(s, w, batch, target)
		w[i] = old
		numeric := (a.Loss - b.Loss) / (2 * epsilon)
		delta := math.Abs(numeric - g[i])
		maximum = math.Max(maximum, delta)
		if delta > 3e-7 {
			t.Fatalf("V2 gradient %d analytic=%g numerical=%g", i, g[i], numeric)
		}
	}
	t.Logf("V2 verified %d gradients max absolute error %.12g", len(w), maximum)
}
func TestNativeMetaV2NoOracleReadonlyAndConditionals(t *testing.T) {
	k := NewBrain()
	if e := k.InitializeCognitiveModel(metaTestConfig(), time.Unix(1800000000, 0)); e != nil {
		t.Fatal(e)
	}
	x := metaTestInput()
	before, _ := k.CanonicalJSON()
	a, e := k.CognitivePreview(x)
	if e != nil {
		t.Fatal(e)
	}
	after, _ := k.CanonicalJSON()
	if !reflect.DeepEqual(before, after) {
		t.Fatal("hash memoization changed canonical state")
	}
	if len(a.ClassBelief) != 2 || len(a.NativeLikelihood) != 1 {
		t.Fatal("missing native beliefs/likelihood")
	}
	for _, row := range a.NativeLikelihood[0].Likelihood {
		if math.Abs(row[0]+row[1]-1) > 1e-12 {
			t.Fatal("conditional likelihood not normalized")
		}
	}
	fake := bmpsTestBelief()
	x.Belief = &fake
	if _, e = k.CognitivePreview(x); e == nil {
		t.Fatal("native mode accepted adapter belief")
	}
	x.Belief = nil
	x.Features[8] = 1
	if _, e = k.CognitivePreview(x); e == nil {
		t.Fatal("observed input accepted target hypothesis")
	}
	if e = ValidateCognitiveModelState(before, true); e != nil {
		t.Fatal(e)
	}
	if e = ValidateMemoryDecayState(before, true); e != nil {
		t.Fatal(e)
	}
}
func TestNativeMetaV2FeedbackChangesLearnedBelief(t *testing.T) {
	k := NewBrain()
	c := metaTestConfig()
	c.Hidden = 6
	if e := k.InitializeCognitiveModel(c, time.Unix(1800000000, 0)); e != nil {
		t.Fatal(e)
	}
	x := metaTestInput()
	x.NeuralMeta.Mode = "recurrent"
	x.Mask = []bool{false, false, true, false}
	x.Draw = .1
	before, _ := k.CognitivePreview(x)
	for i := 0; i < 80; i++ {
		p, e := k.CognitivePreview(x)
		if e != nil {
			t.Fatal(e)
		}
		y := 1
		eps := []CognitiveTrainingEpisode{{VerifiedTarget: &y, SuccessAt: 0, Frames: []CognitiveTrainingFrame{{Features: x.Features, Mask: x.Mask, NeuralMeta: x.NeuralMeta, Action: 2, Draw: x.Draw, Behaviour: p.Policy, Reward: 1}}}}
		b := CognitiveTrainingBatch{Episodes: eps, LearnPolicy: true, ParametersHash: p.ParametersHash, Evidence: CognitiveEvidence{ID: "verified-target", Source: "controlled-native", Verifier: "actual-success", Digest: CognitiveDigest(eps)}}
		if _, e = k.LearnCognitiveBatch(b, time.Unix(1800000001+int64(i), 0)); e != nil {
			t.Fatal(e)
		}
	}
	after, _ := k.CognitivePreview(x)
	if after.ClassBelief[1] <= before.ClassBelief[1] {
		t.Fatal("verified experience did not change neural belief")
	}
	raw, _ := k.CanonicalJSON()
	if e := ValidateCognitiveModelState(raw, true); e != nil {
		t.Fatal(e)
	}
}
func TestNativeMetaV2RecurrenceAndFeedbackAblations(t *testing.T) {
	k := NewBrain()
	if e := k.InitializeCognitiveModel(metaTestConfig(), time.Unix(1800000000, 0)); e != nil {
		t.Fatal(e)
	}
	x := metaTestInput()
	if _, e := k.AdvanceCognitiveModel(x, time.Unix(1800000000, 0)); e != nil {
		t.Fatal(e)
	}
	x.Reset = false
	x.Features = metaNextFeatures(metaTestConfig(), x.Features, 0, 1, false)
	a, e := k.CognitivePreview(x)
	if e != nil {
		t.Fatal(e)
	}
	x.NeuralMeta.NoFeedback = true
	b, e := k.CognitivePreview(x)
	if e != nil {
		t.Fatal(e)
	}
	if reflect.DeepEqual(a.ClassBelief, b.ClassBelief) {
		t.Fatal("neural belief ignores returned result")
	}
	x.NeuralMeta.NoFeedback = false
	x.NoRecurrence = true
	c, e := k.CognitivePreview(x)
	if e != nil {
		t.Fatal(e)
	}
	if reflect.DeepEqual(a.ClassBelief, c.ClassBelief) {
		t.Fatal("neural belief ignores recurrent state")
	}
}
