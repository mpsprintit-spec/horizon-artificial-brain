package knowledge

import (
	"errors"
	"math"
	"slices"
)

const NeuralCompositionMetricV6 = "synaptic-metric-context-v6"

func IsNumericalCompositionVersion(v string) bool {
	return v == NeuralCompositionResolutionV5 || v == NeuralCompositionMetricV6
}

type NProbeRequest struct {
	Kind        string `json:"kind"`
	Coordinates []int  `json:"coordinates,omitempty"`
	Evidence    string `json:"evidence"`
}

func (d NDiagnosis) Clone() NDiagnosis {
	if d.RequiredPrecision != nil {
		v := *d.RequiredPrecision
		d.RequiredPrecision = &v
	}
	if d.EstimatedAbsError != nil {
		v := *d.EstimatedAbsError
		d.EstimatedAbsError = &v
	}
	d.Query = slices.Clone(d.Query)
	d.Hypotheses = slices.Clone(d.Hypotheses)
	d.Probes = slices.Clone(d.Probes)
	for i := range d.Probes {
		d.Probes[i].Coordinates = slices.Clone(d.Probes[i].Coordinates)
	}
	d.Trials = slices.Clone(d.Trials)
	for i := range d.Trials {
		d.Trials[i].RejectedBy = slices.Clone(d.Trials[i].RejectedBy)
		if d.Trials[i].FitValid != nil {
			b := *d.Trials[i].FitValid
			d.Trials[i].FitValid = &b
		}
	}
	return d
}
func validateMetricInput(in CompositionInput) error {
	if in.RequiredPrecision != nil && (math.IsNaN(*in.RequiredPrecision) || math.IsInf(*in.RequiredPrecision, 0) || *in.RequiredPrecision <= 0 || *in.RequiredPrecision > 2) {
		return errors.New("required precision must be finite and in (0,2]")
	}
	if len(in.MeasurementError) > 0 {
		if len(in.MeasurementError) != len(in.Sequence) {
			return errors.New("measurement error must match sequence")
		}
		for _, row := range in.MeasurementError {
			if len(row) != 6 {
				return errors.New("measurement error needs six channels")
			}
			for _, x := range row {
				if math.IsNaN(x) || math.IsInf(x, 0) || x < 0 || x > 2 {
					return errors.New("invalid measurement error")
				}
			}
		}
	}
	return nil
}
func nMetricInputError(in CompositionInput) []float64 {
	if len(in.MeasurementError) == 0 {
		return nil
	}
	v := make([]float64, nDimensions)
	for i, row := range in.MeasurementError {
		for j, x := range row {
			v[(len(in.Sequence)-1-i)*7+j] = x * in.Sequence[i][6]
		}
	}
	return v
}

type nMetricEvaluation struct {
	Activation, Code, Metric, MetricError, ActivationError []float64
	Valid                                                  bool
}

func nFinite(x float64) bool { return !math.IsNaN(x) && !math.IsInf(x, 0) }
func nRoundoff(x float64) float64 {
	a := math.Abs(x)
	if a == 0 {
		return math.SmallestNonzeroFloat64
	}
	return 16 * (math.Nextafter(a, math.Inf(1)) - a)
}
func nImproves(a, b float64) bool {
	return nFinite(a) && a+nRoundoff(math.Max(math.Abs(a), math.Abs(b))) < b
}
func nMetricEvaluate(m nModel, x, inputError []float64) nMetricEvaluation {
	count := len(m.Circuits)
	e := nMetricEvaluation{make([]float64, count), make([]float64, count), make([]float64, count), make([]float64, count), make([]float64, count), true}
	ids := map[NodeID]int{}
	for i, c := range m.Circuits {
		raw, unc := 0., 0.
		if c.Meta.Kind == 1 {
			raw = nDot(c.Axis, x)
			if inputError != nil {
				for j, a := range c.Axis {
					unc += math.Abs(a) * inputError[j]
				}
			}
			// Projection arithmetic and quantization contribute different errors.
			sum := 0.
			for j, a := range c.Axis {
				sum += math.Abs(a * x[j])
			}
			unc += nRoundoff(sum)
		} else {
			vals, errs := e.Activation, e.ActivationError
			if c.Meta.Kind == 3 {
				vals, errs = e.Metric, e.MetricError
			}
			a, b := ids[c.Meta.Parents[0]], ids[c.Meta.Parents[1]]
			raw = vals[a] * vals[b]
			unc = math.Abs(vals[a])*errs[b] + math.Abs(vals[b])*errs[a] + errs[a]*errs[b] + nRoundoff(raw)
		}
		if !nFinite(raw) || !nFinite(unc) || c.Delta <= 0 || !nFinite((raw-c.Center)/c.Delta) || math.Abs((raw-c.Center)/c.Delta) > math.Pow(2, 52) {
			e.Valid = false
			return e
		}
		code := math.Round((raw - c.Center) / c.Delta)
		metric := c.Center + code*c.Delta
		e.Code[i] = code
		e.Metric[i] = metric
		e.Activation[i] = math.Tanh(code)
		e.MetricError[i] = unc + c.Delta/2 + nRoundoff(metric)
		lo := math.Tanh(math.Round((raw - unc - c.Center) / c.Delta))
		hi := math.Tanh(math.Round((raw + unc - c.Center) / c.Delta))
		e.ActivationError[i] = math.Max(math.Abs(e.Activation[i]-lo), math.Abs(hi-e.Activation[i])) + nRoundoff(e.Activation[i])
		ids[c.Meta.Node] = i
	}
	return e
}
func nMetricValues(m nModel, x []float64, metric bool) []float64 {
	e := nMetricEvaluate(m, x, nil)
	if metric {
		return e.Metric
	}
	return e.Activation
}
func nMetricDelta(values []float64, rows []NWitness) float64 {
	scale := 0.
	for _, v := range values {
		scale = math.Max(scale, math.Abs(v))
	}
	floor := math.Max(nRoundoff(scale), math.SmallestNonzeroFloat64)
	gap := math.Inf(1)
	for i, a := range rows {
		for j, b := range rows[:i] {
			if math.Abs(a.Outcome-b.Outcome) > nRoundoff(math.Max(math.Abs(a.Outcome), math.Abs(b.Outcome))) {
				d := math.Abs(values[i] - values[j])
				if d > floor {
					gap = math.Min(gap, d)
				}
			}
		}
	}
	if math.IsInf(gap, 1) {
		if scale == 0 {
			return 1
		}
		return math.Min(7.99, math.Max(floor, scale/16))
	}
	return math.Min(7.99, math.Max(floor, gap/16))
}
func nMetricAxisDelta(axis []float64, rows []NWitness) float64 {
	v := make([]float64, len(rows))
	for i, r := range rows {
		v[i] = nDot(axis, r.X)
	}
	return nMetricDelta(v, rows)
}
func nMetricQuality(m nModel, q []NodeID, rows []NWitness) float64 {
	hi := nHeadIndex(m, q)
	if hi < 0 || !m.Heads[hi].Known {
		return math.Inf(1)
	}
	h := m.Heads[hi]
	worst := 0.
	for _, r := range rows {
		eval := nMetricEvaluate(m, r.X, r.MeasurementError)
		if !eval.Valid {
			return math.Inf(1)
		}
		errors := eval.ActivationError
		if h.Metric {
			errors = eval.MetricError
		}
		bound := 0.
		for i, w := range h.W {
			bound += math.Abs(w) * errors[i]
		}
		worst = math.Max(worst, bound)
	}
	return worst + nMetricMaximumError(m, q, rows)
}
func nMetricPointError(m nModel, q []NodeID, rows []NWitness) float64 {
	err := 0.
	for _, r := range rows {
		err = math.Max(err, math.Abs(nPredict(m, r.X, q)-r.Outcome))
	}
	return err
}
func nMetricMaximumError(m nModel, q []NodeID, rows []NWitness) float64 {
	err := 0.
	for _, r := range rows {
		err = math.Max(err, math.Min(2, math.Abs(nPredict(m, r.X, q)-r.Outcome)+r.OutcomeError))
	}
	return err
}

// Bounded least squares. Targets at +/-1 are censored by the existing output
// range; rows already beyond that boundary have zero clipped-output residual.
// Column normalization is a disposable solver workspace, never fitted memory.
func nFitMetric(m *nModel, hi int, rows []NWitness) bool {
	if len(rows) == 0 {
		return false
	}
	n := len(m.Circuits) + 1
	vals := make([][]float64, len(rows))
	scales := make([]float64, n)
	for i, r := range rows {
		evaluated := nMetricEvaluate(*m, r.X, nil)
		if !evaluated.Valid {
			return false
		}
		v := evaluated.Metric
		if !m.Heads[hi].Metric {
			v = evaluated.Activation
		}
		v = append(v, 1)
		vals[i] = v
		for j, x := range v {
			scales[j] += x * x
		}
	}
	// A constant feature has no identifiable coefficient in this lesson: the
	// intercept already represents it. Normalizing a tiny nonzero constant to
	// unit magnitude otherwise splits the bias across redundant columns and
	// can create enormous coefficients after unscaling. Keep the circuit for
	// other tasks, but give this unidentifiable readout coefficient zero weight.
	for j := 0; j < n-1; j++ {
		lo, hi := vals[0][j], vals[0][j]
		for i := range vals {
			lo, hi = math.Min(lo, vals[i][j]), math.Max(hi, vals[i][j])
		}
		if hi-lo <= nRoundoff(math.Max(math.Abs(lo), math.Abs(hi))) {
			for i := range vals {
				vals[i][j] = 0
			}
			scales[j] = 0
		}
	}
	for j := range scales {
		scales[j] = math.Sqrt(scales[j] / float64(len(rows)))
		if scales[j] == 0 {
			scales[j] = 1
		}
	}
	weights := make([]float64, n)
	for iteration := 0; iteration < 16; iteration++ {
		a := make([][]float64, n)
		for j := range a {
			a[j] = make([]float64, n+1)
			a[j][j] = 1e-12
		}
		active := 0
		for i, r := range rows {
			raw := nDot(weights, vals[i])
			if iteration > 0 && ((r.Outcome == 1 && raw > 1) || (r.Outcome == -1 && raw < -1)) {
				continue
			}
			active++
			for j, x := range vals[i] {
				x /= scales[j]
				a[j][n] += x * r.Outcome
				for k, y := range vals[i] {
					a[j][k] += x * y / scales[k]
				}
			}
		}
		if active == 0 {
			break
		}
		for j := 0; j < n; j++ {
			pivot := j
			for k := j + 1; k < n; k++ {
				if math.Abs(a[k][j]) > math.Abs(a[pivot][j]) {
					pivot = k
				}
			}
			a[j], a[pivot] = a[pivot], a[j]
			if !nFinite(a[j][j]) || math.Abs(a[j][j]) < 1e-15 {
				return false
			}
			d := a[j][j]
			for k := j; k <= n; k++ {
				a[j][k] /= d
			}
			for k := 0; k < n; k++ {
				if k == j {
					continue
				}
				d = a[k][j]
				for l := j; l <= n; l++ {
					a[k][l] -= d * a[j][l]
				}
			}
		}
		change := 0.
		for j := range weights {
			v := a[j][n] / scales[j]
			if !nFinite(v) || math.Abs(v) > 7.99 {
				return false
			}
			change = math.Max(change, math.Abs(v-weights[j]))
			weights[j] = v
		}
		if change < 1e-12 {
			break
		}
	}
	h := m.Heads[hi]
	h.W = slices.Clone(weights[:n-1])
	h.B = weights[n-1]
	h.Known = true
	m.Heads[hi] = h
	return true
}
func nMetricRegression(before, after nModel, rows []NWitness) int {
	type group struct {
		q        []NodeID
		limit, b float64
		n        int
	}
	groups := []group{}
	violations := 0
	for _, r := range rows {
		av, bv := nPredict(before, r.X, r.Query), nPredict(after, r.X, r.Query)
		i := -1
		for j, g := range groups {
			if slices.Equal(g.q, r.Query) {
				i = j
				break
			}
		}
		if i < 0 {
			groups = append(groups, group{q: r.Query})
			i = len(groups) - 1
		}
		hi := nHeadIndex(before, r.Query)
		numerical := nRoundoff(av)
		if hi >= 0 {
			e := nMetricEvaluate(before, r.X, r.MeasurementError)
			errs := e.ActivationError
			if before.Heads[hi].Metric {
				errs = e.MetricError
			}
			for j, w := range before.Heads[hi].W {
				numerical += math.Abs(w) * errs[j]
			}
		}
		limit := math.Abs(av-r.Outcome) + numerical
		groups[i].limit += limit * limit
		groups[i].b += (bv - r.Outcome) * (bv - r.Outcome)
		groups[i].n++
		if hi >= 0 && math.Abs(r.Outcome) == 1 && av*r.Outcome > 0 && math.Abs(av) > before.Heads[hi].Error+nTolerance && bv*r.Outcome <= 0 {
			violations++
		}
	}
	for _, g := range groups {
		if nImproves(g.limit/float64(g.n), g.b/float64(g.n)) {
			violations++
		}
	}
	return violations
}
func nBinarySolved(m nModel, q []NodeID, rows []NWitness) bool {
	hi := nHeadIndex(m, q)
	if hi < 0 || !m.Heads[hi].Known {
		return false
	}
	error := math.Sqrt(nError(m, rows))
	for _, r := range rows {
		v := nPredict(m, r.X, q)
		if v*r.Outcome <= 0 || math.Abs(v) <= error+nTolerance {
			return false
		}
	}
	return true
}
func nMetricConflict(rows []NWitness) bool {
	for i, a := range rows {
		for _, b := range rows[:i] {
			if slices.Equal(a.Query, b.Query) && slices.Equal(a.X, b.X) && math.Abs(a.Outcome-b.Outcome) > nRoundoff(math.Max(math.Abs(a.Outcome), math.Abs(b.Outcome))) {
				return true
			}
		}
	}
	return false
}
func nMetricRedundant(m nModel, rows []NWitness, metric bool) bool {
	last := len(m.Circuits) - 1
	if last <= 0 {
		return false
	}
	for i := 0; i < last; i++ {
		same, opposite := true, true
		for _, r := range rows {
			v := nMetricValues(m, r.X, metric)
			tol := nRoundoff(math.Max(math.Abs(v[i]), math.Abs(v[last])))
			same = same && math.Abs(v[i]-v[last]) <= tol
			opposite = opposite && math.Abs(v[i]+v[last]) <= tol
		}
		if same || opposite {
			return true
		}
	}
	return false
}
func nMetricAliasPairs(m nModel, q []NodeID, rows []NWitness) int {
	count := 0
	hi := nHeadIndex(m, q)
	mode := hi >= 0 && m.Heads[hi].Metric
	for i, a := range rows {
		av := nMetricValues(m, a.X, mode)
		for _, b := range rows[:i] {
			if a.Outcome == b.Outcome {
				continue
			}
			bv := nMetricValues(m, b.X, mode)
			if slices.Equal(av, bv) {
				count++
			}
		}
	}
	return count
}
func (k *KnowledgeBase) readNMetric(m nModel, q []NodeID, in CompositionInput) CompositionReadout {
	hi := nHeadIndex(m, q)
	v := nPredict(m, nInput(in), q)
	out := CompositionReadout{Status: "untrained-n-resolution", SceneAt: in.Scene.ObservedAt, Coordinates: []float64{0, 0, 0, 0, 4 * v, 4}, NResolution: &NResolutionReadout{Value: v}}
	if hi < 0 || !m.Heads[hi].Known {
		return out
	}
	h := m.Heads[hi]
	e := nMetricEvaluate(m, nInput(in), nMetricInputError(in))
	if !e.Valid {
		out.Status = "numerical-range-exceeded"
		out.NResolution.Value = 0
		return out
	}
	vals, errs := e.Activation, e.ActivationError
	if h.Metric {
		vals, errs = e.Metric, e.MetricError
	}
	estimated := h.MaxError
	for i, c := range m.Circuits {
		metric, activation, qerr := e.Metric[i], e.Activation[i], e.MetricError[i]
		out.NResolution.Units = append(out.NResolution.Units, NValue{Node: c.Meta.Node, Code: e.Code[i], Resolution: c.Delta, Contribution: h.W[i] * vals[i], Metric: &metric, Activation: &activation, QuantizationError: &qerr})
		estimated += math.Abs(h.W[i]) * errs[i]
	}
	estimated = math.Min(2, estimated+nRoundoff(v))
	out.NResolution.ObservedCheckRMSE = h.Error
	out.NResolution.MetricReadout = h.Metric
	out.NResolution.ObservedCheckMaxError = &h.MaxError
	out.NResolution.EstimatedAbsError = &estimated
	if in.RequiredPrecision != nil {
		tol := *in.RequiredPrecision
		out.NResolution.RequestedPrecision = &tol
		out.Accepted = estimated <= tol
		out.Status = "precision-not-met"
		if out.Accepted {
			out.Status = "precision-met"
		}
		// No fabricated probability or magnitude signal for a metrical goal.
		return out
	}
	if h.Metric {
		out.Status = "metric-precision-required"
		return out
	}
	out.Confidence = math.Max(0, math.Min(1, math.Abs(v)-h.Error))
	out.Accepted = math.Abs(v) > h.Error+nTolerance
	out.Status = "numeric-prediction"
	if !out.Accepted {
		out.Status = "uncertain-numerical-state"
	}
	return out
}

// LastNDiagnosis is a detached read under the canonical Brain lock.
func (k *KnowledgeBase) LastNDiagnosis() *NDiagnosis {
	k.RLock()
	defer k.RUnlock()
	if k.BrainState.NResolution == nil {
		return nil
	}
	d := k.BrainState.NResolution.Last.Clone()
	return &d
}

// Caller already holds k's read or mutation lock.
func (k *KnowledgeBase) validateNContract(c CompositionConfig) error {
	s := k.BrainState.NResolution
	if s == nil {
		return nil
	}
	if c.Version == NeuralCompositionMetricV6 {
		if s.Contract != NeuralCompositionMetricV6 {
			return errors.New("v5 N state requires an explicit migration before v6")
		}
	} else if s.Contract != "" {
		return errors.New("metric N state requires v6")
	}
	return nil
}
func validateMetricDiagnosis(d NDiagnosis, metric bool) error {
	if !metric {
		if d.RequiredPrecision != nil || d.EstimatedAbsError != nil || d.Stage != "" || len(d.Hypotheses) > 0 || len(d.Probes) > 0 || d.AliasPairs != 0 {
			return errors.New("metric diagnosis requires v6")
		}
		for _, t := range d.Trials {
			if t.ReadoutMode != "" || t.FitValid != nil || len(t.RejectedBy) > 0 {
				return errors.New("metric trial requires v6")
			}
		}
		return nil
	}
	if d.RequiredPrecision != nil && (!nFinite(*d.RequiredPrecision) || *d.RequiredPrecision <= 0 || *d.RequiredPrecision > 2) {
		return errors.New("invalid diagnostic precision")
	}
	if d.EstimatedAbsError != nil && (!nFinite(*d.EstimatedAbsError) || *d.EstimatedAbsError < 0 || *d.EstimatedAbsError > 2) {
		return errors.New("invalid diagnostic uncertainty")
	}
	if len(d.Stage) > 64 || len(d.Hypotheses) > 8 || len(d.Probes) > 8 || len(d.Trials) > 128 || d.AliasPairs < 0 {
		return errors.New("metric diagnosis exceeds resource limits")
	}
	if !nFinite(d.BeforeError) || !nFinite(d.AfterError) || d.BeforeError < 0 || d.AfterError < 0 || d.BeforeError > 4.0000001 || d.AfterError > 4.0000001 {
		return errors.New("invalid diagnostic error")
	}
	for _, h := range d.Hypotheses {
		if len(h) > 64 {
			return errors.New("invalid diagnostic hypothesis")
		}
	}
	for _, p := range d.Probes {
		if len(p.Kind) > 64 || len(p.Evidence) > 256 || len(p.Coordinates) > 8 {
			return errors.New("invalid diagnostic probe")
		}
		for _, i := range p.Coordinates {
			if i < 0 || i >= nDimensions {
				return errors.New("invalid diagnostic coordinate")
			}
		}
	}
	for _, t := range d.Trials {
		if (t.ReadoutMode != "activation" && t.ReadoutMode != "metric") || t.FitValid == nil || len(t.RejectedBy) > 8 || !nFinite(t.CheckError) || t.CheckError < 0 || t.CheckError > 4.0000001 {
			return errors.New("invalid metric trial")
		}
		for _, reason := range t.RejectedBy {
			if len(reason) > 64 {
				return errors.New("invalid rejection reason")
			}
		}
	}
	return nil
}
