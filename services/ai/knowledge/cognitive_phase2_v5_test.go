package knowledge

import (
	"math"
	"reflect"
	"testing"
	"time"
)

func phase2FixtureBatch(kb *KnowledgeBase) CognitiveTrainingBatch {
	c := kb.BrainState.CognitiveModel.Config
	y := 1
	x := metaTestInput()
	x.NeuralMeta.Mode = "recurrent"
	next := metaNextFeatures(c, x.Features, 0, 1, false)
	eps := []CognitiveTrainingEpisode{{VerifiedTarget: &y, VerifiedOperations: []VerifiedMentalResult{{Operation: 0, Outcome: 1}}, SuccessAt: 1, Frames: []CognitiveTrainingFrame{{Features: x.Features, Mask: []bool{true, false, false, false}, NeuralMeta: x.NeuralMeta, Action: 0, Reward: -.1, Draw: .3}, {Features: next, Mask: []bool{false, true, true, true}, NeuralMeta: x.NeuralMeta, Action: 2, Reward: 1, Draw: .3}}}}
	w, _ := kb.CognitiveParameters()
	return CognitiveTrainingBatch{Episodes: eps, ParametersHash: CognitiveDigest(w), Evidence: CognitiveEvidence{ID: "phase2-fixture", Source: "native-unit", Verifier: "executed-operation-and-terminal", Digest: CognitiveDigest(eps)}}
}
func TestPhase2NativeGradientsAndReadoutProtection(t *testing.T) {
	kb := NewBrain()
	at := time.Unix(1900000000, 0)
	if e := kb.InitializeCognitiveModel(metaTestConfig(), at); e != nil {
		t.Fatal(e)
	}
	spec := Phase2SpecV5{Version: Phase2VersionV5, Representation: "pairwise", LearnGRU: true, Optimizer: "historical-global"}
	if e := kb.InitializePhase2V5(spec, at); e != nil {
		t.Fatal(e)
	}
	b := phase2FixtureBatch(kb)
	if _, e := kb.LearnPhase2V5(Phase2BatchV5{Batch: b}, at.Add(time.Second)); e != nil {
		t.Fatal(e)
	}
	b = phase2FixtureBatch(kb)
	s := kb.BrainState.CognitiveModel
	w, _ := kb.CognitiveParameters()
	_, g, e := phase2ObjectiveV5(s, w, b)
	if e != nil {
		t.Fatal(e)
	}
	maximum, count, gruMax := 0., 0, 0.
	for i := range w {
		if !phase2MutableV5(s, i) {
			continue
		}
		old := w[i]
		w[i] = old + 1e-5
		a, _, e := phase2ObjectiveV5(s, w, b)
		if e != nil {
			t.Fatal(e)
		}
		w[i] = old - 1e-5
		z, _, e := phase2ObjectiveV5(s, w, b)
		if e != nil {
			t.Fatal(e)
		}
		w[i] = old
		d := math.Abs((a.Loss-z.Loss)/2e-5 - g[i])
		maximum = math.Max(maximum, d)
		count++
		if i < s.Phase2V5.BaseParameters {
			gruMax = math.Max(gruMax, math.Abs(g[i]))
		}
		if d > 3e-7 {
			t.Fatal("gradient", i, d)
		}
	}
	if gruMax < 1e-9 {
		t.Fatal("BPTT not exercised")
	}
	for i := range s.FirstMoment {
		if !phase2MutableV5(s, i) {
			s.FirstMoment[i] = .13
			s.SecondMoment[i] = .17
		}
	}
	before, _ := kb.CognitiveParameters()
	fm, sm := append([]float64(nil), s.FirstMoment...), append([]float64(nil), s.SecondMoment...)
	b = phase2FixtureBatch(kb)
	if _, e := kb.LearnPhase2V5(Phase2BatchV5{Batch: b}, at.Add(2*time.Second)); e != nil {
		t.Fatal(e)
	}
	after, _ := kb.CognitiveParameters()
	protected := 0
	for i := range before {
		if !phase2MutableV5(s, i) {
			if before[i] != after[i] || fm[i] != s.FirstMoment[i] || sm[i] != s.SecondMoment[i] {
				t.Fatal("protected parameter/moment changed", i)
			}
			protected++
		}
	}
	data, _ := kb.CanonicalJSON()
	if e := ValidateCognitiveModelState(data, true); e != nil {
		t.Fatal(e)
	}
	t.Logf("native phase2 checked %d gradients; maxerr=%g nonzeroGRU=%g protected=%d", count, maximum, gruMax, protected)
}
func TestPhase2DefaultSelectorUnchangedAndCompiledExact(t *testing.T) {
	kb := NewBrain()
	at := time.Unix(1900000000, 0)
	if e := kb.InitializeCognitiveModel(metaTestConfig(), at); e != nil {
		t.Fatal(e)
	}
	before, e := kb.CognitivePreview(metaTestInput())
	if e != nil {
		t.Fatal(e)
	}
	spec := Phase2SpecV5{Version: Phase2VersionV5, Representation: "pairwise", Optimizer: "historical-global"}
	if e := kb.InitializePhase2V5(spec, at); e != nil {
		t.Fatal(e)
	}
	after, e := kb.CognitivePreview(metaTestInput())
	if e != nil {
		t.Fatal(e)
	}
	before.ParametersHash, after.ParametersHash = "", ""
	if !reflect.DeepEqual(before, after) {
		t.Fatal("default production selector changed")
	}
	s := kb.BrainState.CognitiveModel
	w, _ := kb.CognitiveParameters()
	h := []float64{.2, -.1}
	w[s.Phase2V5.Weights.Offset] = .4
	for iter := 0; iter < 3; iter++ {
		a, e := phase2LogitsV5(s, w, h, true)
		if e != nil {
			t.Fatal(e)
		}
		b, e := phase2LogitsV5(s, w, h, false)
		if e != nil {
			t.Fatal(e)
		}
		for i := range a {
			if math.Abs(a[i]-b[i]) > 1e-14 {
				t.Fatal("compiled mismatch")
			}
		}
		w[s.Phase2V5.Weights.Offset] += .3
		h[0] += .1
	}
	data, _ := kb.CanonicalJSON()
	x := metaTestInput()
	x.NeuralMeta.Mode = "muse-bmps"
	f := metaNextFeatures(s.Config, x.Features, 0, 1, false)
	x2 := x
	x2.Reset = false
	x2.Features = f
	x2.Mask[0] = false
	p, e := kb.PreviewPhase2V5(Phase2ProbeV5{Trace: []CognitiveInput{x, x2}, ConditionObserved: true})
	if e != nil {
		t.Fatal(e)
	}
	mass := 0.
	for state, q := range p.Readout.JointBelief {
		mass += q
		if metaOutcome(*s.Config.NeuralMeta, state, 0) != 1 && q != 0 {
			t.Fatal("conditioning contradicts observed outcome")
		}
	}
	if math.Abs(mass-1) > 1e-12 {
		t.Fatal("normalization")
	}
	again, _ := kb.CanonicalJSON()
	if !reflect.DeepEqual(data, again) {
		t.Fatal("probe mutated canonical state")
	}
	t.Log("default selector preserved; dense/compiled agree after weight and h changes; valid zero-support posterior; read-only")
}
func TestPhase2ExactClockFreezeAndSelection(t *testing.T) {
	kb := NewBrain()
	at := time.Unix(1900000000, 0)
	if e := kb.InitializeCognitiveModel(metaTestConfig(), at); e != nil {
		t.Fatal(e)
	}
	kb.BrainState.CognitiveModel.OptimizerStep = 2900
	spec := Phase2SpecV5{Version: Phase2VersionV5, Representation: "pairwise", Optimizer: "new-head-effective"}
	if e := kb.InitializePhase2V5(spec, at); e != nil {
		t.Fatal(e)
	}
	s := kb.BrainState.CognitiveModel
	before, _ := kb.CognitiveParameters()
	for i := 0; i < 3; i++ {
		b := phase2FixtureBatch(kb)
		if _, e := kb.LearnPhase2V5(Phase2BatchV5{Batch: b, FreezeHead: true}, at.Add(time.Duration(i+1)*time.Second)); e != nil {
			t.Fatal(e)
		}
	}
	now, _ := kb.CognitiveParameters()
	if !reflect.DeepEqual(before, now) || s.Phase2V5.HeadSteps != 0 || s.OptimizerStep != 2903 {
		t.Fatal("frozen clock or parameters moved")
	}
	b := phase2FixtureBatch(kb)
	_, g, e := phase2ObjectiveV5(s, now, b)
	if e != nil {
		t.Fatal(e)
	}
	ss := 0.
	for i, v := range g {
		if phase2MutableV5(s, i) {
			ss += v * v
		}
	}
	scale := 1.
	if ss > 1 {
		scale = 1 / math.Sqrt(ss)
	}
	if _, e := kb.LearnPhase2V5(Phase2BatchV5{Batch: b}, at.Add(4*time.Second)); e != nil {
		t.Fatal(e)
	}
	next, _ := kb.CognitiveParameters()
	for i, d := range g {
		if !phase2MutableV5(s, i) {
			continue
		}
		d *= scale
		expected := s.Config.LearningRate * d / (math.Abs(d) + 1e-8)
		if math.Abs((now[i]-next[i])-expected) > 1e-12 {
			t.Fatal("effective Adam bias correction", i)
		}
	}
	z := []float64{1000, -1000, 999}
	event, support := []bool{false, false, true}, []bool{true, false, true}
	_, gradient, e := phase2PartialV5(z, event, support)
	if e != nil || gradient[1] != 0 {
		t.Fatal("selection likelihood")
	}
	if s.Phase2V5.HeadSteps != 1 || s.OptimizerStep != 2904 {
		t.Fatal("unfreeze counters")
	}
	t.Log("exact new-head counts0->1 while historical clock2900->2904; freeze preserves moments; conditional support has zero excluded gradient")
}

func TestPhase2CategoricalHistoricalObjectiveAndAdamExact(t *testing.T) {
	a, b := NewBrain(), NewBrain()
	at := time.Unix(1900000000, 0)
	for _, kb := range []*KnowledgeBase{a, b} {
		if e := kb.InitializeCognitiveModel(metaTestConfig(), at); e != nil {
			t.Fatal(e)
		}
		s := kb.BrainState.CognitiveModel
		s.OptimizerStep = 2900
		for i := range s.FirstMoment {
			s.FirstMoment[i] = .11
			s.SecondMoment[i] = .17
		}
	}
	spec := Phase2SpecV5{Version: Phase2VersionV5, Representation: "categorical", Optimizer: "historical-global"}
	if e := b.InitializePhase2V5(spec, at); e != nil {
		t.Fatal(e)
	}
	ba, bb := phase2FixtureBatch(a), phase2FixtureBatch(b)
	ba.ComponentV4 = "world-decoder"
	wa, _ := a.CognitiveParameters()
	wb, _ := b.CognitiveParameters()
	old, og := cognitiveComponentObjectiveV4(a.BrainState.CognitiveModel, wa, ba, cognitiveTargets{})
	now, ng, e := phase2ObjectiveV5(b.BrainState.CognitiveModel, wb, bb)
	if e != nil {
		t.Fatal(e)
	}
	if math.Abs(old.Loss-now.Loss) > 1e-14 || !reflect.DeepEqual(og, ng) {
		t.Fatal("unconditional categorical objective changed")
	}
	if _, e := a.LearnCognitiveBatch(ba, at); e != nil {
		t.Fatal(e)
	}
	if _, e := b.LearnPhase2V5(Phase2BatchV5{Batch: bb}, at); e != nil {
		t.Fatal(e)
	}
	wa, _ = a.CognitiveParameters()
	wb, _ = b.CognitiveParameters()
	sa, sb := a.BrainState.CognitiveModel, b.BrainState.CognitiveModel
	if !reflect.DeepEqual(wa, wb) || !reflect.DeepEqual(sa.FirstMoment, sb.FirstMoment) || !reflect.DeepEqual(sa.SecondMoment, sb.SecondMoment) || sa.OptimizerStep != sb.OptimizerStep {
		t.Fatal("historical categorical Adam trajectory changed")
	}
	t.Log("B_plain matches V4 categorical objective, every gradient, actual synaptic update and both inherited Adam moments exactly")
}
