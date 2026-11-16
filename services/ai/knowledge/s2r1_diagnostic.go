//go:build horizon_s2r1

package knowledge

// Research-only accessors. Models are detached copies of canonical signed
// synapses. No accessor installs a parameter, records a lesson, or writes WAL.
import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"slices"
	"time"
)

type S2R1Trace struct {
	Before, Best          any
	BestBound             any
	Diagnosis             NDiagnosis
	Evidence              map[string]any
	Trials                []map[string]any
	Ranking               []map[string]any
	ResidualAxis          []float64
	before, best          nModel
	models                []nModel
	fit, check, witnesses []NWitness
}

func s2r1Number(x float64) any {
	if math.IsNaN(x) {
		return "NaN"
	}
	if math.IsInf(x, 1) {
		return "+Inf"
	}
	if math.IsInf(x, -1) {
		return "-Inf"
	}
	return x
}

func s2r1Hash(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func s2r1Evidence(fit, check, witness []NWitness) map[string]any {
	shared, fitIDs, checkIDs := 0, map[string]int{}, map[string]int{}
	for _, w := range fit {
		fitIDs[w.EpisodeID]++
	}
	for _, w := range check {
		checkIDs[w.EpisodeID]++
		if fitIDs[w.EpisodeID] > 0 {
			shared++
		}
	}
	coords := []map[string]any{}
	for j := 0; j < nDimensions; j++ {
		lo, hi, mean := math.Inf(1), math.Inf(-1), 0.
		for _, w := range fit {
			lo = math.Min(lo, w.X[j])
			hi = math.Max(hi, w.X[j])
			mean += w.X[j] / float64(len(fit))
		}
		variance := 0.
		for _, w := range fit {
			variance += (w.X[j] - mean) * (w.X[j] - mean) / float64(len(fit))
		}
		coords = append(coords, map[string]any{"coordinate": j, "min": lo, "max": hi, "mean": mean, "sd": math.Sqrt(variance)})
	}
	return map[string]any{"fit_n": len(fit), "check_n": len(check), "witness_n": len(witness), "fit_unique_episode_n": len(fitIDs), "check_overlap_fit_n": shared, "fit_sha256": s2r1Hash(fit), "check_sha256": s2r1Hash(check), "witness_sha256": s2r1Hash(witness), "coordinates": coords}
}

func s2r1Retention(before, after nModel, rows []NWitness) []map[string]any {
	type group struct {
		query       []NodeID
		a, b, limit float64
		n           int
		flips       int
	}
	groups := []group{}
	for _, r := range rows {
		i := -1
		for j, g := range groups {
			if slices.Equal(g.query, r.Query) {
				i = j
				break
			}
		}
		if i < 0 {
			groups = append(groups, group{query: slices.Clone(r.Query)})
			i = len(groups) - 1
		}
		av, bv := nPredict(before, r.X, r.Query), nPredict(after, r.X, r.Query)
		hi := nHeadIndex(before, r.Query)
		numerical := nRoundoff(av)
		if hi >= 0 {
			e := nMetricEvaluate(before, r.X, r.MeasurementError)
			v := e.ActivationError
			if before.Heads[hi].Metric {
				v = e.MetricError
			}
			for j, w := range before.Heads[hi].W {
				numerical += math.Abs(w) * v[j]
			}
		}
		limit := math.Abs(av-r.Outcome) + numerical
		g := &groups[i]
		g.a += (av - r.Outcome) * (av - r.Outcome)
		g.b += (bv - r.Outcome) * (bv - r.Outcome)
		g.limit += limit * limit
		g.n++
		if hi >= 0 && math.Abs(r.Outcome) == 1 && av*r.Outcome > 0 && math.Abs(av) > before.Heads[hi].Error+nTolerance && bv*r.Outcome <= 0 {
			g.flips++
		}
	}
	out := []map[string]any{}
	for _, g := range groups {
		n := float64(g.n)
		out = append(out, map[string]any{"query": g.query, "n": g.n, "before_mse": g.a / n, "candidate_mse": g.b / n, "allowed_mse": g.limit / n, "regression": nImproves(g.limit/n, g.b/n), "sign_flips": g.flips})
	}
	return out
}

func s2r1Parameters(before, after nModel, q []NodeID) []map[string]any {
	out := []map[string]any{}
	bi, ai := nHeadIndex(before, q), nHeadIndex(after, q)
	for i, c := range after.Circuits {
		if i >= len(before.Circuits) {
			out = append(out, map[string]any{"kind": "proposed-feature", "feature": c.Meta, "axis": c.Axis, "delta": c.Delta, "center": c.Center})
		} else if c.Delta != before.Circuits[i].Delta {
			out = append(out, map[string]any{"kind": "resolution", "feature": c.Meta.Node, "before": before.Circuits[i].Delta, "after": c.Delta})
		}
		if ai >= 0 {
			a := after.Heads[ai].W[i]
			b := 0.
			if bi >= 0 && i < len(before.Heads[bi].W) {
				b = before.Heads[bi].W[i]
			}
			if a != b {
				out = append(out, map[string]any{"kind": "readout-signed-parameter", "feature": c.Meta.Node, "query": q, "before": b, "after": a, "exc_dynamic_weight": math.Max(0, a) / compositionScale, "inh_dynamic_weight": math.Max(0, -a) / compositionScale})
			}
		}
	}
	if ai >= 0 {
		b := 0.
		if bi >= 0 {
			b = before.Heads[bi].B
		}
		out = append(out, map[string]any{"kind": "bias", "before": b, "after": after.Heads[ai].B})
	}
	return out
}

func (tr *S2R1Trace) capture(kind string, feature NodeID, before, best, candidate nModel, fit, check, witness []NWitness, bestErr, bestBound, goal float64, why []string, valid, duplicate, known, precision, gain bool) {
	q := check[0].Query
	tr.models = append(tr.models, nClone(candidate))
	tr.Trials = append(tr.Trials, map[string]any{"index": len(tr.Trials), "kind": kind, "feature": feature, "metric": candidate.Heads[nHeadIndex(candidate, q)].Metric, "fit_valid": valid, "duplicate": duplicate, "rejected_by": slices.Clone(why), "eligible": len(why) == 0, "fit_mse": nError(candidate, fit), "check_mse": nError(candidate, check), "before_fit_mse": nError(before, fit), "before_check_mse": nError(before, check), "best_check_mse_at_decision": bestErr, "best_quality_at_decision": s2r1Number(bestBound), "candidate_quality": s2r1Number(nMetricQuality(candidate, q, check)), "goal": goal, "known_gain": known, "precision_gain": precision, "error_gain": gain, "roundoff_margin": nRoundoff(math.Max(math.Abs(bestErr), math.Abs(nError(candidate, check)))), "regressions": nMetricRegression(before, candidate, witness), "retention_by_query": s2r1Retention(before, candidate, witness), "parameters": s2r1Parameters(before, candidate, q), "candidate_model_sha256": s2r1Hash(candidate)})
}

func (k *KnowledgeBase) S2R1Diagnose(c CompositionConfig, e CompositionExposure, at time.Time) (*S2R1Trace, error) {
	if err := k.ValidateCompositionExposure(c, e); err != nil {
		return nil, err
	}
	k.RLock()
	defer k.RUnlock()
	if k.BrainState.NResolution == nil {
		return nil, errors.New("requires restored S2 lineage")
	}
	bank, err := k.compositionBankLocked()
	if err != nil {
		return nil, err
	}
	ids, _ := k.compositionWordsLocked(e.Input.Words)
	q := ids[0]
	if len(q) == 0 {
		q = []NodeID{0}
	}
	tr := &S2R1Trace{}
	if err = k.s2r1TraceMetric(bank, q, e, at, tr); err != nil {
		return nil, err
	}
	return tr, nil
}

func (k *KnowledgeBase) s2r1Model() (nModel, error) {
	bank, err := k.compositionBankLocked()
	if err != nil {
		return nModel{}, err
	}
	queries := [][]NodeID{}
	for _, h := range k.BrainState.NResolution.Heads {
		queries = append(queries, h.Query)
	}
	return k.loadNModel(bank, queries), nil
}

func (k *KnowledgeBase) S2R1Inspect(in CompositionInput) (map[string]any, error) {
	k.RLock()
	defer k.RUnlock()
	m, err := k.s2r1Model()
	if err != nil {
		return nil, err
	}
	m = nClone(m)
	ids, _ := k.compositionWordsLocked(in.Words)
	q := ids[0]
	if len(q) == 0 {
		q = []NodeID{0}
	}
	x := nInput(in)
	e := nMetricEvaluate(m, x, nMetricInputError(in))
	hi := nHeadIndex(m, q)
	features := []map[string]any{}
	for i, c := range m.Circuits {
		w := 0.
		if hi >= 0 {
			w = m.Heads[hi].W[i]
		}
		v := e.Activation[i]
		if hi >= 0 && m.Heads[hi].Metric {
			v = e.Metric[i]
		}
		features = append(features, map[string]any{"node": c.Meta.Node, "kind": c.Meta.Kind, "parents": c.Meta.Parents, "axis": c.Axis, "activation": e.Activation[i], "metric": e.Metric[i], "code": e.Code[i], "delta": c.Delta, "center": c.Center, "signed_readout_weight": w, "readout_contribution": w * v})
	}
	return map[string]any{"x": x, "sources": slices.Clone(k.BrainState.NResolution.Sources), "query": slices.Clone(q), "model": m, "model_sha256": s2r1Hash(m), "features": features, "value": nPredict(m, x, q), "valid": e.Valid}, nil
}

func (k *KnowledgeBase) S2R1Input(in CompositionInput) []float64 { return nInput(in) }

func s2r1Metric(m nModel, rows []NWitness) map[string]any {
	delta, maximum := 0., 0.
	groups := map[string][]NWitness{}
	for _, r := range rows {
		v := nPredict(m, r.X, r.Query)
		delta += math.Abs(v-r.Outcome) / float64(len(rows))
		maximum = math.Max(maximum, math.Abs(v-r.Outcome))
		label := "propagate"
		if r.X[0] > 0 {
			label = "stop"
		}
		groups[label] = append(groups[label], r)
	}
	contexts := map[string]any{}
	for name, g := range groups {
		contexts[name] = map[string]any{"n": len(g), "mse": nError(m, g)}
	}
	return map[string]any{"n": len(rows), "mse": nError(m, rows), "rmse": math.Sqrt(nError(m, rows)), "mae": delta, "max_abs_error": maximum, "by_operation": contexts}
}

// Evaluates proposed readout/source parameter interventions without fitting.
func (k *KnowledgeBase) S2R1Causal(rows []NObservation) (map[string]any, error) {
	k.RLock()
	defer k.RUnlock()
	m, err := k.s2r1Model()
	if err != nil {
		return nil, err
	}
	w := []NWitness{}
	for _, r := range rows {
		ids, _ := k.compositionWordsLocked(r.Input.Words)
		w = append(w, NWitness{X: nInput(r.Input), Query: ids[0], Outcome: r.Outcome, EpisodeID: r.EpisodeID})
	}
	q := w[0].Query
	hi := nHeadIndex(m, q)
	if hi < 0 {
		return nil, errors.New("query absent")
	}
	base := make([]float64, len(w))
	for i, r := range w {
		base[i] = nPredict(m, r.X, q)
	}
	evaluate := func(label string, c nModel, data []NWitness) map[string]any {
		changes, maxDelta := 0, 0.
		for i, r := range data {
			d := math.Abs(nPredict(c, r.X, q) - base[i])
			if d > 0 {
				changes++
			}
			maxDelta = math.Max(maxDelta, d)
		}
		return map[string]any{"intervention": label, "metrics": s2r1Metric(c, data), "changed_predictions": changes, "max_prediction_delta": maxDelta}
	}
	out := []map[string]any{}
	for i, c := range m.Circuits {
		for _, factor := range []float64{0, .5, -1, 1.5} {
			copy := nClone(m)
			copy.Heads[hi].W[i] *= factor
			row := evaluate(fmt.Sprintf("readout/%d/factor/%g", c.Meta.Node, factor), copy, w)
			row["node"] = c.Meta.Node
			row["factor"] = factor
			out = append(out, row)
		}
		copy := nClone(m)
		if c.Meta.Kind == 1 {
			for j := range copy.Circuits[i].Axis {
				copy.Circuits[i].Axis[j] = 0
			}
		} else {
			copy.Circuits[i].Center = 0
			copy.Circuits[i].Delta = 7.99
		}
		row := evaluate(fmt.Sprintf("circuit/%d/zero-direction-or-coarsen-product", c.Meta.Node), copy, w)
		row["node"] = c.Meta.Node
		out = append(out, row)
	}
	for coordinate := 0; coordinate < 70; coordinate++ {
		data := slices.Clone(w)
		for i := range data {
			data[i].X = slices.Clone(data[i].X)
			data[i].X[coordinate] = 0
		}
		row := evaluate(fmt.Sprintf("coordinate/%d/zero", coordinate), m, data)
		row["coordinate"] = coordinate
		out = append(out, row)
	}
	return map[string]any{"baseline": s2r1Metric(m, w), "interventions": out, "model_sha256": s2r1Hash(m)}, nil
}

// Counterfactual basis proposals use the existing quantizer, bounded solver,
// admission arithmetic and <=32-feature budget. They are not installed. These
// proposals are explicitly outside the native single-residual-axis search.
func (tr *S2R1Trace) S2R1BasisProbes() []map[string]any {
	q := tr.check[0].Query
	hi := nHeadIndex(tr.before, q)
	out := []map[string]any{}
	for coordinate := 56; coordinate < 70; coordinate++ {
		if coordinate == 62 || coordinate >= 65 {
			continue
		}
		for _, factor := range []float64{1.0 / 16, 1, 16} {
			axis := make([]float64, nDimensions)
			axis[coordinate] = 1
			for _, metric := range []bool{false, true} {
				m := nClone(tr.before)
				m.Circuits = append(m.Circuits, nCircuit{Meta: NFeature{Kind: 1}, Axis: axis, Center: nCenter(axis, tr.fit), Delta: nMetricAxisDelta(axis, tr.fit) * factor})
				for i := range m.Heads {
					m.Heads[i].W = append(m.Heads[i].W, 0)
				}
				m.Heads[hi].Metric = metric
				valid := nFit(&m, q, tr.fit)
				err := nError(m, tr.check)
				reg := nMetricRegression(tr.before, m, tr.witnesses)
				out = append(out, map[string]any{"coordinate": coordinate, "delta_factor": factor, "metric": metric, "fit_valid": valid, "fit_mse": nError(m, tr.fit), "check_mse": err, "check_gain": nImproves(err, nError(tr.before, tr.check)), "regressions": reg, "duplicate": nMetricRedundant(m, append(slices.Clone(tr.fit), tr.check...), metric), "retention_by_query": s2r1Retention(tr.before, m, tr.witnesses), "new_axis": axis, "features": len(m.Circuits)})
			}
		}
	}
	return out
}

// A parameter intervention on rejected full refits. It does not call a new
// optimizer or install a learning rule. Restrict to an unchanged readout mode
// and unchanged old circuit parameters so factor=0 is exactly the incumbent.
func (tr *S2R1Trace) S2R1PartialStepProbes() []map[string]any {
	q := tr.check[0].Query
	hi := nHeadIndex(tr.before, q)
	out := []map[string]any{}
	for i, candidate := range tr.models {
		trial := tr.Trials[i]
		if candidate.Heads[hi].Metric != tr.before.Heads[hi].Metric || !trial["fit_valid"].(bool) || trial["kind"] == "resolution" {
			continue
		}
		for _, fraction := range []float64{0, .01, .05, .1, .25, .5, 1} {
			m := nClone(candidate)
			m.Heads[hi].B = tr.before.Heads[hi].B + fraction*(candidate.Heads[hi].B-tr.before.Heads[hi].B)
			for j := range m.Heads[hi].W {
				previous := 0.
				if j < len(tr.before.Heads[hi].W) {
					previous = tr.before.Heads[hi].W[j]
				}
				m.Heads[hi].W[j] = previous + fraction*(candidate.Heads[hi].W[j]-previous)
			}
			out = append(out, map[string]any{"candidate_index": i, "kind": trial["kind"], "fraction": fraction, "fit_mse": nError(m, tr.fit), "check_mse": nError(m, tr.check), "check_gain": nImproves(nError(m, tr.check), nError(tr.before, tr.check)), "regressions": nMetricRegression(tr.before, m, tr.witnesses), "retention_by_query": s2r1Retention(tr.before, m, tr.witnesses), "quality": s2r1Number(nMetricQuality(m, q, tr.check)), "model_sha256": s2r1Hash(m)})
		}
	}
	return out
}

func (tr *S2R1Trace) S2R1OriginalRetention(witness []NWitness) map[string]any {
	q := tr.check[0].Query
	return map[string]any{"selected_model": s2r1Metric(tr.best, tr.check), "regressions_against_original_witnesses": nMetricRegression(tr.before, tr.best, witness), "retention_by_query": s2r1Retention(tr.before, tr.best, witness), "maximum_error": nMetricMaximumError(tr.best, q, tr.check)}
}

// Evaluate a fixed, declared slice of the native ranking that the top-eight
// search did not fit. Uses the same feature definition, fit, validity checks,
// redundancy test and retention gate; never installs the candidate.
func (tr *S2R1Trace) S2R1TailProbes(first, last int) []map[string]any {
	q := tr.check[0].Query
	hi := nHeadIndex(tr.before, q)
	goal := tr.Trials[0]["goal"].(float64)
	bestErr := nError(tr.before, tr.check)
	bestBound := nMetricQuality(tr.before, q, tr.check)
	all := append(slices.Clone(tr.fit), tr.check...)
	out := []map[string]any{}
	for _, entry := range tr.Ranking {
		rank := entry["rank"].(int)
		if rank < first || rank > last {
			continue
		}
		parents := entry["parents"].([]NodeID)
		kind := entry["kind"].(int)
		a, b := -1, -1
		for i, c := range tr.before.Circuits {
			if c.Meta.Node == parents[0] {
				a = i
			}
			if c.Meta.Node == parents[1] {
				b = i
			}
		}
		raw := make([]float64, len(tr.fit))
		center := 0.
		for i, r := range tr.fit {
			v := nMetricEvaluate(tr.before, r.X, nil)
			vals := v.Activation
			if kind == 3 {
				vals = v.Metric
			}
			raw[i] = vals[a] * vals[b]
			center += raw[i] / float64(len(raw))
		}
		if math.Abs(center) > 7.99 {
			out = append(out, map[string]any{"rank": rank, "kind": kind, "parents": parents, "rejected_by": []string{"center-bound"}})
			continue
		}
		for _, mode := range []bool{false, true} {
			m := nClone(tr.before)
			m.Circuits = append(m.Circuits, nCircuit{Meta: NFeature{Kind: kind, Parents: slices.Clone(parents)}, Delta: nMetricDelta(raw, tr.fit), Center: center})
			for i := range m.Heads {
				m.Heads[i].W = append(m.Heads[i].W, 0)
			}
			m.Heads[hi].Metric = mode
			valid := nFit(&m, q, tr.fit)
			for _, r := range all {
				if !nMetricEvaluate(m, r.X, r.MeasurementError).Valid {
					valid = false
					break
				}
			}
			err := nError(m, tr.check)
			reg := nMetricRegression(tr.before, m, tr.witnesses)
			duplicate := nMetricRedundant(m, all, mode)
			bound := nMetricQuality(m, q, tr.check)
			errorGain := nImproves(err, bestErr)
			if goal > 0 && bestErr <= goal*goal/4 {
				errorGain = false
			}
			precisionGain := goal > 0 && bestBound > goal && nImproves(bound, bestBound) && err <= math.Max(bestErr, goal*goal)+nRoundoff(math.Max(bestErr, goal*goal))
			why := []string{}
			if !valid {
				why = append(why, "fit-invalid-or-parameter-bound")
			}
			if duplicate {
				why = append(why, "functionally-duplicate")
			}
			if reg > 0 {
				why = append(why, "retention-regression")
			}
			if !errorGain && !precisionGain {
				why = append(why, "no-check-improvement")
			}
			if tr.before.Heads[hi].Known && goal > 0 && nMetricPointError(tr.before, q, tr.check) <= goal {
				why = append(why, "existing-knowledge-meets-precision")
			}
			out = append(out, map[string]any{"rank": rank, "kind": kind, "parents": parents, "metric": mode, "fit_valid": valid, "fit_mse": nError(m, tr.fit), "check_mse": err, "quality": s2r1Number(bound), "regressions": reg, "duplicate": duplicate, "eligible": len(why) == 0, "rejected_by": why, "parameters": s2r1Parameters(tr.before, m, q), "model_sha256": s2r1Hash(m), "retention_by_query": s2r1Retention(tr.before, m, tr.witnesses)})
		}
	}
	return out
}

func (tr *S2R1Trace) S2R1DirectionStepContribution() []map[string]any {
	q := tr.check[0].Query
	hi := nHeadIndex(tr.before, q)
	out := []map[string]any{}
	for i, candidate := range tr.models {
		if tr.Trials[i]["kind"] != "missing-direction" || candidate.Heads[hi].Metric != tr.before.Heads[hi].Metric {
			continue
		}
		for _, fraction := range []float64{0, .01, .05, .1, .25, .5, 1} {
			m := nClone(candidate)
			m.Heads[hi].B = tr.before.Heads[hi].B + fraction*(candidate.Heads[hi].B-tr.before.Heads[hi].B)
			for j := range m.Heads[hi].W {
				old := 0.
				if j < len(tr.before.Heads[hi].W) {
					old = tr.before.Heads[hi].W[j]
				}
				m.Heads[hi].W[j] = old + fraction*(candidate.Heads[hi].W[j]-old)
			}
			ablations := []map[string]any{}
			for _, condition := range []string{"full", "no-forecast", "no-history", "no-error-magnitudes", "no-history-presence", "no-history-action", "no-extra", "permute-extra", "permute-error-magnitudes"} {
				rows := slices.Clone(tr.check)
				order := rand.New(rand.NewSource(3301)).Perm(len(rows))
				maximum := 0.
				changed := 0
				for j := range rows {
					rows[j].X = slices.Clone(rows[j].X)
					for coordinate := 56; coordinate < 70; coordinate++ {
						if coordinate == 62 || coordinate == 69 {
							continue
						}
						switch condition {
						case "no-forecast":
							if coordinate < 59 {
								rows[j].X[coordinate] = 0
							}
						case "no-history":
							if coordinate >= 59 {
								rows[j].X[coordinate] = 0
							}
						case "no-error-magnitudes":
							if coordinate >= 59 && coordinate <= 61 {
								rows[j].X[coordinate] = 0
							}
						case "no-history-presence":
							if coordinate == 63 {
								rows[j].X[coordinate] = 0
							}
						case "no-history-action":
							if coordinate == 64 {
								rows[j].X[coordinate] = 0
							}
						case "no-extra":
							rows[j].X[coordinate] = 0
						case "permute-extra":
							rows[j].X[coordinate] = tr.check[order[j]].X[coordinate]
						case "permute-error-magnitudes":
							if coordinate >= 59 && coordinate <= 61 {
								rows[j].X[coordinate] = tr.check[order[j]].X[coordinate]
							}
						}
					}
					d := math.Abs(nPredict(m, rows[j].X, q) - nPredict(m, tr.check[j].X, q))
					if d > 0 {
						changed++
					}
					maximum = math.Max(maximum, d)
				}
				ablations = append(ablations, map[string]any{"condition": condition, "metrics": s2r1Metric(m, rows), "max_prediction_delta": maximum, "changed_predictions": changed})
			}
			oldOnly, newOnly := nClone(m), nClone(m)
			last := len(m.Circuits) - 1
			oldOnly.Heads[hi].W[last] = 0
			newOnly.Heads[hi].B = tr.before.Heads[hi].B
			for j := 0; j < last; j++ {
				newOnly.Heads[hi].W[j] = tr.before.Heads[hi].W[j]
			}
			out = append(out, map[string]any{"candidate_index": i, "fraction": fraction, "new_feature_weight": m.Heads[hi].W[last], "full_metrics": s2r1Metric(m, tr.check), "old_readout_change_only": s2r1Metric(oldOnly, tr.check), "new_feature_change_only": s2r1Metric(newOnly, tr.check), "regressions": nMetricRegression(tr.before, m, tr.witnesses), "new_feature_only_regressions": nMetricRegression(tr.before, newOnly, tr.witnesses), "input_ablations": ablations, "model_sha256": s2r1Hash(m), "parameters": s2r1Parameters(tr.before, m, q)})
		}
	}
	return out
}

// A preregistered diagnostic positive control: replace all mutable stimulus
// coordinates with seeded independent, legal variation and use a known linear
// target of forecast/history. This tests reachability, never actual S2 skill.
func S2R1DiagnosticLesson(e CompositionExposure, seed int64, coordinate int, allVary bool) CompositionExposure {
	clone, _ := json.Marshal(e)
	var out CompositionExposure
	_ = json.Unmarshal(clone, &out)
	rng := rand.New(rand.NewSource(seed))
	out.ID = fmt.Sprintf("s2r1-positive-%d-%d", seed, coordinate)
	for set, rows := range [][]NObservation{out.Resolution.Fit, out.Resolution.Check} {
		for i := range rows {
			in := &rows[i].Input
			for a := range in.Sequence {
				for j := 0; j < 6; j++ {
					in.Sequence[a][j] = 0
					if allVary {
						in.Sequence[a][j] = .6 * (rng.Float64() - .5)
					}
				}
			}
			lag, part := coordinate/7, coordinate%7
			value := .6 * (rng.Float64() - .5)
			in.Sequence[len(in.Sequence)-1-lag][part] = value
			rows[i].Outcome = .1 + .2*value
			rows[i].EpisodeID = fmt.Sprintf("diagnostic-%d-%d-%d", seed, set, i)
		}
	}
	out.Input = out.Resolution.Fit[0].Input
	return out
}
