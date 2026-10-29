package knowledge

// Experimental sequence contract. All learned numbers are reconstructed from
// canonical signed dynamic synapses. Arrays and attention traces are detached
// workspaces; no external model or persistent weight cache exists.
import (
	"errors"
	"math"
	"time"
)

const MaxCompositionSequence = 64

var sequenceInputBanks = [8]int{18, 19, 20, 21, 22, 23, 25, 26}

func validateCompositionSequence(seq [][]float64) error {
	if len(seq) > MaxCompositionSequence {
		return errors.New("sequence resource budget exceeded")
	}
	for _, v := range seq {
		if !compositionNumbers(v, 7) || (v[6] != 0 && v[6] != 1) {
			return errors.New("sequence requires six bounded measurements and explicit presence")
		}
	}
	return nil
}
func validateCompositionMode(c CompositionConfig, in CompositionInput) error {
	if c.Version != NeuralCompositionMetricV6 && (in.RequiredPrecision != nil || len(in.MeasurementError) > 0) {
		return errors.New("precision metadata requires metric contract")
	}
	if c.Version != NeuralCompositionSequenceV4 && !IsNumericalCompositionVersion(c.Version) {
		if len(in.Sequence) > 0 {
			return errors.New("sequence requires explicit v4 contract")
		}
		return nil
	}
	if len(in.Words) == 0 && len(in.Sequence) == 0 {
		return nil
	}
	if len(in.Words) != 1 || len(in.Sequence) == 0 {
		return errors.New("v4 requires one learned query and an observed sequence")
	}
	return nil
}

type sequenceWeights struct {
	W  [compositionHidden][8]float64
	R  [compositionHidden][compositionHidden]float64
	HB [compositionHidden]float64
	Q  [compositionHidden]float64
	K  [compositionHidden]float64
	H  [compositionOutputs][compositionHidden]float64
	B  [compositionOutputs]float64
}
type sequenceTrace struct {
	X                   [][8]float64
	States              [][compositionHidden]float64
	Values, Keys        [][compositionHidden]float64
	Attention           []float64
	Context, Activation [compositionHidden]float64
	Z                   [compositionOutputs]float64
}

func (k *KnowledgeBase) initializeSequenceComposition(bank [compositionBanks][]NodeID, c CompositionConfig, at time.Time) {
	for h := 0; h < compositionHidden; h++ {
		k.adjustCompositionParameter(bank[24], bank[h], 0, 0, at, true)
		for j, b := range sequenceInputBanks {
			k.adjustCompositionParameter(bank[b], bank[h], .15*compositionRandom(c.Seed, 100+h, j), 0, at, true)
		}
		for j := 0; j < compositionHidden; j++ {
			v := .02 * compositionRandom(c.Seed, 200+h, j)
			if h == j {
				v += .65
			}
			k.adjustCompositionParameter(bank[j], bank[h], v, 0, at, true)
		}
	}
}
func (k *KnowledgeBase) loadSequenceComposition(bank [compositionBanks][]NodeID, query []NodeID) sequenceWeights {
	var w sequenceWeights
	for h := 0; h < compositionHidden; h++ {
		w.Q[h], _ = k.compositionParameter(query, bank[h])
		w.K[h], _ = k.compositionParameter(query, bank[12+h])
		w.HB[h], _ = k.compositionParameter(bank[24], bank[h])
		for j, b := range sequenceInputBanks {
			w.W[h][j], _ = k.compositionParameter(bank[b], bank[h])
		}
		for j := 0; j < compositionHidden; j++ {
			w.R[h][j], _ = k.compositionParameter(bank[j], bank[h])
		}
	}
	for o := 0; o < compositionOutputs; o++ {
		w.B[o], _ = k.compositionParameter(bank[24], bank[12+o])
		for h := 0; h < compositionHidden; h++ {
			w.H[o][h], _ = k.compositionParameter(bank[h], bank[12+o])
		}
	}
	return w
}
func sequenceForward(in CompositionInput, w sequenceWeights) sequenceTrace {
	n := len(in.Sequence)
	t := sequenceTrace{X: make([][8]float64, n), States: make([][compositionHidden]float64, n), Attention: make([]float64, n), Values: make([][compositionHidden]float64, n), Keys: make([][compositionHidden]float64, n), Z: w.B}
	maxScore := math.Inf(-1)
	for i, v := range in.Sequence {
		for j := 0; j < 6; j++ {
			t.X[i][j] = v[j] * v[6]
		}
		t.X[i][7] = v[6]
		if n > 1 {
			t.X[i][6] = 2*float64(i)/float64(n-1) - 1
		}
		for h := 0; h < compositionHidden; h++ {
			a := w.HB[h]
			for j, x := range t.X[i] {
				a += w.W[h][j] * x
			}
			if i > 0 {
				for j, x := range t.States[i-1] {
					a += w.R[h][j] * x
				}
			}
			t.States[i][h] = math.Tanh(a)
			t.Values[i][h] = t.States[i][h]
			if h < 6 {
				t.Values[i][h] += t.X[i][h]
			}
			frequencies := [6]float64{1, 2, 3, 5, 8, 13}
			angle := 2 * math.Pi * frequencies[h/2] * float64(i) / float64(n)
			t.Keys[i][h] = math.Sin(angle)
			if h%2 == 1 {
				t.Keys[i][h] = math.Cos(angle)
			}
			t.Attention[i] += w.K[h] * t.Keys[i][h] / math.Sqrt(compositionHidden)
		}
		maxScore = max(maxScore, t.Attention[i])
	}
	sum := 0.
	for i := range t.Attention {
		t.Attention[i] = math.Exp(t.Attention[i] - maxScore)
		sum += t.Attention[i]
	}
	for i := range t.Attention {
		t.Attention[i] /= sum
		for h, v := range t.Values[i] {
			t.Context[h] += t.Attention[i] * v
		}
	}
	for h, v := range t.Context {
		t.Activation[h] = v*(1+math.Tanh(w.Q[h])) + w.Q[h]
	}
	for o := range t.Z {
		for h, v := range t.Activation {
			t.Z[o] += w.H[o][h] * v
		}
	}
	return t
}

type sequenceSupervision struct {
	Focus  []float64
	Weight float64
}

func sequenceGradient(in CompositionInput, w sequenceWeights, dz [compositionOutputs]float64, supervision ...sequenceSupervision) sequenceWeights {
	t := sequenceForward(in, w)
	var g sequenceWeights
	var dc, da [compositionHidden]float64
	for o, d := range dz {
		g.B[o] = d
		for h, v := range t.Activation {
			g.H[o][h] = d * v
			da[h] += d * w.H[o][h]
		}
	}
	for h, v := range t.Context {
		q := math.Tanh(w.Q[h])
		dc[h] = da[h] * (1 + q)
		g.Q[h] += da[h] * (1 + v*(1-q*q))
	}
	dh := make([][compositionHidden]float64, len(t.States))
	attGrad := make([]float64, len(t.States))
	mean := 0.
	for i, s := range t.Values {
		for h, v := range s {
			attGrad[i] += dc[h] * v
		}
		mean += t.Attention[i] * attGrad[i]
	}
	for i := range t.States {
		dscore := t.Attention[i] * (attGrad[i] - mean)
		if len(supervision) > 0 && len(supervision[0].Focus) > 0 {
			dscore += supervision[0].Weight * (t.Attention[i] - supervision[0].Focus[i])
		}
		for h, v := range t.Keys[i] {
			g.K[h] += dscore * v / math.Sqrt(compositionHidden)
			dh[i][h] += t.Attention[i] * dc[h]
		}
	}
	for i := len(t.States) - 1; i >= 0; i-- {
		for h, v := range t.States[i] {
			d := dh[i][h] * (1 - v*v)
			g.HB[h] += d
			for j, x := range t.X[i] {
				g.W[h][j] += d * x
			}
			if i > 0 {
				for j, x := range t.States[i-1] {
					g.R[h][j] += d * x
					dh[i-1][j] += d * w.R[h][j]
				}
			}
		}
	}
	return g
}
func sequenceStep(w, g sequenceWeights, reliability float64) sequenceWeights {
	for h := 0; h < compositionHidden; h++ {
		w.Q[h] = compositionStep(w.Q[h], g.Q[h], reliability)
		w.K[h] = compositionStep(w.K[h], g.K[h], reliability)
		w.HB[h] = compositionStep(w.HB[h], g.HB[h], reliability)
		for j := range w.W[h] {
			w.W[h][j] = compositionStep(w.W[h][j], g.W[h][j], reliability)
		}
		for j := range w.R[h] {
			w.R[h][j] = compositionStep(w.R[h][j], g.R[h][j], reliability)
		}
	}
	for o := 0; o < compositionOutputs; o++ {
		w.B[o] = compositionStep(w.B[o], g.B[o], reliability)
		for h := 0; h < compositionHidden; h++ {
			w.H[o][h] = compositionStep(w.H[o][h], g.H[o][h], reliability)
		}
	}
	return w
}
func (k *KnowledgeBase) learnSequenceComposition(bank [compositionBanks][]NodeID, query []NodeID, e CompositionExposure, at time.Time) error {
	for h := 0; h < compositionHidden; h++ {
		if _, ok := k.compositionParameter(query, bank[12+h]); !ok {
			k.adjustCompositionParameter(query, bank[12+h], 0, 0, at, true)
		}
	}
	w := k.loadSequenceComposition(bank, query)
	target, state, force, err := e.target()
	if err != nil {
		return err
	}
	for micro := 0; micro < 5; micro++ {
		t := sequenceForward(e.Input, w)
		_, prob := compositionForward(e.Input, compositionWeights{B: t.Z})
		var dz [compositionOutputs]float64
		if force > 0 {
			for j, p := range e.Input.Scene.Patches {
				d := prob[j]
				if j == target {
					d--
				}
				for c := 0; c < 3; c++ {
					dz[c] += d * p.Values[c]
				}
				dz[3] += d * compositionPoint(e.Input.Scene, p)
			}
			dz[4] = compositionSigmoid(t.Z[4]) - (state+1)/2
		}
		dz[5] = compositionSigmoid(t.Z[5]) - force
		w = sequenceStep(w, sequenceGradient(e.Input, w, dz, sequenceSupervision{e.SequenceFocus, e.SequenceFocusWeight}), e.Reliability)
	}
	for h := 0; h < compositionHidden; h++ {
		k.adjustCompositionParameter(query, bank[h], w.Q[h], e.Reliability, at, false)
		k.adjustCompositionParameter(query, bank[12+h], w.K[h], e.Reliability, at, false)
		k.adjustCompositionParameter(bank[24], bank[h], w.HB[h], e.Reliability, at, false)
		for j, b := range sequenceInputBanks {
			k.adjustCompositionParameter(bank[b], bank[h], w.W[h][j], e.Reliability, at, false)
		}
		for j := 0; j < compositionHidden; j++ {
			k.adjustCompositionParameter(bank[j], bank[h], w.R[h][j], e.Reliability, at, false)
		}
	}
	for o := 0; o < compositionOutputs; o++ {
		k.adjustCompositionParameter(bank[24], bank[12+o], w.B[o], e.Reliability, at, false)
		for h := 0; h < compositionHidden; h++ {
			k.adjustCompositionParameter(bank[h], bank[12+o], w.H[o][h], e.Reliability, at, false)
		}
	}
	return nil
}

// Focus is teaching provenance only; ReadComposition never receives it.
func validateSequenceFocus(e CompositionExposure) error {
	if len(e.SequenceFocus) == 0 {
		if e.SequenceFocusWeight != 0 {
			return errors.New("focus weight lacks teacher distribution")
		}
		return nil
	}
	if len(e.SequenceFocus) != len(e.Input.Sequence) || math.IsNaN(e.SequenceFocusWeight) || e.SequenceFocusWeight <= 0 || e.SequenceFocusWeight > 1 {
		return errors.New("invalid sequence focus supervision")
	}
	sum := 0.
	for _, v := range e.SequenceFocus {
		if math.IsNaN(v) || v < 0 || v > 1 {
			return errors.New("invalid teacher focus mass")
		}
		sum += v
	}
	if math.Abs(sum-1) > 1e-9 {
		return errors.New("teacher focus must sum to one")
	}
	return nil
}
