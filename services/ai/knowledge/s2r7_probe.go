//go:build horizon_s2r7 && horizon_s2r6

package knowledge

// Read-only research instrumentation. Nothing here computes a utility label,
// ranks a pathway, or keeps a learned model outside the native substrate.
import (
	"errors"
	"math"
	"slices"
	"sort"
)

type S2R7Pathway struct {
	Value, Activation, Weight, Delta, Eligibility float64
	ParentValues, ParentActivations               []float64
	Fanout                                        int
}

type S2R7Context struct {
	Before   float64
	Pathways []S2R7Pathway
	Original S2R6Context
}

// Canonicalization sorts whole local records, never individual measurements.
// Ordinals and neural IDs are intentionally absent from the exported records.
func (k *KnowledgeBase) S2R7Describe(in CompositionInput, change S2R6Intervention) (S2R7Context, error) {
	k.RLock()
	defer k.RUnlock()
	original, err := k.s2r6DescribeLocked(in, change)
	if err != nil {
		return S2R7Context{}, err
	}
	m, q, err := k.s2r6Model(in)
	if err != nil {
		return S2R7Context{}, err
	}
	ev := nMetricEvaluate(m, nInput(in), nil)
	if !ev.Valid {
		return S2R7Context{}, errors.New("invalid measured basis")
	}
	v := ev.Activation
	if m.Heads[0].Metric {
		v = ev.Metric
	}
	out := S2R7Context{Before: original.BeforePrediction, Original: original}
	index := map[NodeID]int{}
	for i, c := range m.Circuits {
		index[c.Meta.Node] = i
	}
	for a, i := range change.Ordinals {
		c := m.Circuits[i]
		p := S2R7Pathway{Value: v[i], Activation: ev.Activation[i], Weight: m.Heads[0].W[i], Delta: original.ParameterDelta[a], Eligibility: original.Eligibility[a]}
		for _, id := range c.Meta.Parents {
			j := index[id]
			p.ParentValues = append(p.ParentValues, v[j])
			p.ParentActivations = append(p.ParentActivations, ev.Activation[j])
		}
		for _, other := range m.Circuits {
			if slices.Contains(other.Meta.Parents, c.Meta.Node) {
				p.Fanout++
			}
		}
		out.Pathways = append(out.Pathways, p)
	}
	_ = q
	SortS2R7Pathways(out.Pathways)
	return out, nil
}

func SortS2R7Pathways(p []S2R7Pathway) {
	sort.SliceStable(p, func(i, j int) bool {
		a, b := p[i], p[j]
		x := append(append([]float64{a.Value, a.Activation, a.Weight, a.Delta, a.Eligibility, float64(a.Fanout)}, a.ParentValues...), a.ParentActivations...)
		y := append(append([]float64{b.Value, b.Activation, b.Weight, b.Delta, b.Eligibility, float64(b.Fanout)}, b.ParentValues...), b.ParentActivations...)
		return slices.Compare(x, y) < 0
	})
}

type S2R7Representation struct {
	Input, Activation, Metric   []float64
	Weights                     []float64
	Bias                        float64
	MetricReadout, Known, Valid bool
	RefinableLeaves, Features   int
	ActiveFeatures              int
}

func (k *KnowledgeBase) S2R7Observe(in CompositionInput) (S2R7Representation, error) {
	k.RLock()
	defer k.RUnlock()
	bank, err := k.compositionBankLocked()
	if err != nil {
		return S2R7Representation{}, err
	}
	words, _ := k.compositionWordsLocked(in.Words)
	if len(words) != 1 {
		return S2R7Representation{}, errors.New("invalid observed query")
	}
	queries := [][]NodeID{slices.Clone(words[0])}
	for _, h := range k.BrainState.NResolution.Heads {
		if !slices.ContainsFunc(queries, func(q []NodeID) bool { return slices.Equal(q, h.Query) }) {
			queries = append(queries, slices.Clone(h.Query))
		}
	}
	m := k.loadNModel(bank, queries)
	hi := nHeadIndex(m, words[0])
	h := m.Heads[hi]
	x := nInput(in)
	ev := nMetricEvaluate(m, x, nil)
	o := S2R7Representation{Input: x, Activation: ev.Activation, Metric: ev.Metric, Weights: slices.Clone(h.W), Bias: h.B, MetricReadout: h.Metric, Known: h.Known, Valid: ev.Valid, Features: len(m.Circuits)}
	for i := range m.Circuits {
		if nMetricRefinableLeaf(m, i, words[0]) {
			o.RefinableLeaves++
		}
		if math.Abs(ev.Activation[i]) > 1e-12 {
			o.ActiveFeatures++
		}
	}
	return o, nil
}
