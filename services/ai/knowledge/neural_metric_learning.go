package knowledge

import (
	"errors"
	"math"
	"slices"
	"sort"
	"time"
)

func (k *KnowledgeBase) learnNMetric(bank [compositionBanks][]NodeID, query []NodeID, e CompositionExposure, at time.Time) error {
	if e.Resolution == nil {
		return errors.New("v6 requires fit/check numerical evidence")
	}
	if k.BrainState.NResolution == nil {
		k.BrainState.NResolution = &NResolutionState{Contract: NeuralCompositionMetricV6, Sources: make([]NodeID, nDimensions)}
	}
	state := k.BrainState.NResolution
	if state.Contract != NeuralCompositionMetricV6 {
		return errors.New("trained numerical contract cannot change implicitly")
	}
	queries := [][]NodeID{slices.Clone(query)}
	for _, h := range state.Heads {
		if !slices.ContainsFunc(queries, func(q []NodeID) bool { return slices.Equal(q, h.Query) }) {
			queries = append(queries, slices.Clone(h.Query))
		}
	}
	before := k.loadNModel(bank, queries)
	convert := func(rows []NObservation) []NWitness {
		out := []NWitness{}
		for _, o := range rows {
			out = append(out, NWitness{OutcomeError: o.OutcomeError, MeasurementError: nMetricInputError(o.Input), X: nInput(o.Input), Query: slices.Clone(query), Outcome: o.Outcome, EpisodeID: o.EpisodeID})
		}
		return out
	}
	fit, check := convert(e.Resolution.Fit), convert(e.Resolution.Check)
	for _, w := range state.Witnesses {
		if slices.Equal(w.Query, query) {
			fit = append(fit, w)
		}
	}
	all := append(slices.Clone(fit), check...)
	d := NDiagnosis{Query: slices.Clone(query), Cause: "insufficient-predictive-evidence", Stage: "validation", BeforeError: nError(before, check)}
	d.AfterError = d.BeforeError
	d.AliasPairs = nMetricAliasPairs(before, query, check)
	if nMetricConflict(all) {
		d.Cause = "conflicting-observations"
		d.Stage = "evidence"
		d.Hypotheses = []string{"missing-context", "changed-dynamics", "label-noise"}
		d.Probes = []NProbeRequest{{Kind: "context-contrast", Evidence: "observe differing conditions for identical encoded measurements"}, {Kind: "repeat-observation", Evidence: "repeat the observation with measurement uncertainty and outcome provenance"}}
		if query[0] != 0 {
			rmse := math.Sqrt(nError(before, all))
			maximum := nMetricMaximumError(before, query, all)
			k.writeNParameter(bank[26], query, rmse, e.Reliability, at)
			if len(state.Controls) > 0 {
				k.writeNParameter(state.Controls[:1], query, maximum, e.Reliability, at)
			}
		}
		k.finishNMetricDiagnosis(d, e.Input.RequiredPrecision, nMetricMaximumError(before, query, all), at)
		return nil
	}
	best := nClone(before)
	bestErr := d.BeforeError
	bestKnown := before.Heads[nHeadIndex(before, query)].Known
	bestBound := nMetricQuality(before, query, check)
	selected := false
	goal := 0.
	binary := true
	for _, row := range all {
		binary = binary && math.Abs(row.Outcome) == 1
	}
	if e.Input.RequiredPrecision != nil {
		goal = *e.Input.RequiredPrecision
	}
	consider := func(kind string, feature NodeID, candidate nModel) {
		for _, mode := range []bool{false, true} {
			m := nClone(candidate)
			hi := nHeadIndex(m, query)
			m.Heads[hi].Metric = mode
			valid := nFit(&m, query, fit)
			for _, row := range all {
				if !nMetricEvaluate(m, row.X, row.MeasurementError).Valid {
					valid = false
					break
				}
			}
			err := nError(m, check)
			if !nFinite(err) {
				valid = false
				err = 4
			}
			reg := nMetricRegression(before, m, state.Witnesses)
			duplicate := len(m.Circuits) > len(before.Circuits) && nMetricRedundant(m, all, mode)
			why := []string{}
			if binary && goal == 0 && mode {
				why = append(why, "metric-goal-unspecified")
			}
			if !valid {
				why = append(why, "fit-invalid-or-parameter-bound")
			}
			if duplicate {
				why = append(why, "functionally-duplicate")
			}
			if reg > 0 {
				why = append(why, "retention-regression")
			}
			knownGain := !bestKnown && valid && err == 0
			bound := nMetricQuality(m, query, check)
			precisionGain := goal > 0 && bestBound > goal && nImproves(bound, bestBound) && err <= math.Max(bestErr, goal*goal)+nRoundoff(math.Max(bestErr, goal*goal))
			errorGain := nImproves(err, bestErr)
			if goal > 0 && bestErr <= goal*goal/4 {
				errorGain = false
			}
			if goal == 0 && binary && bestKnown && bestErr <= nTolerance*nTolerance {
				errorGain = false
			}
			gain := errorGain || knownGain || precisionGain
			if binary && goal == 0 && mode && bestKnown && bestErr-err <= nTolerance {
				gain = false
			}
			if !gain {
				why = append(why, "no-check-improvement")
			}
			if len(m.Circuits) > len(before.Circuits) && bestKnown && ((goal > 0 && nMetricPointError(best, query, check) <= goal) || (goal == 0 && binary && nBinarySolved(best, query, check))) {
				why = append(why, "existing-knowledge-meets-precision")
			}
			eligible := len(why) == 0
			modeName := "activation"
			if mode {
				modeName = "metric"
			}
			v := valid
			d.Trials = append(d.Trials, NTrial{Kind: kind, Feature: feature, CheckError: err, RetentionRegressions: reg, Eligible: eligible, ReadoutMode: modeName, FitValid: &v, RejectedBy: why})
			if eligible {
				best = m
				bestErr = err
				bestBound = bound
				bestKnown = true
				selected = true
				d.Cause = kind
				d.Stage = modeName + "-readout"
			}
		}
	}
	consider("readout", 0, nClone(before))
	for i, c := range before.Circuits {
		raw, floor := nMetricFeatureRaw(before, i, fit)
		delta := nMetricDelta(raw, fit)
		if goal > 0 && nMetricQuality(best, query, check) > goal {
			delta = math.Min(delta, math.Max(floor, c.Delta/16))
		}
		if delta < c.Delta/2 {
			m := nClone(before)
			m.Circuits[i].Delta = delta
			consider("resolution", c.Meta.Node, m)
		}
	}
	if len(before.Circuits) < maxNFeatures {
		axis, mean := make([]float64, nDimensions), make([]float64, nDimensions)
		for j := range mean {
			base := fit[0].X[j]
			deltas := 0.
			for _, r := range fit {
				deltas += (r.X[j] - base) / float64(len(fit))
			}
			mean[j] = base + deltas
		}
		maxResidual := 0.
		for _, r := range fit {
			maxResidual = math.Max(maxResidual, math.Abs(r.Outcome-nPredict(best, r.X, r.Query)))
		}
		if maxResidual > 0 {
			for _, r := range fit {
				residual := (r.Outcome - nPredict(best, r.X, r.Query)) / maxResidual
				for j, x := range r.X {
					axis[j] += residual * (x - mean[j])
				}
			}
		}
		norm := math.Sqrt(nDot(axis, axis))
		if nFinite(norm) && norm > math.SmallestNonzeroFloat64 {
			for j := range axis {
				axis[j] /= norm
			}
			m := nClone(before)
			m.Circuits = append(m.Circuits, nCircuit{Meta: NFeature{Kind: 1}, Axis: axis, Center: nCenter(axis, fit), Delta: nMetricAxisDelta(axis, fit)})
			for i := range m.Heads {
				m.Heads[i].W = append(m.Heads[i].W, 0)
			}
			kind := "missing-direction"
			for _, old := range before.Circuits {
				if old.Meta.Kind == 1 && math.Abs(math.Abs(nDot(old.Axis, axis))-1) < 1e-8 {
					kind = "missing-boundary"
				}
			}
			consider(kind, 0, m)
		}
		type product struct {
			a, b, kind int
			score      float64
		}
		candidates := []product{}
		acts, metrics := make([][]float64, len(fit)), make([][]float64, len(fit))
		residual := make([]float64, len(fit))
		for i, r := range fit {
			v := nMetricEvaluate(before, r.X, nil)
			acts[i], metrics[i] = v.Activation, v.Metric
			residual[i] = r.Outcome - nPredict(best, r.X, r.Query)
		}
		for _, kind := range []int{2, 3} {
			values := acts
			if kind == 3 {
				values = metrics
			}
			means := make([]float64, len(before.Circuits))
			varying := make([]bool, len(before.Circuits))
			for i := range means {
				lo, hi := values[0][i], values[0][i]
				for _, v := range values {
					means[i] += v[i] / float64(len(values))
					lo = math.Min(lo, v[i])
					hi = math.Max(hi, v[i])
				}
				varying[i] = hi-lo > nRoundoff(math.Max(math.Abs(lo), math.Abs(hi)))
			}
			residualMean := 0.
			for _, v := range residual {
				residualMean += v / float64(len(residual))
			}
			for i := range before.Circuits {
				for j := i; j < len(before.Circuits); j++ {
					// Multiplying a varying feature by an observed constant adds no
					// identifiable distinction beyond its existing scale and bias.
					if !varying[i] || !varying[j] {
						continue
					}
					parents := []NodeID{before.Circuits[i].Meta.Node, before.Circuits[j].Meta.Node}
					if slices.ContainsFunc(before.Circuits, func(c nCircuit) bool { return c.Meta.Kind == kind && slices.Equal(c.Meta.Parents, parents) }) {
						continue
					}
					score := 0.
					for _, centerParents := range []bool{false, true} {
						products := make([]float64, len(values))
						meanProduct := 0.
						for row, v := range values {
							value := v[i] * v[j]
							if centerParents {
								value = (v[i] - means[i]) * (v[j] - means[j])
							}
							products[row] = value
							meanProduct += value / float64(len(values))
						}
						covariance, variance := 0., 0.
						for row, v := range products {
							delta := v - meanProduct
							covariance += (residual[row] - residualMean) * delta
							variance += delta * delta
						}
						if variance > 0 {
							score = math.Max(score, math.Abs(covariance)/math.Sqrt(variance))
						}
					}
					// Centered and scale-normalized scores expose zero-context gates.
					// Only ranking changes; candidates remain canonical raw products.
					candidates = append(candidates, product{i, j, kind, score})
				}
			}

		}
		sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].score > candidates[j].score })
		for _, p := range candidates[:min(8, len(candidates))] {
			if p.score == 0 {
				continue
			}
			values := acts
			if p.kind == 3 {
				values = metrics
			}
			raw := make([]float64, len(fit))
			for i, v := range values {
				raw[i] = v[p.a] * v[p.b]
			}
			// Product centers are descriptive quantizers, not extra answer functions.
			center := 0.
			for _, x := range raw {
				center += x / float64(len(raw))
			}
			if math.Abs(center) > 7.99 {
				continue
			}
			m := nClone(before)
			m.Circuits = append(m.Circuits, nCircuit{Meta: NFeature{Kind: p.kind, Parents: []NodeID{before.Circuits[p.a].Meta.Node, before.Circuits[p.b].Meta.Node}}, Delta: nMetricDelta(raw, fit), Center: center})
			for i := range m.Heads {
				m.Heads[i].W = append(m.Heads[i].W, 0)
			}
			consider("missing-relation", 0, m)
		}
	}
	if selected {
		d.Committed = true
		d.AfterError = bestErr
		if query[0] == 0 {
			old := slices.Clone(query)
			query = k.newNQuery(e.Input.Words[0], at)
			best.Heads[nHeadIndex(best, old)].Query = slices.Clone(query)
			for i := range check {
				check[i].Query = slices.Clone(query)
			}
			d.Query = slices.Clone(query)
		}
		if len(best.Circuits) > len(before.Circuits) {
			i := len(best.Circuits) - 1
			c := &best.Circuits[i]
			c.Meta.Node = k.newNNode(2, i, at)
			d.Grown = c.Meta.Node
			if c.Meta.Kind == 1 {
				for j, w := range c.Axis {
					if w != 0 {
						if state.Sources[j] == 0 {
							state.Sources[j] = k.newNNode(1, j, at)
						}
						k.writeNParameter([]NodeID{state.Sources[j]}, []NodeID{c.Meta.Node}, w, e.Reliability, at)
					}
				}
			} else {
				for _, parent := range c.Meta.Parents {
					k.writeNParameter([]NodeID{parent}, []NodeID{c.Meta.Node}, 1, e.Reliability, at)
				}
			}
			state.Features = append(state.Features, c.Meta)
		}
		for i, c := range best.Circuits {
			if i >= len(before.Circuits) || c.Delta != before.Circuits[i].Delta {
				if i < len(before.Circuits) {
					d.ResolutionBefore = before.Circuits[i].Delta
					d.ResolutionAfter = c.Delta
				}
				k.writeNParameter(bank[25], []NodeID{c.Meta.Node}, c.Delta, e.Reliability, at)
				k.writeNParameter(bank[24], []NodeID{c.Meta.Node}, c.Center, e.Reliability, at)
			}
		}
		head := best.Heads[nHeadIndex(best, query)]
		support := slices.Clone(check)
		for _, w := range state.Witnesses {
			if slices.Equal(w.Query, query) {
				support = append(support, w)
			}
		}
		maximum := nMetricMaximumError(best, query, support)
		if len(state.Controls) == 0 {
			state.Controls = []NodeID{k.newNNode(3, 0, at)}
		}
		k.writeNParameter(bank[24], query, head.B, e.Reliability, at)
		k.writeNParameter(bank[26], query, math.Sqrt(bestErr), e.Reliability, at)
		mode := 0.
		if head.Metric {
			mode = 1
		}
		k.writeNParameter(bank[27], query, mode, e.Reliability, at)
		k.writeNParameter(state.Controls[:1], query, maximum, e.Reliability, at)
		for i, c := range best.Circuits {
			k.writeNParameter([]NodeID{c.Meta.Node}, query, head.W[i], e.Reliability, at)
		}
		topo := NHeadTopology{Query: slices.Clone(query)}
		for _, c := range best.Circuits {
			topo.Features = append(topo.Features, c.Meta.Node)
		}
		found := false
		for i, h := range state.Heads {
			if slices.Equal(h.Query, query) {
				state.Heads[i] = topo
				found = true
			}
		}
		if !found {
			state.Heads = append(state.Heads, topo)
		}
		state.Witnesses = append(state.Witnesses, check...)
		if len(state.Witnesses) > maxNWitnesses {
			state.Witnesses = nMetricWitnesses(state.Witnesses, state.Heads, maxNWitnesses, best)
		}
	} else if goal > 0 && nMetricMaximumError(before, query, check) <= goal && before.Heads[nHeadIndex(before, query)].Known {
		d.Cause = "already-within-precision"
		d.Stage = "validated"
	} else {
		d.Hypotheses = []string{"missing-measurement", "insufficient-composition", "changed-dynamics"}
		d.Probes = []NProbeRequest{{Kind: "context-contrast", Evidence: "compare observed outcomes while varying an independently measured context"}, {Kind: "transition-observation", Evidence: "supply the measured after-state and known action, not a repeated prediction"}}
	}
	k.finishNMetricDiagnosis(d, e.Input.RequiredPrecision, nMetricQuality(best, query, check), at)
	return nil
}
func nMetricAxisFloor(axis []float64, rows []NWitness) float64 {
	scale := 0.
	for _, r := range rows {
		scale = math.Max(scale, math.Abs(nDot(axis, r.X)))
	}
	return math.Max(math.SmallestNonzeroFloat64, nRoundoff(scale))
}
