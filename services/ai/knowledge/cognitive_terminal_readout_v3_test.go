package knowledge

import (
	"math"
	"reflect"
	"testing"
	"time"
)

func terminalTestBatchV3(k *KnowledgeBase) CognitiveTrainingBatch {
	c := k.BrainState.CognitiveModel.Config
	n := c.NeuralMeta
	y := 1
	x := metaTestInput()
	x.NeuralMeta.Mode = "recurrent"
	x.Mask = []bool{false, true, true, true}
	eps := []CognitiveTrainingEpisode{}
	for a := n.Computations; a < c.Actions; a++ {
		success := -1
		reward := 0.
		if a-n.Computations == y {
			success = 0
			reward = 1
		}
		if a == c.Actions-1 {
			reward = .35
		}
		eps = append(eps, CognitiveTrainingEpisode{VerifiedTarget: &y, SuccessAt: success, Frames: []CognitiveTrainingFrame{{Features: x.Features, Mask: x.Mask, NeuralMeta: x.NeuralMeta, Action: a, Reward: reward}}})
	}
	w, _ := k.CognitiveParameters()
	return CognitiveTrainingBatch{TerminalReadoutOnly: true, Episodes: eps, ParametersHash: CognitiveDigest(w), Evidence: CognitiveEvidence{ID: "terminal-probes", Source: "native-verified-branch", Verifier: "actual-outcome", Digest: CognitiveDigest(eps)}}
}
func TestV3TerminalReadoutConstrainedGradientAndFrozenObservation(t *testing.T) {
	k := NewBrain()
	if e := k.InitializeCognitiveModel(metaTestConfig(), time.Unix(1800000000, 0)); e != nil {
		t.Fatal(e)
	}
	s := k.BrainState.CognitiveModel
	w, _ := k.CognitiveParameters()
	b := terminalTestBatchV3(k)
	if e := k.ValidateCognitiveBatch(b); e != nil {
		t.Fatal(e)
	}
	_, g := cognitiveTerminalReadoutObjectiveV3(s, w, b)
	epsilon := 1e-5
	maxError := 0.
	count := 0
	for i := range w {
		if !terminalReadoutParameterV3(s, i) {
			if g[i] != 0 {
				t.Fatal("protected gradient", i)
			}
			continue
		}
		old := w[i]
		w[i] = old + epsilon
		a, _ := cognitiveTerminalReadoutObjectiveV3(s, w, b)
		w[i] = old - epsilon
		bb, _ := cognitiveTerminalReadoutObjectiveV3(s, w, b)
		w[i] = old
		delta := math.Abs((a.Loss-bb.Loss)/(2*epsilon) - g[i])
		maxError = math.Max(delta, maxError)
		count++
		if delta > 3e-7 {
			t.Fatal("constrained derivative", i, delta)
		}
	}
	x := metaTestInput()
	x.NeuralMeta.Mode = "recurrent"
	before, _ := k.CognitivePreview(x)
	for i := 0; i < 100; i++ {
		b = terminalTestBatchV3(k)
		if _, e := k.LearnCognitiveBatch(b, time.Unix(1800000001+int64(i), 0)); e != nil {
			t.Fatal(e)
		}
	}
	after, _ := k.CognitivePreview(x)
	if !reflect.DeepEqual(before.Hidden, after.Hidden) || !reflect.DeepEqual(before.Policy, after.Policy) || !reflect.DeepEqual(before.ClassBelief, after.ClassBelief) {
		t.Fatal("changed observation/actor/world mapping")
	}
	actual, _ := k.CognitiveParameters()
	changed := 0
	for i := range actual {
		if !terminalReadoutParameterV3(s, i) && actual[i] != w[i] {
			t.Fatal("protected parameter changed", i)
		}
		if actual[i] != w[i] {
			changed++
		}
	}
	if changed == 0 {
		t.Fatal("no canonical learning")
	}
	t.Logf("V3 %d eligible derivatives; max error=%g; changed edges=%d", count, maxError, changed)
	b = terminalTestBatchV3(k)
	if e := k.CalibrateNeuralMeta(b, time.Unix(1800001000, 0)); e != nil {
		t.Fatal(e)
	}
	cal, _ := k.CognitivePreview(x)
	if !reflect.DeepEqual(cal.ClassBelief, before.ClassBelief) {
		t.Fatal("readout calibration changed class information")
	}
}
func TestV3TerminalReadoutRejectInvalidEvidenceAndPolicy(t *testing.T) {
	k := NewBrain()
	k.InitializeCognitiveModel(metaTestConfig(), time.Unix(1800000000, 0))
	b := terminalTestBatchV3(k)
	b.LearnPolicy = true
	if k.ValidateCognitiveBatch(b) == nil {
		t.Fatal("accepted off-policy REINFORCE")
	}
	b = terminalTestBatchV3(k)
	b.Episodes[0].SuccessAt = 0
	b.Evidence.Digest = CognitiveDigest(b.Episodes)
	if k.ValidateCognitiveBatch(b) == nil {
		t.Fatal("accepted false outcome")
	}
	b = terminalTestBatchV3(k)
	b.Episodes[0].Frames[0].Features = []float64{0}
	b.Evidence.Digest = CognitiveDigest(b.Episodes)
	if k.ValidateCognitiveBatch(b) == nil {
		t.Fatal("accepted short features")
	}
}
