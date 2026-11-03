package knowledge

import (
	"math"
	"reflect"
	"testing"
	"time"
)

func TestV4PartialNLLStableAndMarginalGradient(t *testing.T) {
	z := []float64{1000, -1000, 999}
	a := []bool{false, true, false}
	loss, g := partialCategoricalV4(z, a)
	if !finiteNumber(loss) || math.Abs(loss-(2000+math.Log1p(math.Exp(-1)))) > 1e-10 || g[1] != -1 {
		t.Fatal(loss, g)
	}
	z = []float64{.2, -.3, .7}
	a = []bool{true, false, true}
	_, g = partialCategoricalV4(z, a)
	for i := range z {
		old := z[i]
		z[i] = old + 1e-5
		lp, _ := partialCategoricalV4(z, a)
		z[i] = old - 1e-5
		lm, _ := partialCategoricalV4(z, a)
		z[i] = old
		if math.Abs(g[i]-(lp-lm)/2e-5) > 1e-8 {
			t.Fatal(i)
		}
	}
	t.Log("finite NLL for a supervised event whose softmax probability underflows to zero")
}
func TestV4ComponentGradientMasksAndReadoutRetention(t *testing.T) {
	for _, component := range []string{"world-decoder", "actor-critic", "actor-only"} {
		kb := NewBrain()
		if e := kb.InitializeCognitiveModel(metaTestConfig(), time.Unix(1800000000, 0)); e != nil {
			t.Fatal(e)
		}
		s := kb.BrainState.CognitiveModel
		w, _ := kb.CognitiveParameters()
		b := terminalTestBatchV3(kb)
		b.TerminalReadoutOnly = false
		b.ComponentV4 = component
		b.LearnPolicy = component != "world-decoder"
		if b.LearnPolicy {
			b.Episodes = b.Episodes[:1]
			f := &b.Episodes[0].Frames[0]
			f.NeuralMeta.Mode = "recurrent"
			f.Mask = []bool{false, true, true, true}
			h := cognitiveGRU(s.Config, s.Layout, w, f.Features, make([]float64, s.Config.Hidden)).h
			p, _, _ := cognitiveHeads(s.Layout, w, h, f.Mask)
			f.Draw = .42
			f.Behaviour = p
			f.Action = samplePolicy(p, f.Draw)
			f.Reward = 0
			b.Episodes[0].SuccessAt = -1
			if f.Action == 2 {
				f.Reward = 1
				b.Episodes[0].SuccessAt = 0
			}
			if f.Action == 3 {
				f.Reward = .35
			}
			v := .12
			b.EntropyWeightV4 = &v
		}
		b.Evidence.Digest = CognitiveDigest(b.Episodes)
		target, e := makeCognitiveTargets(s.Config, s.Layout, w, b, true)
		if e != nil {
			t.Fatal(e)
		}
		base, g := cognitiveComponentObjectiveV4(s, w, b, target)
		if !finiteNumber(base.Loss) {
			t.Fatal(base)
		}
		maxerr, count := 0., 0
		for i := range w {
			if !componentParameterV4(s, component, i) {
				if g[i] != 0 {
					t.Fatal("protected gradient")
				}
				continue
			}
			old := w[i]
			w[i] = old + 1e-5
			lp, _ := cognitiveComponentObjectiveV4(s, w, b, target)
			w[i] = old - 1e-5
			lm, _ := cognitiveComponentObjectiveV4(s, w, b, target)
			w[i] = old
			d := math.Abs((lp.Loss-lm.Loss)/2e-5 - g[i])
			maxerr = math.Max(maxerr, d)
			count++
			if d > 3e-7 {
				t.Fatal(component, i, d)
			}
		}
		// Nonzero inherited moments MUST NOT move protected parameters.
		for i := range s.FirstMoment {
			s.FirstMoment[i] = .11
			s.SecondMoment[i] = .17
		}
		fm := append([]float64{}, s.FirstMoment...)
		sm := append([]float64{}, s.SecondMoment...)
		before, _ := kb.CognitivePreview(metaTestInput())
		if _, e = kb.LearnCognitiveBatch(b, time.Unix(1800000001, 0)); e != nil {
			t.Fatal(e)
		}
		after, _ := kb.CognitivePreview(metaTestInput())
		actual, _ := kb.CognitiveParameters()
		for i := range actual {
			if !componentParameterV4(s, component, i) && (actual[i] != w[i] || s.FirstMoment[i] != fm[i] || s.SecondMoment[i] != sm[i]) {
				t.Fatal("protected parameter/moment changed", i)
			}
		}
		if !reflect.DeepEqual(before.Hidden, after.Hidden) {
			t.Fatal("GRU observation mapping changed")
		}
		if component == "world-decoder" && !reflect.DeepEqual(before.Policy, after.Policy) {
			t.Fatal("actor changed")
		}
		t.Logf("%s: %d derivatives; max error %.12g; protected weights/moments exact", component, count, maxerr)
	}
}
func TestV4RejectInvalidComponentContracts(t *testing.T) {
	kb := NewBrain()
	kb.InitializeCognitiveModel(metaTestConfig(), time.Unix(1800000000, 0))
	b := terminalTestBatchV3(kb)
	b.ComponentV4 = "world-decoder"
	if kb.ValidateCognitiveBatch(b) == nil {
		t.Fatal("mixed V3/V4 accepted")
	}
	b.TerminalReadoutOnly = false
	b.LearnPolicy = true
	if kb.ValidateCognitiveBatch(b) == nil {
		t.Fatal("world policy accepted")
	}
	b.LearnPolicy = false
	b.ComponentV4 = "unknown"
	if kb.ValidateCognitiveBatch(b) == nil {
		t.Fatal("unknown mask")
	}
}
