//go:build horizon_s2r3

package knowledge

import (
	"errors"
	"fmt"
)

// Read-only research view. Detached clamps are not persistable brain models.
type S2R3Audit struct {
	Model                                                        any
	Inputs, Activation, Code, Metric, MetricError, Contributions [][]float64
	Predictions                                                  map[string][]float64
}

func (k *KnowledgeBase) S2R3Inspect(inputs []CompositionInput, old []NodeID) (S2R3Audit, error) {
	k.RLock()
	defer k.RUnlock()
	out := S2R3Audit{Predictions: map[string][]float64{}}
	if len(inputs) == 0 {
		return out, errors.New("empty inspection")
	}
	bank, err := k.compositionBankLocked()
	if err != nil {
		return out, err
	}
	q, _ := k.compositionWordsLocked(inputs[0].Words)
	if len(q) != 1 {
		return out, errors.New("unknown query")
	}
	m := k.loadNModel(bank, q)
	hi := nHeadIndex(m, q[0])
	if hi < 0 || !m.Heads[hi].Known {
		return out, errors.New("unknown head")
	}
	out.Model = m
	oldSet := map[NodeID]bool{}
	for _, id := range old {
		oldSet[id] = true
	}
	for _, in := range inputs {
		x := nInput(in)
		e := nMetricEvaluate(m, x, nMetricInputError(in))
		if !e.Valid {
			return out, errors.New("invalid native evaluation")
		}
		v := e.Activation
		if m.Heads[hi].Metric {
			v = e.Metric
		}
		contribution := make([]float64, len(v))
		for i, w := range m.Heads[hi].W {
			contribution[i] = w * v[i]
		}
		out.Inputs = append(out.Inputs, x)
		out.Activation = append(out.Activation, e.Activation)
		out.Code = append(out.Code, e.Code)
		out.Metric = append(out.Metric, e.Metric)
		out.MetricError = append(out.MetricError, e.MetricError)
		out.Contributions = append(out.Contributions, contribution)
	}
	models := map[string]nModel{"full": m}
	for _, mode := range []string{"new-feature-zero", "new-readout-zero", "all-readout-zero"} {
		c := nClone(m)
		if mode == "all-readout-zero" {
			c.Heads[hi].B = 0
		}
		for i, f := range c.Circuits {
			if oldSet[f.Meta.Node] && mode != "all-readout-zero" {
				continue
			}
			if mode == "new-feature-zero" {
				c.Circuits[i] = nCircuit{Meta: NFeature{Node: f.Meta.Node, Kind: 1}, Axis: make([]float64, nDimensions), Delta: 1}
			} else {
				c.Heads[hi].W[i] = 0
			}
		}
		models[mode] = c
	}
	for i, f := range m.Circuits {
		c := nClone(m)
		c.Circuits[i] = nCircuit{Meta: NFeature{Node: f.Meta.Node, Kind: 1}, Axis: make([]float64, nDimensions), Delta: 1}
		models[fmt.Sprintf("feature-zero-%d", f.Meta.Node)] = c
		if !oldSet[f.Meta.Node] {
			c = nClone(m)
			c.Heads[hi].W[i] = 0
			models[fmt.Sprintf("readout-zero-%d", f.Meta.Node)] = c
		}
	}
	for name, model := range models {
		for _, x := range out.Inputs {
			out.Predictions[name] = append(out.Predictions[name], nPredict(model, x, q[0]))
		}
	}
	return out, nil
}
