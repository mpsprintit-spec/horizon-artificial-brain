package knowledge

import (
	"math"
	"reflect"
	"testing"
	"time"
)

func coordinatedFixtureV5(t *testing.T) *Brain {
	t.Helper()
	b := NewBrain()
	c := CognitiveModelConfig{Version: CognitiveModelV2, Inputs: 28, Hidden: 32, Actions: 8, Horizon: 4, Seed: 101, LearningRate: .003, Gamma: 1, ValueWeight: .2, EntropyWeight: .02, AssessmentWeight: .2, NeuralMeta: &NeuralMetaConfig{Classes: 4, Computations: 3, Outcomes: 3, ContextSize: 5, ResultOffset: 5, OperationOffset: 8, ActionOffset: 11, HypothesisOffset: 19, PlanMarker: 23, StepIndex: 24, JointWeight: .7, RewardWeight: .3, PlanDepth: 1}}
	at := time.Unix(2200400000, 0)
	if e := b.InitializeCognitiveModel(c, at); e != nil {
		t.Fatal(e)
	}
	if e := b.InitializePhase2V5(Phase2SpecV5{Version: Phase2VersionV5, Representation: "pairwise", Optimizer: "historical-global"}, at); e != nil {
		t.Fatal(e)
	}
	if e := b.InitializePhase4V5(Phase4SpecV5{Version: Phase4VersionV5, Seed: 101, LearningRate: .003, EntropyWeight: .08, ValueWeight: .2}, at); e != nil {
		t.Fatal(e)
	}
	return b
}
func coordinatedTraceV5() []CognitiveInput {
	x := CognitiveInput{Features: make([]float64, 28), Mask: []bool{false, false, false, true, true, true, true, true}, Reset: true, Draw: .1, NeuralMeta: &NeuralMetaInput{Mode: "recurrent", Costs: []float64{.025, .03, .035}, Remaining: 3}}
	x.Features[0] = 1
	return []CognitiveInput{x}
}
func TestCoordinatedV5SelectedOutcomeGradientsAndFrozenWorld(t *testing.T) {
	b := coordinatedFixtureV5(t)
	s := b.BrainState.CognitiveModel
	w, _ := b.CognitiveParameters()
	tr := coordinatedTraceV5()
	p, e := b.PreviewCoordinatedV5(tr, Phase4DropV5{})
	if e != nil {
		t.Fatal(e)
	}
	f := Phase4FrameV5{Features: p.Features, Mask: tr[0].Mask, Action: p.Action, Reward: 1, Draw: tr[0].Draw, BehaviorProbability: p.Policy[p.Action], BehaviorValue: p.Value}
	ep := CoordinatedEpisodeV5{Traces: [][]CognitiveInput{tr}, Frames: []Phase4FrameV5{f}}
	batch := CoordinatedBatchV5{Episodes: []CoordinatedEpisodeV5{ep}, ParametersHash: p.ParametersHash}
	batch.Evidence = CognitiveEvidence{ID: "actual", Source: "unit", Verifier: "selected-reward", Digest: CognitiveDigest(batch.Episodes)}
	temps := b.metaTemperatures(s)
	_, _, g, e := coordinatedMUSEObjectiveV5(s, w, temps, batch)
	if e != nil {
		t.Fatal(e)
	}
	maximum, count := 0., 0
	for i := range w {
		if !coordinatedMUSEParameterV5(s, i) {
			if g[i] != 0 {
				t.Fatal("gradient escaped MUSE ownership")
			}
			continue
		}
		old := w[i]
		w[i] = old + 1e-5
		a, r, _, _ := coordinatedMUSEObjectiveV5(s, w, temps, batch)
		plus := s.Config.AssessmentWeight*a + s.Config.NeuralMeta.RewardWeight*r
		w[i] = old - 1e-5
		a, r, _, _ = coordinatedMUSEObjectiveV5(s, w, temps, batch)
		minus := s.Config.AssessmentWeight*a + s.Config.NeuralMeta.RewardWeight*r
		w[i] = old
		err := math.Abs((plus-minus)/2e-5 - g[i])
		maximum = math.Max(maximum, err)
		count++
		if err > 2e-7 {
			t.Fatalf("MUSEgradient%d error%g", i, err)
		}
	}
	before, _ := b.CognitiveParameters()
	m1 := append([]float64(nil), s.FirstMoment...)
	m2 := append([]float64(nil), s.SecondMoment...)
	out, e := b.LearnCoordinatedV5(batch, time.Unix(2200400001, 0))
	if e != nil {
		t.Fatal(e)
	}
	after, _ := b.CognitiveParameters()
	changed := 0
	for i := range before {
		allowed := i >= s.Phase4V5.BaseParameters || coordinatedMUSEParameterV5(s, i)
		if !allowed && (before[i] != after[i] || m1[i] != s.FirstMoment[i] || m2[i] != s.SecondMoment[i]) {
			t.Fatal("frozen parameter/moment moved", i)
		}
		if before[i] != after[i] {
			changed++
		}
	}
	if changed == 0 || out.MUSEGradientNorm == 0 || !b.HasCoordinatedV5() {
		t.Fatal("no actual coordinated learning")
	}
	if b.ValidateCoordinatedBatchV5(batch) == nil {
		t.Fatal("stale batch accepted")
	}
	if _, e = b.PreviewPhase4V5(Phase2ProbeV5{Trace: tr}, Phase4DropV5{}); e == nil {
		t.Fatal("wrong feature contract accepted")
	}
	t.Logf("MUSE numerical derivatives%d maxerr%g; canonical changed%d; frozen world, observation GRU, old actor and moments preserved", count, maximum, changed)
}
func TestCoordinatedV5ContinuationCostAndMechanismInterventions(t *testing.T) {
	b := coordinatedFixtureV5(t)
	// Isolate the direct scheduler charge from learned sensitivity to cost inputs.
	s := b.BrainState.CognitiveModel
	for row := 0; row < 16; row++ {
		for col := 45; col < 48; col++ {
			edge := s.Layout.Edges[s.Phase4V5.WI.Offset+row*58+col]
			b.adjustCompositionParameter([]NodeID{edge.From}, []NodeID{edge.To}, 0, 1, time.Unix(2200400000, 0), false)
		}
	}
	tr := coordinatedTraceV5()
	tr[0].Mask[0] = true
	p, e := b.PreviewCoordinatedV5(tr, Phase4DropV5{})
	if e != nil {
		t.Fatal(e)
	}
	for a := 49; a < 58; a++ {
		if p.Features[a] != 0 {
			t.Fatal("second actor entered final policy")
		}
	}
	if p.TerminalUtility[4] != .35 {
		t.Fatal("abstain payoff")
	}
	tr[0].NeuralMeta.Costs = append([]float64(nil), tr[0].NeuralMeta.Costs...)
	tr[0].NeuralMeta.Costs[0] += .01
	q, e := b.PreviewCoordinatedV5(tr, Phase4DropV5{})
	if e != nil {
		t.Fatal(e)
	}
	if math.Abs(q.VOC[0]-p.VOC[0]+.01) > 1e-12 {
		t.Fatal("VOC cost counted incorrectly", p.VOC, q.VOC)
	}
	a, _ := b.PreviewCoordinatedV5(tr, Phase4DropV5{MUSE: true})
	v, _ := b.PreviewCoordinatedV5(tr, Phase4DropV5{VOC: true})
	if reflect.DeepEqual(a.Policy, q.Policy) || reflect.DeepEqual(v.Policy, q.Policy) {
		t.Fatal("mechanism intervention disconnected")
	}
	before, _ := b.CanonicalJSON()
	tr[0].Belief = &MetaBelief{}
	if _, e = b.PreviewCoordinatedV5(tr, Phase4DropV5{}); e == nil {
		t.Fatal("oracle belief accepted")
	}
	after, _ := b.CanonicalJSON()
	if !reflect.DeepEqual(before, after) {
		t.Fatal("preview mutated N")
	}
}
