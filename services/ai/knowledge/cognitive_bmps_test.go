package knowledge

import (
	"math"
	"testing"
	"time"
)

func bmpsTestBelief() MetaBelief {
	return MetaBelief{Probabilities: []float64{.5, .5}, Utilities: [][]float64{{1, 0}, {0, 1}}, Remaining: 4, Computations: []MetaComputation{{Cost: .1, Likelihood: [][]float64{{1, 0}, {0, 1}}, Groups: []int{0, 1}}}}
}
func TestBMPSMatchesExactFiniteInformation(t *testing.T) {
	b := bmpsTestBelief()
	f, err := BMPSInformation(b)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(f[0].Myopic-.5) > 1e-14 || math.Abs(f[0].Perfect-.5) > 1e-14 || math.Abs(f[0].Subset-.5) > 1e-14 {
		t.Fatalf("wrong information values %+v", f)
	}
	d, err := SelectBMPS(b, [4]float64{.2, .4, .4, 1})
	if err != nil || d.Computation != 0 || math.Abs(d.VOC[0]-.4) > 1e-14 {
		t.Fatalf("wrong VOC %+v %v", d, err)
	}
	post, err := b.After(0, 1)
	if err != nil {
		t.Fatal(err)
	}
	d, err = SelectBMPS(post, [4]float64{.2, .4, .4, 1})
	if err != nil || d.Computation != -1 || d.TerminalAction != 1 {
		t.Fatalf("did not stop with complete information %+v %v", d, err)
	}
	b.Computations[0].Cost = .6
	d, _ = SelectBMPS(b, [4]float64{1, 0, 0, 1})
	if d.Computation != -1 {
		t.Fatal("ignores computation cost")
	}
}
func TestBMPSKeepsNonMyopicPerfectInformation(t *testing.T) {
	b := MetaBelief{Probabilities: []float64{.25, .25, .25, .25}, Utilities: [][]float64{{1, 0}, {0, 1}, {0, 1}, {1, 0}}, Remaining: 4, Computations: []MetaComputation{{Cost: .1, Likelihood: [][]float64{{1, 0}, {1, 0}, {0, 1}, {0, 1}}, Groups: []int{0, 0, 1, 1}}, {Cost: .1, Likelihood: [][]float64{{1, 0}, {0, 1}, {1, 0}, {0, 1}}, Groups: []int{0, 1, 0, 1}}}}
	f, err := BMPSInformation(b)
	if err != nil {
		t.Fatal(err)
	}
	if f[0].Myopic != 0 || f[0].Subset != 0 || f[0].Perfect != .5 {
		t.Fatalf("lost complementary information %+v", f)
	}
	greedy, _ := SelectBMPS(b, [4]float64{1, 0, 0, 1})
	full, _ := SelectBMPS(b, [4]float64{0, 1, 0, 1})
	if greedy.Computation != -1 || full.Computation < 0 {
		t.Fatal("BMPS collapsed to myopic heuristic")
	}
}
func TestBMPSBayesianOptimizationAndCanonicalWeights(t *testing.T) {
	k := cognitiveTestBrain(t)
	cfg := k.BrainState.CognitiveModel.Config
	history := []BMPSEvaluation{}
	for i := 0; i < 20; i++ {
		p, err := ProposeBMPS(history, cfg.Seed)
		if err != nil {
			t.Fatal(err)
		}
		value := -math.Pow(p[0]-.7, 2) - math.Pow(p[1]-.2, 2) - math.Pow(p[2]-.1, 2)
		history = append(history, BMPSEvaluation{Point: p, Return: value, Episodes: 40})
	}
	best := math.Inf(-1)
	for _, e := range history {
		best = math.Max(best, e.Return)
	}
	if best < -.1 {
		t.Fatalf("GP/EI did not improve objective %g", best)
	}
	ev := CognitiveEvidence{ID: "bo", Source: "test", Verifier: "returns", Digest: CognitiveDigest(history)}
	if err := k.FitCognitiveBMPS(history, ev, time.Unix(1800000002, 0)); err != nil {
		t.Fatal(err)
	}
	w, _ := k.CognitiveBMPSWeights()
	if err := ValidateBMPSWeights(w, cfg.Horizon); err != nil {
		t.Fatal(err)
	}
	raw, _ := k.CanonicalJSON()
	if err := ValidateCognitiveModelState(raw, true); err != nil {
		t.Fatal(err)
	}
}
func BenchmarkBMPSFiniteBelief(b *testing.B) {
	x := bmpsTestBelief()
	w := [4]float64{.2, .4, .4, 1}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := SelectBMPS(x, w); err != nil {
			b.Fatal(err)
		}
	}
}
