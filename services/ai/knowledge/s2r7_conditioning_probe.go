//go:build horizon_s2r7 && horizon_s2r6

package knowledge

// Controlled optimizer workspaces on a fresh developmental corpus. They are
// never installed and never used as Horizon's utility controller or forecast.
import (
	"errors"
	"slices"
)

func (k *KnowledgeBase) S2R7ConditioningProbe(inputs []CompositionInput, y []float64, checks []CompositionInput, cy []float64, tests []CompositionInput) (any, error) {
	if len(inputs) == 0 || len(inputs) != len(y) || len(checks) != len(cy) || len(checks) == 0 {
		return nil, errors.New("invalid independent conditioning evidence")
	}
	k.RLock()
	defer k.RUnlock()
	bank, err := k.compositionBankLocked()
	if err != nil {
		return nil, err
	}
	words, _ := k.compositionWordsLocked(inputs[0].Words)
	q := words[0]
	m := k.loadNModel(bank, [][]NodeID{q})
	hi := nHeadIndex(m, q)
	m.Heads[hi].Known = true
	m.Heads[hi].Metric = true
	rows := []NWitness{}
	check := []NWitness{}
	for i, in := range inputs {
		rows = append(rows, NWitness{X: nInput(in), Query: q, Outcome: y[i]})
	}
	for i, in := range checks {
		check = append(check, NWitness{X: nInput(in), Query: q, Outcome: cy[i]})
	}
	p := &NMetricProposalConfig{Update: "readout"}
	prior := nMetricConstrainedProposals(m, m, q, rows, nil, nil, p)[0].Model
	result := map[string]any{"metric_prior_readout": slices.Clone(prior.Heads[hi].W), "metric_prior_bias": prior.Heads[hi].B, "prior_fit_MSE": nError(prior, rows), "prior_Check_MSE": nError(prior, check), "fresh_development_only": true, "installed": false}
	zeroCheck := 0.
	for _, v := range cy {
		zeroCheck += v * v / float64(len(cy))
	}
	result["zero_Check_MSE"] = zeroCheck
	result["metric_prior_Check_improves_zero"] = nImproves(nError(prior, check), zeroCheck)
	for _, name := range []string{"metric-coefficients-warm-activation", "zero-initialized-activation", "standardized-ridge-activation"} {
		origin := nClone(prior)
		origin.Heads[hi].Metric = false
		if name != "metric-coefficients-warm-activation" {
			origin.Heads[hi].W = make([]float64, len(m.Circuits))
			origin.Heads[hi].B = 0
		}
		candidate := nMetricConstrainedProposals(origin, origin, q, rows, nil, nil, p)[0].Model
		if name == "standardized-ridge-activation" {
			candidate = nMetricRegularizedProposal(origin, q, rows, .01).Model
		}
		pred := []float64{}
		for _, in := range tests {
			pred = append(pred, nPredict(candidate, nInput(in), q))
		}
		result[name] = map[string]any{"Readout": candidate.Heads[hi].W, "Bias": candidate.Heads[hi].B, "Fit_MSE": nError(candidate, rows), "Check_MSE": nError(candidate, check), "Predictions": pred, "Check_better_than_metric_prior": nImproves(nError(candidate, check), nError(prior, check))}
	}
	return result, nil
}
