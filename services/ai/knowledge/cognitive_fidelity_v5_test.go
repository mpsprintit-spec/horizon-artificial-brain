package knowledge

import (
	"math"
	"reflect"
	"testing"
	"time"
)

func fidelityTestBatchV5(t *testing.T, b *Brain) CoordinatedBatchV5 {
	t.Helper()
	s := b.BrainState.CognitiveModel
	trace := coordinatedTraceV5()
	trace[0].Mask = []bool{true, true, true, true, true, true, true, true}
	trace[0].NeuralMeta.Mode = CoordinatedFidelityV5
	ep := CoordinatedEpisodeV5{}
	for step, action := range []int{0, 3} {
		last := len(trace) - 1
		p, err := b.PreviewCoordinatedV5(trace, Phase4DropV5{})
		if err != nil {
			t.Fatal(err)
		}
		draw := 0.
		for a := 0; a < action; a++ {
			draw += p.Policy[a]
		}
		draw += p.Policy[action] / 2
		trace[last].Draw = draw
		p, err = b.PreviewCoordinatedV5(trace, Phase4DropV5{})
		if err != nil || p.Action != action {
			t.Fatal("draw selection", err)
		}
		reward := 1.
		if step == 0 {
			reward = -trace[last].NeuralMeta.Costs[0]
		}
		ep.Traces = append(ep.Traces, append([]CognitiveInput(nil), trace...))
		ep.Frames = append(ep.Frames, Phase4FrameV5{Features: p.Features, Mask: append([]bool(nil), trace[last].Mask...), Action: p.Action, Reward: reward, Draw: draw, BehaviorProbability: p.Policy[action], BehaviorValue: p.Value})
		if step == 0 {
			next := trace[last]
			next.Features = metaNextFeatures(s.Config, next.Features, 0, 1, false)
			next.Mask = append([]bool(nil), next.Mask...)
			next.Mask[0] = false
			next.Reset = false
			meta := *next.NeuralMeta
			meta.Remaining--
			next.NeuralMeta = &meta
			trace = append(trace, next)
		}
	}
	w, _ := b.CognitiveParameters()
	out := CoordinatedBatchV5{Fidelity: &FidelityLearningV5{}, Episodes: []CoordinatedEpisodeV5{ep}, ParametersHash: CognitiveDigest(w)}
	out.Evidence = CognitiveEvidence{ID: "fidelity-unit", Source: "executed-toy-trace", Verifier: "binary-selected-outcome", Digest: CoordinatedEvidenceDigestV5(out)}
	return out
}

func TestFidelityV5ScientificFeaturesAndLegality(t *testing.T) {
	b := coordinatedFixtureV5(t)
	batch := fidelityTestBatchV5(t, b)
	for _, trace := range batch.Episodes[0].Traces {
		p, err := b.PreviewCoordinatedV5(trace, Phase4DropV5{})
		if err != nil {
			t.Fatal(err)
		}
		for a := 0; a < 4; a++ {
			if p.Competence[a] != p.TerminalUtility[a] || p.TerminalUtility[a] < 0 || p.TerminalUtility[a] > 1 {
				t.Fatal("binary competence/payoff contract")
			}
		}
		if p.TerminalUtility[4] != .35 || p.BMPS == nil {
			t.Fatal("missing BMPS or abstain")
		}
		for _, f := range p.BMPS.Features {
			if f.Myopic > f.Subset+1e-12 || f.Subset > f.Perfect+1e-12 {
				t.Fatal("BMPS information bounds")
			}
		}
		for a, q := range p.Policy {
			if !trace[len(trace)-1].Mask[a] && q != 0 {
				t.Fatal("illegal probability")
			}
		}
	}
	last := batch.Episodes[0].Traces[1]
	broken := append([]CognitiveInput(nil), last...)
	broken[1].Features = append([]float64(nil), broken[1].Features...)
	broken[1].Features[19] = 1
	if _, err := b.PreviewCoordinatedV5(broken, Phase4DropV5{}); err == nil {
		t.Fatal("oracle hint accepted")
	}
	if err := b.ValidateCoordinatedBatchV5(batch); err != nil {
		t.Fatal(err)
	}
	bad := batch
	copyOptions := *batch.Fidelity
	copyOptions.FreezeRecurrence = true
	bad.Fidelity = &copyOptions
	if b.ValidateCoordinatedBatchV5(bad) == nil {
		t.Fatal("tampered objective options accepted")
	}
	bad = batch
	bad.Fidelity = nil
	if b.ValidateCoordinatedBatchV5(bad) == nil {
		t.Fatal("missing explicit learning options")
	}
}

func TestFidelityV5BPTTAndAssessmentNumericalGradients(t *testing.T) {
	b := coordinatedFixtureV5(t)
	batch := fidelityTestBatchV5(t, b)
	s := b.BrainState.CognitiveModel
	w, _ := b.CognitiveParameters()
	policy := Phase4BatchV5{Episodes: []Phase4EpisodeV5{{Frames: batch.Episodes[0].Frames}}}
	a, r := phase4Targets(s.Phase4V5, w, policy)
	_, g := fidelityPolicyObjectiveV5(s, w, batch, a, r)
	maximum, count := 0., 0
	for i := range w {
		if !(i >= s.Phase4V5.BaseParameters || fidelityRecurrentParameterV5(s, i)) {
			if g[i] != 0 {
				t.Fatal("policy ownership")
			}
			continue
		}
		old := w[i]
		w[i] = old + 1e-5
		plus, _ := fidelityPolicyObjectiveV5(s, w, batch, a, r)
		w[i] = old - 1e-5
		minus, _ := fidelityPolicyObjectiveV5(s, w, batch, a, r)
		w[i] = old
		err := math.Abs((plus.Loss-minus.Loss)/2e-5 - g[i])
		maximum = math.Max(maximum, err)
		count++
		if err > 2e-7 {
			t.Fatalf("BPTT coordinate%d err%g", i, err)
		}
	}
	t.Logf("full head plus recurrent numerical coordinates %d maxerr %.12g", count, maximum)
	temps := b.metaTemperatures(s)
	_, mg, err := fidelityMUSEObjectiveV5(s, w, temps, batch)
	if err != nil {
		t.Fatal(err)
	}
	maximum, count = 0, 0
	for i := range w {
		if !fidelityMUSEParameterV5(s, i) {
			if mg[i] != 0 {
				t.Fatal("MUSE ownership")
			}
			continue
		}
		old := w[i]
		w[i] = old + 1e-5
		plus, _, _ := fidelityMUSEObjectiveV5(s, w, temps, batch)
		w[i] = old - 1e-5
		minus, _, _ := fidelityMUSEObjectiveV5(s, w, temps, batch)
		w[i] = old
		err := math.Abs(s.Config.AssessmentWeight*(plus-minus)/2e-5 - mg[i])
		maximum = math.Max(maximum, err)
		count++
		if err > 2e-7 {
			t.Fatalf("assessment coordinate%d err%g", i, err)
		}
	}
	t.Logf("binary assessment numerical coordinates %d maxerr %.12g", count, maximum)
}

func TestFidelityV5ActualUpdateOwnershipAndFrozenControl(t *testing.T) {
	for _, options := range []FidelityLearningV5{{}, {FreezeRecurrence: true}, {AssessmentOnly: true}} {
		b := coordinatedFixtureV5(t)
		batch := fidelityTestBatchV5(t, b)
		batch.Fidelity = &options
		if options.AssessmentOnly {
			trace := batch.Episodes[0].Traces[1]
			trace = append([]CognitiveInput(nil), trace...)
			last := len(trace) - 1
			trace[last].Mask = append([]bool(nil), trace[last].Mask...)
			clear(trace[last].Mask[:3])
			p, err := b.PreviewCoordinatedV5(trace, Phase4DropV5{})
			if err != nil {
				t.Fatal(err)
			}
			q := FidelityAssessmentPolicyV5(p.Policy, trace[last].Mask)
			draw := q[3] / 2
			trace[last].Draw = draw
			batch.Episodes = []CoordinatedEpisodeV5{{Traces: [][]CognitiveInput{trace}, Frames: []Phase4FrameV5{{Features: p.Features, Mask: trace[last].Mask, Action: 3, Reward: 1, Draw: draw, BehaviorProbability: q[3], BehaviorValue: p.Value}}}}
		}
		batch.Evidence.Digest = CoordinatedEvidenceDigestV5(batch)
		s := b.BrainState.CognitiveModel
		before, _ := b.CognitiveParameters()
		m1, m2 := append([]float64(nil), s.FirstMoment...), append([]float64(nil), s.SecondMoment...)
		out, err := b.LearnCoordinatedV5(batch, time.Unix(2400000000, 0))
		if err != nil {
			t.Fatal(err)
		}
		after, _ := b.CognitiveParameters()
		recurrentChanges := 0
		for i := range before {
			allowed := fidelityMUSEParameterV5(s, i) || !options.AssessmentOnly && (i >= s.Phase4V5.BaseParameters || !options.FreezeRecurrence && fidelityRecurrentParameterV5(s, i))
			if !allowed && (before[i] != after[i] || m1[i] != s.FirstMoment[i] || m2[i] != s.SecondMoment[i]) {
				t.Fatal("frozen parameter or moments changed", i)
			}
			if fidelityRecurrentParameterV5(s, i) && before[i] != after[i] {
				recurrentChanges++
			}
		}
		if out.MUSEGradientNorm == 0 || !options.AssessmentOnly && !options.FreezeRecurrence && (recurrentChanges == 0 || out.RecurrentGradientNorm == 0) {
			t.Fatal("no responsible neural update")
		}
		if (options.FreezeRecurrence || options.AssessmentOnly) && recurrentChanges != 0 {
			t.Fatal("recurrent control moved")
		}
		if b.ValidateCoordinatedBatchV5(batch) == nil {
			t.Fatal("stale evidence accepted")
		}
		t.Logf("options%+v updated recurrent %d gradient%g", options, recurrentChanges, out.RecurrentGradientNorm)
	}
}

func TestFidelityV5DefaultAndRareActionRetention(t *testing.T) {
	b := coordinatedFixtureV5(t)
	tr := coordinatedTraceV5()
	old, err := b.PreviewCoordinatedV5(tr, Phase4DropV5{})
	if err != nil {
		t.Fatal(err)
	}
	probe := coordinatedTraceV5()
	probe[0].NeuralMeta.Mode = CoordinatedFidelityV5
	if _, err = b.PreviewCoordinatedV5(probe, Phase4DropV5{}); err != nil {
		t.Fatal(err)
	}
	now, _ := b.PreviewCoordinatedV5(tr, Phase4DropV5{})
	if !reflect.DeepEqual(old, now) {
		t.Fatal("preview altered default state")
	}
	batch := fidelityTestBatchV5(t, b)
	s := b.BrainState.CognitiveModel
	w, _ := b.CognitiveParameters()
	w[s.Phase4V5.ActorBias.Offset+3] = -220
	a := [][]float64{{1, 1}}
	r := [][]float64{{.975, 1}}
	st, g := fidelityPolicyObjectiveV5(s, w, batch, a, r)
	if !finiteNumber(st.Loss) {
		t.Fatal("rare action nonfinite")
	}
	i := s.Phase4V5.ActorBias.Offset + 3
	oldW := w[i]
	w[i] = oldW + 1e-4
	plus, _ := fidelityPolicyObjectiveV5(s, w, batch, a, r)
	w[i] = oldW - 1e-4
	minus, _ := fidelityPolicyObjectiveV5(s, w, batch, a, r)
	if math.Abs((plus.Loss-minus.Loss)/2e-4-g[i]) > 1e-7 {
		t.Fatal("rare-action gradient was clipped")
	}
}

func TestFidelityV5RejectsMalformedAndUnrecordedRoots(t *testing.T) {
	b := coordinatedFixtureV5(t)
	for _, kind := range []string{"shape", "history", "mask"} {
		batch := fidelityTestBatchV5(t, b)
		if kind == "shape" {
			batch.Episodes[0].Traces[0][0].Features = nil
		}
		if kind == "history" {
			batch.Episodes[0].Traces[0][0].Features = append([]float64(nil), batch.Episodes[0].Traces[0][0].Features...)
			batch.Episodes[0].Traces[0][0].Features[5] = 1
		}
		if kind == "mask" {
			batch.Fidelity.AssessmentOnly = true
			batch.Episodes[0].Frames = batch.Episodes[0].Frames[:1]
			batch.Episodes[0].Traces = batch.Episodes[0].Traces[:1]
			batch.Episodes[0].Traces[0][0].Mask = nil
		}
		batch.Evidence.Digest = CoordinatedEvidenceDigestV5(batch)
		if err := b.ValidateCoordinatedBatchV5(batch); err == nil {
			t.Fatal("invalid root accepted", kind)
		}
	}
}

func TestFidelityV5BMPSDelayedInformationAndCost(t *testing.T) {
	// Two legal bit observations reveal an XOR target. The first observation
	// alone has zero information value for the decision; the second completes it.
	b := MetaBelief{Probabilities: []float64{.25, .25, .25, .25}, Remaining: 2, Horizon: 2}
	for theta := 0; theta < 4; theta++ {
		y := (theta & 1) ^ ((theta >> 1) & 1)
		u := []float64{0, 0}
		u[y] = 1
		b.Utilities = append(b.Utilities, u)
	}
	for bit := 0; bit < 2; bit++ {
		c := MetaComputation{Cost: .05}
		for theta := 0; theta < 4; theta++ {
			o := (theta >> bit) & 1
			lik := []float64{0, 0}
			lik[o] = 1
			c.Likelihood = append(c.Likelihood, lik)
			c.Groups = append(c.Groups, o)
		}
		b.Computations = append(b.Computations, c)
	}
	f, err := BMPSInformation(b)
	if err != nil {
		t.Fatal(err)
	}
	for _, x := range f {
		if math.Abs(x.Myopic) > 1e-12 || math.Abs(x.Perfect-.5) > 1e-12 || math.Abs(x.Subset) > 1e-12 {
			t.Fatal("incorrect delayed-information bounds", x)
		}
	}
	d, err := SelectBMPS(b, [4]float64{1.0 / 3, 1.0 / 3, 1.0 / 3, 1})
	if err != nil || d.VOC[0] <= 0 {
		t.Fatal("VPI feature lost non-myopic opportunity", err)
	}
	b.Computations[0].Cost = .6
	b.Computations[1].Cost = .6
	d, err = SelectBMPS(b, [4]float64{1.0 / 3, 1.0 / 3, 1.0 / 3, 1})
	if err != nil || d.Computation != -1 {
		t.Fatal("costly computation not rejected", err)
	}
}
