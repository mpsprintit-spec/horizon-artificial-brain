//go:build horizon_s2r2

package knowledge

import "errors"

// Detached experimental clamp. It never writes synapses, witnesses or state.
// A clamped graph is intentionally NOT a valid persistable unit-axis model.
func (k *KnowledgeBase) S2R2Probe(in CompositionInput, after NodeID, mode string) (float64, any, error) {
	k.RLock()
	defer k.RUnlock()
	bank, err := k.compositionBankLocked()
	if err != nil {
		return 0, nil, err
	}
	q, _ := k.compositionWordsLocked(in.Words)
	if len(q) != 1 {
		return 0, nil, errors.New("probe needs one known query")
	}
	m := k.loadNModel(bank, q)
	hi := nHeadIndex(m, q[0])
	if hi < 0 {
		return 0, nil, errors.New("probe query not found")
	}
	if mode == "all-readout-clamp" {
		m.Heads[hi].B = 0
		for i := range m.Heads[hi].W {
			m.Heads[hi].W[i] = 0
		}
	}
	for i, c := range m.Circuits {
		if c.Meta.Node <= after {
			continue
		}
		switch mode {
		case "feature-clamp":
			m.Circuits[i] = nCircuit{Meta: NFeature{Node: c.Meta.Node, Kind: 1}, Axis: make([]float64, nDimensions), Delta: 1}
		case "readout-clamp":
			m.Heads[hi].W[i] = 0
		}
	}
	return nPredict(m, nInput(in), q[0]), m, nil
}
