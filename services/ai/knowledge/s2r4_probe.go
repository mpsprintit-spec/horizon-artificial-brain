//go:build horizon_s2r4

package knowledge

import (
	"errors"
	"math"
	"slices"
)

// Disposable diagnosis of the exact native fitting basis. No fitted object
// escapes into Brain state; actual learning still uses learnNMetric.
func (k *KnowledgeBase) S2R4FitAudit(inputs []CompositionInput, outcomes []float64, ids []string) (any, error) {
	k.RLock()
	defer k.RUnlock()
	if len(inputs) == 0 || len(inputs) != len(outcomes) || len(inputs) != len(ids) {
		return nil, errors.New("invalid audit evidence")
	}
	bank, e := k.compositionBankLocked()
	if e != nil {
		return nil, e
	}
	words, _ := k.compositionWordsLocked(inputs[0].Words)
	if len(words) != 1 {
		return nil, errors.New("unknown audit query")
	}
	q := words[0]
	queries := [][]NodeID{slices.Clone(q)}
	for _, h := range k.BrainState.NResolution.Heads {
		if !slices.ContainsFunc(queries, func(v []NodeID) bool { return slices.Equal(v, h.Query) }) {
			queries = append(queries, slices.Clone(h.Query))
		}
	}
	m := k.loadNModel(bank, queries)
	hi := nHeadIndex(m, q)
	if hi < 0 {
		return nil, errors.New("head absent")
	}
	fit := []NWitness{}
	for i, in := range inputs {
		fit = append(fit, NWitness{X: nInput(in), Query: slices.Clone(q), Outcome: outcomes[i], EpisodeID: ids[i]})
	}
	for _, w := range k.BrainState.NResolution.Witnesses {
		if slices.Equal(w.Query, q) && w.Role != "evaluation" {
			fit = append(fit, w)
		}
	}
	result := map[string]any{"model": m, "FitN": len(fit)}
	for _, mode := range []bool{false, true} {
		c := nClone(m)
		c.Heads[hi].Metric = mode
		d := nMetricFitDiagnostic{}
		valid := nFitMetricDiagnosed(&c, hi, fit, &d)
		name := "activation"
		if mode {
			name = "metric"
		}
		// Nonfinite diagnostic values cannot be serialized as JSON.
		if math.IsNaN(d.Value) || math.IsInf(d.Value, 0) {
			d.Value = 0
		}
		result[name] = map[string]any{"valid": valid, "diagnostic": d, "fit_error": nError(c, fit), "shared_change": nMetricSharedChange(m, c, q), "retention_regressions": nMetricRegression(m, c, k.BrainState.NResolution.Witnesses)}
	}
	return result, nil
}
