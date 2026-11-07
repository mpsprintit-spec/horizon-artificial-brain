package knowledge

import (
	"encoding/json"
	"math"
	"testing"
	"time"
)

func TestFidelityV5FullFeatureGradientNumerical(t *testing.T) {
	b := coordinatedFixtureV5(t)
	batch := fidelityTestBatchV5(t, b)
	batch.Fidelity.FullFeatureGradient = true
	s := b.BrainState.CognitiveModel
	w, _ := b.CognitiveParameters()
	temps := b.metaTemperatures(s)
	policy := Phase4BatchV5{Episodes: []Phase4EpisodeV5{{Frames: batch.Episodes[0].Frames}}}
	a, r := phase4Targets(s.Phase4V5, w, policy)
	_, g, e := fidelityPolicyObjectiveFullV5(s, w, temps, batch, a, r)
	if e != nil {
		t.Fatal(e)
	}
	maximum, count := 0., 0
	// Independently recompute only the scalar loss for finite differences. This
	// avoids evaluating thousands of unused backward passes under the race test.
	loss := func(weights []float64) float64 {
		total := 0.
		for ei, ep := range batch.Episodes {
			for j, tr := range ep.Traces {
				pred, err := fidelityForwardV5(s, weights, temps, tr, Phase4DropV5{})
				if err != nil {
					t.Fatal(err)
				}
				delta, entropy := pred.Value-r[ei][j], 0.
				for _, q := range pred.Policy {
					if q > 0 {
						entropy += q * math.Log(q)
					}
				}
				total += -a[ei][j]*math.Log(pred.Policy[ep.Frames[j].Action]) + .5*s.Phase4V5.Spec.ValueWeight*delta*delta + s.Phase4V5.Spec.EntropyWeight*entropy
			}
		}
		return total / float64(len(batch.Episodes))
	}
	for i := range w {
		if !(i >= s.Phase4V5.BaseParameters || fidelityRecurrentParameterV5(s, i)) {
			if g[i] != 0 {
				t.Fatal("ownership", i)
			}
			continue
		}
		old := w[i]
		w[i] = old + 1e-5
		plus := loss(w)
		w[i] = old - 1e-5
		minus := loss(w)
		w[i] = old
		err := math.Abs((plus-minus)/2e-5 - g[i])
		maximum = math.Max(maximum, err)
		count++
		if err > 3e-7 {
			t.Fatalf("full-policy coordinate %d error %.12g analytic %.12g", i, err, g[i])
		}
	}
	t.Logf("full executing policy %d coordinates max error %.12g", count, maximum)
}

func TestFidelityV5FullFeatureGradientEvidenceAndOwnership(t *testing.T) {
	b := coordinatedFixtureV5(t)
	batch := fidelityTestBatchV5(t, b)
	// Old serialized options and their digest remain byte-for-byte compatible.
	raw, _ := json.Marshal(batch.Fidelity)
	if string(raw) != "{\"AssessmentOnly\":false,\"FreezeRecurrence\":false}" {
		t.Fatal(string(raw))
	}
	batch.Fidelity.FullFeatureGradient = true
	if b.ValidateCoordinatedBatchV5(batch) == nil {
		t.Fatal("new learning option without new receipt accepted")
	}
	batch.Evidence.Digest = CoordinatedEvidenceDigestV5(batch)
	if e := b.ValidateCoordinatedBatchV5(batch); e != nil {
		t.Fatal(e)
	}
	s := b.BrainState.CognitiveModel
	w, _ := b.CognitiveParameters()
	m1, m2 := append([]float64(nil), s.FirstMoment...), append([]float64(nil), s.SecondMoment...)
	if _, e := b.LearnCoordinatedV5(batch, time.Unix(2400000000, 0)); e != nil {
		t.Fatal(e)
	}
	after, _ := b.CognitiveParameters()
	changed := 0
	for i := range w {
		if !fidelityMUSEParameterV5(s, i) && !fidelityRecurrentParameterV5(s, i) && i < s.Phase4V5.BaseParameters {
			if w[i] != after[i] || m1[i] != s.FirstMoment[i] || m2[i] != s.SecondMoment[i] {
				t.Fatal("protected N/moments changed", i)
			}
		}
		if fidelityRecurrentParameterV5(s, i) && w[i] != after[i] {
			changed++
		}
	}
	if changed == 0 {
		t.Fatal("no recurrent update")
	}
	for _, opts := range []FidelityLearningV5{{AssessmentOnly: true, FullFeatureGradient: true}, {FreezeRecurrence: true, FullFeatureGradient: true}} {
		batch.Fidelity = &opts
		if validateFidelityContractV5(s, batch) == nil {
			t.Fatal("ambiguous ownership accepted")
		}
	}
}

func TestFidelityV5FullGradientRareAndMaskedActions(t *testing.T) {
	b := coordinatedFixtureV5(t)
	batch := fidelityTestBatchV5(t, b)
	batch.Fidelity.FullFeatureGradient = true
	s := b.BrainState.CognitiveModel
	w, _ := b.CognitiveParameters()
	temps := b.metaTemperatures(s)
	w[s.Phase4V5.ActorBias.Offset+3] = -80
	a, r := [][]float64{{.4, .8}}, [][]float64{{.9, 1}}
	st, g, e := fidelityPolicyObjectiveFullV5(s, w, temps, batch, a, r)
	if e != nil || !finiteNumber(st.Loss) {
		t.Fatal("rare loss", e)
	}
	for i, x := range g {
		if !finiteNumber(x) {
			t.Fatal("rare derivative", i)
		}
	}
	i := s.Phase4V5.ActorBias.Offset + 3
	v := w[i]
	w[i] = v + 1e-4
	plus, _, _ := fidelityPolicyObjectiveFullV5(s, w, temps, batch, a, r)
	w[i] = v - 1e-4
	minus, _, _ := fidelityPolicyObjectiveFullV5(s, w, temps, batch, a, r)
	if math.Abs((plus.Loss-minus.Loss)/2e-4-g[i]) > 1e-7 {
		t.Fatal("rare action lost derivative")
	}
	// The second frame legally masks the already executed operation.
	p, e := fidelityForwardV5(s, w, temps, batch.Episodes[0].Traces[1], Phase4DropV5{})
	if e != nil || p.Policy[0] != 0 {
		t.Fatal("illegal action probability", e)
	}
}
