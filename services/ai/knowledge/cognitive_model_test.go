package knowledge

import (
	"encoding/json"
	"math"
	"reflect"
	"testing"
	"time"
)

func cognitiveTestConfig() CognitiveModelConfig {
	return CognitiveModelConfig{Version: CognitiveModelVersion, Inputs: 3, Hidden: 2, Actions: 3, Horizon: 4, Seed: 17, LearningRate: .003, Gamma: 1, ValueWeight: .2, EntropyWeight: .01, AssessmentWeight: .3, WorldWeight: .2, RegulationSteps: 2, RegulationRate: .02}
}
func cognitiveTestBrain(t testing.TB) *Brain {
	t.Helper()
	k := NewBrain()
	if err := k.InitializeCognitiveModel(cognitiveTestConfig(), time.Unix(1800000000, 0).UTC()); err != nil {
		t.Fatal(err)
	}
	return k
}

func TestCognitiveDimensionValidationCannotOverflow(t *testing.T) {
	c := cognitiveTestConfig()
	c.Inputs = int(^uint(0) >> 1)
	c.Hidden = 4
	if c.Validate() == nil {
		t.Fatal("overflowing dimensions accepted")
	}
	c = cognitiveTestConfig()
	c.Hidden = int(^uint(0) >> 1)
	if c.Validate() == nil {
		t.Fatal("overflowing hidden width accepted")
	}
}
func TestCognitiveGRUMatchesScalarEquations(t *testing.T) {
	k := cognitiveTestBrain(t)
	s := k.BrainState.CognitiveModel
	c, l := s.Config, s.Layout
	w, _ := k.CognitiveParameters()
	x := []float64{.7, -.2, .1}
	hp := []float64{-.3, .4}
	f := cognitiveGRU(c, l, w, x, hp)
	for i := 0; i < c.Hidden; i++ {
		a := make([]float64, 3)
		b := make([]float64, 3)
		for g := 0; g < 3; g++ {
			row := g*c.Hidden + i
			a[g] = w[l.BI.Offset+row]
			b[g] = w[l.BH.Offset+row]
			for j := range x {
				a[g] += w[l.WI.Offset+row*c.Inputs+j] * x[j]
			}
			for j := range hp {
				b[g] += w[l.WH.Offset+row*c.Hidden+j] * hp[j]
			}
		}
		r := 1 / (1 + math.Exp(-a[0]-b[0]))
		z := 1 / (1 + math.Exp(-a[1]-b[1]))
		n := math.Tanh(a[2] + r*b[2])
		want := (1-z)*n + z*hp[i]
		if math.Abs(want-f.h[i]) > 1e-14 {
			t.Fatalf("GRU coordinate %d %.16g != %.16g", i, want, f.h[i])
		}
	}
}
func TestCognitiveBPTTMatchesFiniteDifferences(t *testing.T) {
	k := cognitiveTestBrain(t)
	s := k.BrainState.CognitiveModel
	w, _ := k.CognitiveParameters()
	mask := []bool{true, true, true}
	batch := CognitiveTrainingBatch{LearnPolicy: true, Episodes: []CognitiveTrainingEpisode{{SuccessAt: 1, Frames: []CognitiveTrainingFrame{{Features: []float64{.4, -.7, .2}, Mask: mask, Action: 0, Reward: -.1}, {Features: []float64{-.2, .3, .8}, Mask: mask, Action: 2, Reward: 1}}}, {SuccessAt: -1, Frames: []CognitiveTrainingFrame{{Features: []float64{-.9, .4, .3}, Mask: mask, Action: 1, Reward: -.2}, {Features: []float64{.1, .5, -.7}, Mask: mask, Action: 0, Reward: 0}}}}}
	target, err := makeCognitiveTargets(s.Config, s.Layout, w, batch, false)
	if err != nil {
		t.Fatal(err)
	}
	_, g := cognitiveBatchObjective(s.Config, s.Layout, w, batch, target)
	eps := 1e-5
	maxErr := 0.0
	for i := range w {
		old := w[i]
		w[i] = old + eps
		a, _ := cognitiveBatchObjective(s.Config, s.Layout, w, batch, target)
		w[i] = old - eps
		b, _ := cognitiveBatchObjective(s.Config, s.Layout, w, batch, target)
		w[i] = old
		n := (a.Loss - b.Loss) / (2 * eps)
		e := math.Abs(n - g[i])
		maxErr = math.Max(maxErr, e)
		if e > 2e-7 {
			t.Fatalf("gradient %d native=%g finite=%g error=%g", i, g[i], n, e)
		}
	}
	t.Logf("all %d gradients checked, maximum absolute error %.3g", len(w), maxErr)
}
func TestCognitiveReadOnlyAndCanonicalRoundTrip(t *testing.T) {
	k := cognitiveTestBrain(t)
	before, _ := k.CanonicalJSON()
	in := CognitiveInput{Features: []float64{.4, .1, -.2}, Mask: []bool{true, true, true}, Reset: true, Regulate: true, Draw: .3}
	a, err := k.CognitivePreview(in)
	if err != nil {
		t.Fatal(err)
	}
	after, _ := k.CanonicalJSON()
	if !reflect.DeepEqual(before, after) {
		t.Fatal("preview changed canonical state")
	}
	if err := ValidateCognitiveModelState(before, true); err != nil {
		t.Fatal(err)
	}
	if err := ValidateCognitiveModelState(before, false); err == nil {
		t.Fatal("unauthenticated cognitive state accepted")
	}
	p := t.TempDir() + "/brain.json"
	if err := k.Save(p); err != nil {
		t.Fatal(err)
	}
	loaded := NewBrain()
	if err := loaded.Load(p); err != nil {
		t.Fatal(err)
	}
	b, err := loaded.CognitivePreview(in)
	if err != nil || !reflect.DeepEqual(a, b) {
		t.Fatalf("restore mismatch %v", err)
	}
	var raw map[string]any
	json.Unmarshal(before, &raw)
	raw["cognitive_model"].(map[string]any)["first_moment"] = []float64{}
	tampered, _ := json.Marshal(raw)
	if ValidateCognitiveModelState(tampered, true) == nil {
		t.Fatal("tampered optimizer accepted")
	}
}
func TestCognitiveFeedbackChangesCanonicalSynapses(t *testing.T) {
	k := cognitiveTestBrain(t)
	w, _ := k.CognitiveParameters()
	mask := []bool{true, true, true}
	in := CognitiveInput{Features: []float64{.9, -.2, .1}, Mask: mask, Reset: true, Draw: .2}
	p, _ := k.CognitivePreview(in)
	episodes := []CognitiveTrainingEpisode{{SuccessAt: 0, Frames: []CognitiveTrainingFrame{{Features: in.Features, Mask: mask, Action: p.Action, Draw: in.Draw, Behaviour: p.Policy, Reward: 1}}}}
	batch := CognitiveTrainingBatch{Episodes: episodes, ParametersHash: p.ParametersHash, LearnPolicy: true, Evidence: CognitiveEvidence{ID: "verified-1", Source: "test-simulator", Verifier: "independent-outcome", Digest: CognitiveDigest(episodes)}}
	n := len(k.Registry.Nodes())
	stats, err := k.LearnCognitiveBatch(batch, time.Unix(1800000001, 0))
	if err != nil {
		t.Fatal(err)
	}
	got, _ := k.CognitiveParameters()
	if reflect.DeepEqual(w, got) || stats.GradientNorm == 0 {
		t.Fatal("no neural learning")
	}
	if n != len(k.Registry.Nodes()) {
		t.Fatal("each lesson allocated another circuit")
	}
	if err := k.ValidateCognitiveBatch(batch); err == nil {
		t.Fatal("stale on-policy trace accepted")
	}
	s, _ := k.CanonicalJSON()
	if err := ValidateCognitiveModelState(s, true); err != nil {
		t.Fatal(err)
	}
}
func TestCognitiveRecurrentResultAffectsNextState(t *testing.T) {
	k := cognitiveTestBrain(t)
	at := time.Unix(1800000001, 0)
	x := CognitiveInput{Features: []float64{.8, -.2, .1}, Mask: []bool{true, true, true}, Reset: true, Draw: .1}
	a, err := k.AdvanceCognitiveModel(x, at)
	if err != nil {
		t.Fatal(err)
	}
	x.Reset = false
	b, _ := k.CognitivePreview(x)
	x.NoRecurrence = true
	c, _ := k.CognitivePreview(x)
	if reflect.DeepEqual(a.Hidden, b.Hidden) || reflect.DeepEqual(b.Policy, c.Policy) {
		t.Fatal("returned information is not used recurrently")
	}
}
func BenchmarkCognitivePreview(b *testing.B) {
	k := cognitiveTestBrain(b)
	x := CognitiveInput{Features: []float64{.2, -.4, .1}, Mask: []bool{true, true, true}, Draw: .5}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := k.CognitivePreview(x); err != nil {
			b.Fatal(err)
		}
	}
}
func BenchmarkCognitiveRegulatedPreview(b *testing.B) {
	k := cognitiveTestBrain(b)
	x := CognitiveInput{Features: []float64{.2, -.4, .1}, Mask: []bool{true, true, true}, Regulate: true, Draw: .5}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := k.CognitivePreview(x); err != nil {
			b.Fatal(err)
		}
	}
}
