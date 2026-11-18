//go:build horizon_s2r6

package knowledge

// Research-only counterfactual workspaces. Parameters are always loaded from N;
// a detached intervention cannot be installed or used as a second predictor.
import (
	"errors"
	"math"
	"slices"
)

type S2R6Intervention struct {
	Ordinals []int
	Delta    []float64
	Clamp    bool
}

// Descriptor contains only pre-intervention measurements and declared deltas.
// It deliberately has no neuron ID, target, postprediction, gate decision or
// loss. Outcomes are measured by a separate call after the forecast is logged.
type S2R6Context struct {
	Measurements                         []float64
	BeforePrediction                     float64
	ActivationMagnitude, WeightMagnitude float64
	ParameterDelta                       []float64
	Eligibility                          []float64
	FeatureNodes                         []NodeID
}

func (k *KnowledgeBase) s2r6Model(in CompositionInput) (nModel, []NodeID, error) {
	bank, err := k.compositionBankLocked()
	if err != nil {
		return nModel{}, nil, err
	}
	q, _ := k.compositionWordsLocked(in.Words)
	if len(q) != 1 || len(q[0]) == 0 {
		return nModel{}, nil, errors.New("unknown causal query")
	}
	m := k.loadNModel(bank, q)
	if !m.Heads[0].Known {
		return m, nil, errors.New("unknown causal head")
	}
	return m, q[0], nil
}

func (k *KnowledgeBase) S2R6Describe(in CompositionInput, change S2R6Intervention) (S2R6Context, error) {
	k.RLock()
	defer k.RUnlock()
	return k.s2r6DescribeLocked(in, change)
}

func (k *KnowledgeBase) s2r6DescribeLocked(in CompositionInput, change S2R6Intervention) (S2R6Context, error) {
	out := S2R6Context{}
	m, q, err := k.s2r6Model(in)
	if err != nil {
		return out, err
	}
	if len(change.Ordinals) == 0 || len(change.Ordinals) != len(change.Delta) || len(change.Ordinals) > 32 {
		return out, errors.New("invalid intervention")
	}
	x := nInput(in)
	ev := nMetricEvaluate(m, x, nil)
	if !ev.Valid {
		return out, errors.New("invalid basis")
	}
	v := ev.Activation
	if m.Heads[0].Metric {
		v = ev.Metric
	}
	out.BeforePrediction = nPredict(m, x, q)
	// Bounded summaries are structural measurements, not a utility formula.
	out.Measurements = make([]float64, 18)
	out.Measurements[0] = out.BeforePrediction
	out.Measurements[7] = float64(len(change.Ordinals)) / 32
	for a, i := range change.Ordinals {
		if i < 0 || i >= len(m.Circuits) || !nFinite(change.Delta[a]) {
			return out, errors.New("invalid intervention index/delta")
		}
		c := m.Circuits[i]
		w := m.Heads[0].W[i]
		d := change.Delta[a]
		if change.Clamp {
			d = -w
		}
		if math.Abs(w+d) > 7.99 {
			return out, errors.New("signed bound")
		}
		el := 0.
		for _, target := range q {
			for _, syn := range k.Registry.GetByID(c.Meta.Node).Synapses[target] {
				if syn != nil {
					el += syn.Dynamic.Eligibility / float64(len(q))
				}
			}
		}
		out.FeatureNodes = append(out.FeatureNodes, c.Meta.Node)
		out.Eligibility = append(out.Eligibility, el)
		out.ParameterDelta = append(out.ParameterDelta, d)
		n := float64(len(change.Ordinals))
		out.Measurements[1] += math.Tanh(v[i]) / n
		out.Measurements[2] += math.Tanh(w) / n
		out.Measurements[3] += math.Tanh(d) / n
		out.Measurements[4] += ev.Activation[i] / n
		out.Measurements[5] += el / n
		out.Measurements[6] += float64(len(c.Meta.Parents)) / 2 / n
		out.ActivationMagnitude += math.Abs(ev.Activation[i]) / n
		out.WeightMagnitude += math.Abs(w) / n
		for _, other := range m.Circuits {
			if slices.Contains(other.Meta.Parents, c.Meta.Node) {
				out.Measurements[8] += 1 / 32. / n
			}
		}
	}
	// Raw observed context is summarized without the actual prediction error.
	for i := 0; i < 9; i++ {
		out.Measurements[9+i] = x[i]
	}
	return out, nil
}

func (k *KnowledgeBase) S2R6Counterfactual(in CompositionInput, change S2R6Intervention) (float64, float64, error) {
	k.RLock()
	defer k.RUnlock()
	m, q, err := k.s2r6Model(in)
	if err != nil {
		return 0, 0, err
	}
	if len(change.Ordinals) == 0 || len(change.Ordinals) != len(change.Delta) {
		return 0, 0, errors.New("invalid intervention")
	}
	x := nInput(in)
	before := nPredict(m, x, q)
	for a, i := range change.Ordinals {
		if i < 0 || i >= len(m.Circuits) {
			return 0, 0, errors.New("invalid ordinal")
		}
		d := change.Delta[a]
		if change.Clamp {
			d = -m.Heads[0].W[i]
		}
		if !nFinite(d) || math.Abs(m.Heads[0].W[i]+d) > 7.99 {
			return 0, 0, errors.New("signed bound")
		}
		m.Heads[0].W[i] += d
	}
	return before, nPredict(m, x, q), nil
}
