package knowledge

import (
	"math"
	"math/rand"
	"slices"
	"sort"
)

// Transitive exclusion is an explicit causal intervention on the proposed
// query readout. It never deletes a shared circuit or changes another query.
func nMetricExcluded(m nModel, p *NMetricProposalConfig) map[NodeID]bool {
	out := map[NodeID]bool{}
	for _, id := range p.ExcludedFeatures {
		out[id] = true
	}
	for _, c := range m.Circuits {
		for _, id := range c.Meta.Parents {
			if out[id] {
				out[c.Meta.Node] = true
			}
		}
	}
	return out
}

func nMetricAdaptiveSteps(before, basis nModel, query []NodeID, fit []NWitness, p *NMetricProposalConfig) []nMetricProposal {
	// Evaluate the same dyadic parameters that the signed population writer
	// can reconstruct exactly. Never quantize an unchanged shared circuit.
	for i := range basis.Circuits {
		if i >= len(before.Circuits) || !slices.Equal(basis.Circuits[i].Axis, before.Circuits[i].Axis) || basis.Circuits[i].Delta != before.Circuits[i].Delta || basis.Circuits[i].Center != before.Circuits[i].Center {
			basis.Circuits[i].Axis = slices.Clone(basis.Circuits[i].Axis)
			for j, v := range basis.Circuits[i].Axis {
				basis.Circuits[i].Axis[j] = nConstraintCoefficient(v)
			}
			basis.Circuits[i].Center = nConstraintCoefficient(basis.Circuits[i].Center)
			basis.Circuits[i].Delta = math.Max(math.Ldexp(1, -40), nConstraintCoefficient(basis.Circuits[i].Delta))
		}
	}
	basis.evaluationCache = make(map[nMetricEvaluationKey]nMetricEvaluation)
	excluded := nMetricExcluded(basis, p)
	full := nMetricRegularizedMaskedProposal(basis, query, fit, p.Ridge, excluded)
	steps := nMetricProposals(before, full.Model, query, fit, p, full.Valid)
	for i := range steps {
		hi := nHeadIndex(steps[i].Model, query)
		bi := nHeadIndex(before, query)
		if bi < 0 || steps[i].Model.Heads[hi].B != before.Heads[bi].B {
			steps[i].Model.Heads[hi].B = nConstraintCoefficient(steps[i].Model.Heads[hi].B)
		}
		for j, c := range steps[i].Model.Circuits {
			if bi < 0 || j >= len(before.Heads[bi].W) || steps[i].Model.Heads[hi].W[j] != before.Heads[bi].W[j] {
				steps[i].Model.Heads[hi].W[j] = nConstraintCoefficient(steps[i].Model.Heads[hi].W[j])
			}
			if excluded[c.Meta.Node] {
				steps[i].Model.Heads[hi].W[j] = 0
			}
		}
	}
	return steps
}

// Only structural proposal ranking sees the permutation. Both refitting and
// independent Check/admission always see the real observed outcomes.
func nMetricGrowthEvidence(best nModel, fit []NWitness, p *NMetricProposalConfig) []NWitness {
	if p == nil || p.Protocol != NMetricAdaptiveV1 || !p.PermuteGrowth {
		return fit
	}
	out := slices.Clone(fit)
	perm := rand.New(rand.NewSource(p.GrowthSeed)).Perm(len(fit))
	for i := range out {
		donor := fit[perm[i]]
		out[i].Outcome = nPredict(best, fit[i].X, fit[i].Query) + donor.Outcome - nPredict(best, donor.X, donor.Query)
	}
	return out
}

// Rank measured coordinates by residual association at linear and quadratic
// order. These are directions for the existing native projection units, not
// external features or task-specific answers. Product composition can then
// reuse them through the existing native relation mechanism.
func nMetricAdaptiveAxes(best nModel, query []NodeID, fit []NWitness) [][]float64 {
	type axisScore struct {
		index int
		score float64
	}
	ranked := []axisScore{}
	residuals := make([]float64, len(fit))
	rm := 0.
	for i, r := range fit {
		residuals[i] = r.Outcome - nPredict(best, r.X, query)
		rm += residuals[i] / float64(len(fit))
	}
	for j := 0; j < nDimensions; j++ {
		mean := 0.
		for _, r := range fit {
			mean += r.X[j] / float64(len(fit))
		}
		score := 0.
		for _, squared := range []bool{false, true} {
			values := make([]float64, len(fit))
			vm := 0.
			for i, r := range fit {
				v := r.X[j] - mean
				if squared {
					v *= v
				}
				values[i] = v
				vm += v / float64(len(fit))
			}
			cov, variance := 0., 0.
			for i := range fit {
				v := values[i] - vm
				cov += v * (residuals[i] - rm)
				variance += v * v
			}
			if variance > 0 {
				score = math.Max(score, math.Abs(cov)/math.Sqrt(variance))
			}
		}
		if score > 0 && nFinite(score) {
			ranked = append(ranked, axisScore{j, score})
		}
	}
	sort.SliceStable(ranked, func(i, j int) bool { return ranked[i].score > ranked[j].score })
	out := nMetricBlockAxes(best, query, fit, nil)
	for _, r := range ranked[:min(4, len(ranked))] {
		a := make([]float64, nDimensions)
		a[r.index] = 1
		out = append(out, a)
	}
	return out
}

// An activation-scaled quantizer is a second candidate, never a replacement
// for an inherited quantizer. It avoids forcing all novel distinctions into
// saturated tanh codes. Metric candidates retain their original precision.
func nMetricAdaptiveDelta(axis []float64, fit []NWitness) float64 {
	center := nCenter(axis, fit)
	variance := 0.
	for _, r := range fit {
		d := nDot(axis, r.X) - center
		variance += d * d / float64(len(fit))
	}
	return math.Min(7.99, math.Max(nMetricAxisFloor(axis, fit), math.Sqrt(variance)))
}
