package knowledge

// Disposable opt-in constrained numerical workspace. Only the existing
// admission/writer may install its result in the canonical signed synapses.
import (
	"math"
	"slices"
)

const NMetricConstrainedV1 = "retention-constrained-native-v1"

type NConstraintTrace struct {
	Multiplier, FitMSE, HistoricalMSE, HistoricalLimit, Displacement float64
	Iterations, BoundActive, NativeViolations, LineSearches          int
	Status                                                           string
	RawReadout                                                       []float64
	RawBias, RawFitMSE, AnchorMSE, AnchorLimit                       float64
	InitialViolations                                                int
}

// The quadratic subproblem uses RAW affine predictions on a fixed quantized
// basis. Clipping, confident binary sign preservation and the complete original
// grouped regression contract are checked by nMetricRegression, never replaced
// by this surrogate. Box-constrained coordinate descent solves a sequence of
// Lagrangian subproblems; it is a finite search, not an infeasibility proof.
func nMetricConstrainedProposals(before, basis nModel, query []NodeID, fit, witnesses, anchors []NWitness, p *NMetricProposalConfig) []nMetricProposal {
	// Use a reproducible dyadic coefficient grid before fitting/evaluation.
	// Native population averaging otherwise introduces ulp drift when the same
	// coefficient is summed 4/16 times. This grid changes no tolerance.
	for i := range basis.Circuits {
		if i >= len(before.Circuits) || !slices.Equal(basis.Circuits[i].Axis, before.Circuits[i].Axis) || basis.Circuits[i].Delta != before.Circuits[i].Delta || basis.Circuits[i].Center != before.Circuits[i].Center {
			basis.Circuits[i].Axis = slices.Clone(basis.Circuits[i].Axis)
			for j, v := range basis.Circuits[i].Axis {
				basis.Circuits[i].Axis[j] = nConstraintCoefficient(v)
			}
			basis.Circuits[i].Delta = nConstraintCoefficient(basis.Circuits[i].Delta)
			basis.Circuits[i].Center = nConstraintCoefficient(basis.Circuits[i].Center)
		}
	}
	hi, bi := nHeadIndex(basis, query), nHeadIndex(before, query)
	if hi < 0 || bi < 0 || !before.Heads[bi].Known {
		return nil
	}
	base := nClone(basis)
	inc := before.Heads[bi]
	h := &base.Heads[hi]
	h.Metric, h.B, h.Known = inc.Metric, inc.B, true
	h.W = make([]float64, len(basis.Circuits))
	copy(h.W, inc.W)
	n := len(h.W) + 1
	initial := append(slices.Clone(h.W), h.B)
	// Bias-only and affine calibration restrict the search subspace explicitly.
	dim := n
	if p.Update == "bias" {
		dim = 1
	}
	if p.Update == "affine" {
		dim = 2
	}
	directions := make([][]float64, dim)
	for j := range directions {
		directions[j] = make([]float64, n)
		if p.Update == "bias" {
			directions[j][n-1] = 1
		} else if p.Update == "affine" {
			if j == 0 {
				copy(directions[j], initial)
			} else {
				directions[j][n-1] = 1
			}
		} else {
			directions[j][j] = 1
		}
	}
	type sample struct {
		v      []float64
		y, old float64
	}
	makeRows := func(rows []NWitness) []sample {
		out := []sample{}
		for _, r := range rows {
			if !slices.Equal(r.Query, query) {
				continue
			}
			v := append(nMetricValues(base, r.X, h.Metric), 1)
			z := make([]float64, dim)
			for j := range z {
				z[j] = nDot(directions[j], v)
			}
			out = append(out, sample{z, r.Outcome - nDot(initial, v), nDot(initial, v)})
		}
		return out
	}
	train, hist := makeRows(fit), makeRows(append(slices.Clone(witnesses), anchors...))
	if len(train) == 0 {
		return nil
	}
	scales := make([]float64, dim)
	for _, set := range [][]sample{train, hist} {
		for _, r := range set {
			for j, x := range r.v {
				scales[j] = math.Max(scales[j], math.Abs(x))
			}
		}
	}
	for j := range scales {
		if scales[j] == 0 {
			scales[j] = 1
		}
	}
	matrix := func(rows []sample) ([][]float64, []float64) {
		a := make([][]float64, dim)
		b := make([]float64, dim)
		for j := range a {
			a[j] = make([]float64, dim)
		}
		for _, r := range rows {
			for j, x := range r.v {
				x /= scales[j]
				b[j] += x * r.y / float64(len(rows))
				for k, y := range r.v {
					a[j][k] += x * y / scales[k] / float64(len(rows))
				}
			}
		}
		return a, b
	}
	a, b := matrix(train)
	g, c := matrix(hist)
	// Independent banks preserve the original contract without dilution.
	type retained struct {
		v       []float64
		y       float64
		protect bool
		bank    int
	}
	retainedRows := []retained{}
	limits := [2]float64{}
	counts := [2]int{}
	for bank, rows := range [][]NWitness{witnesses, anchors} {
		for _, r := range rows {
			if !slices.Equal(r.Query, query) {
				continue
			}
			old := nPredict(before, r.X, query)
			ev := nMetricEvaluate(before, r.X, r.MeasurementError)
			errs := ev.ActivationError
			if inc.Metric {
				errs = ev.MetricError
			}
			num := nRoundoff(old)
			for j, w := range inc.W {
				num += math.Abs(w) * errs[j]
			}
			l := math.Abs(old-r.Outcome) + num
			limits[bank] += l * l
			counts[bank]++
			retainedRows = append(retainedRows, retained{nMetricValues(base, r.X, inc.Metric), r.Outcome, math.Abs(r.Outcome) == 1 && old*r.Outcome > 0 && math.Abs(old) > inc.Error+nTolerance, bank})
		}
	}
	feasible := func(m nModel) bool {
		if !nMetricValidHead(m, hi) {
			return false
		}
		sums := [2]float64{}
		h := m.Heads[hi]
		for _, r := range retainedRows {
			v := math.Max(-1, math.Min(1, h.B+nDot(h.W, r.v)))
			if r.protect && v*r.y <= 0 {
				return false
			}
			d := v - r.y
			sums[r.bank] += d * d
		}
		for bank := range counts {
			if counts[bank] > 0 && nImproves(limits[bank]/float64(counts[bank]), sums[bank]/float64(counts[bank])) {
				return false
			}
		}
		return true
	}
	out := []nMetricProposal{}
	// A fixed logarithmic path plus feasible segment search. Check is not read.
	for _, lambda := range []float64{0, 1e-4, 1e-3, .01, .1, 1, 10, 100, 1e3, 1e4, 1e5, 1e6, 1e8} {
		z := make([]float64, dim)
		iterations := 0
		for it := 0; it < 256; it++ {
			iterations = it + 1
			change := 0.
			for j := range z {
				den := a[j][j] + lambda*g[j][j] + 1e-12
				rhs := b[j] + lambda*c[j]
				for k, v := range z {
					if k != j {
						rhs -= (a[j][k] + lambda*g[j][k]) * v
					}
				}
				value := rhs / den
				if p.Update == "readout" || p.Update == "bias" {
					idx := j
					if p.Update == "bias" {
						idx = n - 1
					}
					value = math.Max((-7.99-initial[idx])*scales[j], math.Min((7.99-initial[idx])*scales[j], value))
				}
				if !nFinite(value) {
					value = z[j]
				}
				change = math.Max(change, math.Abs(value-z[j]))
				z[j] = value
			}
			if change < 1e-11 {
				break
			}
		}
		build := func(fraction float64) nModel {
			m := nClone(base)
			theta := slices.Clone(initial)
			for j, v := range z {
				for k, d := range directions[j] {
					theta[k] += fraction * v / scales[j] * d
				}
			}
			for j, v := range theta {
				theta[j] = nConstraintCoefficient(v)
			}
			m.Heads[hi].W = slices.Clone(theta[:n-1])
			m.Heads[hi].B = theta[n-1]
			return m
		}
		candidate := build(1)
		rawReadout := slices.Clone(candidate.Heads[hi].W)
		rawBias := candidate.Heads[hi].B
		rawFit := nError(candidate, fit)
		reg := nMetricRegression(before, candidate, witnesses) + nMetricRegression(before, candidate, anchors)
		line := 0
		// A feasible incumbent anchors the segment only when its *basis* passes.
		// Structural candidates can have an infeasible zero step and are not
		// incorrectly declared safe by interpolation in a changed representation.
		if !feasible(candidate) && feasible(base) {
			lo, upper := 0., 1.
			for i := 0; i < 40; i++ {
				mid := (lo + upper) / 2
				line++
				if feasible(build(mid)) {
					lo = mid
				} else {
					upper = mid
				}
			}
			candidate = build(lo * .999999)
		}
		tr := &NConstraintTrace{Multiplier: lambda, Iterations: iterations, NativeViolations: reg, LineSearches: line, Status: "SEARCHED", RawReadout: rawReadout, RawBias: rawBias, RawFitMSE: rawFit, InitialViolations: reg}
		tr.NativeViolations = nMetricRegression(before, candidate, witnesses) + nMetricRegression(before, candidate, anchors)
		for _, r := range fit {
			if slices.Equal(r.Query, query) {
				v := nPredict(candidate, r.X, query) - r.Outcome
				tr.FitMSE += v * v / float64(len(fit))
			}
		}
		count := 0
		for _, r := range witnesses {
			if slices.Equal(r.Query, query) {
				count++
				v := nPredict(candidate, r.X, query) - r.Outcome
				tr.HistoricalMSE += v * v
				old := nPredict(before, r.X, query)
				e := nMetricEvaluate(before, r.X, r.MeasurementError)
				errs := e.ActivationError
				if inc.Metric {
					errs = e.MetricError
				}
				num := nRoundoff(old)
				for j, w := range inc.W {
					num += math.Abs(w) * errs[j]
				}
				v = math.Abs(old-r.Outcome) + num
				tr.HistoricalLimit += v * v
			}
		}
		for _, r := range anchors {
			if !slices.Equal(r.Query, query) {
				continue
			}
			v := nPredict(candidate, r.X, query) - r.Outcome
			tr.AnchorMSE += v * v / float64(len(anchors))
			old := nPredict(before, r.X, query)
			ev := nMetricEvaluate(before, r.X, r.MeasurementError)
			num := nRoundoff(old)
			errs := ev.ActivationError
			if inc.Metric {
				errs = ev.MetricError
			}
			for j, w := range inc.W {
				num += math.Abs(w) * errs[j]
			}
			v = math.Abs(old-r.Outcome) + num
			tr.AnchorLimit += v * v / float64(len(anchors))
		}
		if count > 0 {
			tr.HistoricalMSE /= float64(count)
			tr.HistoricalLimit /= float64(count)
		}
		theta := append(slices.Clone(candidate.Heads[hi].W), candidate.Heads[hi].B)
		for j, v := range theta {
			tr.Displacement += (v - initial[j]) * (v - initial[j])
			if math.Abs(v) >= 7.99-1e-9 {
				tr.BoundActive++
			}
		}
		tr.Displacement = math.Sqrt(tr.Displacement)
		ok := nMetricValidHead(candidate, hi)
		if tr.Displacement < 1e-14 {
			tr.Status = "NO_BENEFICIAL_FEASIBLE_UPDATE"
		}
		if !feasible(candidate) {
			tr.Status = "INFEASIBLE_CANDIDATE"
		}
		out = append(out, nMetricProposal{Model: candidate, Name: "retention-constrained", Fraction: 1, Valid: ok, Constraint: tr})
	}
	return out
}

func nConstraintCoefficient(v float64) float64 {
	if v == 0 || !nFinite(v) {
		return v
	}
	f, e := math.Frexp(v)
	z := math.Ldexp(math.Round(math.Ldexp(f, 40)), e-40)
	if math.Abs(z) > 7.99 && math.Abs(v) <= 7.99 {
		z = math.Copysign(math.Abs(z)-math.Ldexp(1, e-40), z)
	}
	return z
}
