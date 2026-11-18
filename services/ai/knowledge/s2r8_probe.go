//go:build horizon_s2r8

package knowledge

import "errors"

// Detached evidence, never a stored parallel model or prediction override.
func (k *KnowledgeBase) S2R8Inspect(inputs []CompositionInput, old []NodeID) (map[string]any, error) {
	k.RLock()
	defer k.RUnlock()
	if len(inputs) == 0 {
		return nil, errors.New("empty inspection")
	}
	bank, err := k.compositionBankLocked()
	if err != nil {
		return nil, err
	}
	q, _ := k.compositionWordsLocked(inputs[0].Words)
	m := k.loadNModel(bank, q)
	hi := nHeadIndex(m, q[0])
	if hi < 0 {
		return nil, errors.New("unknown query")
	}
	excluded := nMetricExcluded(m, &NMetricProposalConfig{ExcludedFeatures: old})
	out := map[string]any{"model": m}
	raw, act, metric := [][]float64{}, [][]float64{}, [][]float64{}
	pred := map[string][]float64{"full": {}, "new-readout-off": {}, "old-ancestry-readout-off": {}}
	for _, in := range inputs {
		x := nInput(in)
		ev := nMetricEvaluate(m, x, nil)
		raw = append(raw, x)
		act = append(act, ev.Activation)
		metric = append(metric, ev.Metric)
		for _, name := range []string{"full", "new-readout-off", "old-ancestry-readout-off"} {
			c := nClone(m)
			for j, f := range c.Circuits {
				isOld := false
				for _, id := range old {
					isOld = isOld || f.Meta.Node == id
				}
				if (name == "new-readout-off" && !isOld) || (name == "old-ancestry-readout-off" && excluded[f.Meta.Node]) {
					c.Heads[hi].W[j] = 0
				}
			}
			pred[name] = append(pred[name], nPredict(c, x, q[0]))
		}
	}
	out["raw"] = raw
	out["activation"] = act
	out["metric"] = metric
	out["predictions"] = pred
	return out, nil
}
