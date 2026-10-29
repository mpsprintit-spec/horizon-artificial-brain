package knowledge

// Opt-in research contract. Numeric witnesses are experience, never an answer
// lookup. All fitted directions, resolutions and readouts are canonical signed
// synapses. No similarity threshold grants semantic identity or neuron growth.
import (
	"errors"
	"math"
	"slices"
	"sort"
	"time"
)

const NeuralCompositionResolutionV5 = "synaptic-resolution-diagnosis-v5"
const nDimensions = MaxCompositionSequence * 7
const maxNFeatures = 32 // resource ceiling, not a quota per object
const maxNWitnesses = 128
const nTolerance = 1e-8 // numerical solver tolerance; never an action threshold

// A validated direction and its resolution are one numerical contract. Passive
// decay must not change their ratio or invalidate normalized directions. Support
// metadata can still decay. Circuit retirement requires a separate diagnosis.
// This closure is a detached index rebuilt for one memory update, not a cache.
func (k *KnowledgeBase) protectedNParameter() func(NodeID, NodeID) bool {
	units, queries, scalarSources := map[NodeID]bool{}, map[NodeID]bool{}, map[NodeID]bool{}
	if s := k.BrainState.NResolution; s != nil {
		for _, id := range s.Sources {
			if id != 0 {
				units[id] = true
			}
		}
		for _, id := range s.Controls {
			units[id] = true
		}
		for _, f := range s.Features {
			units[f.Node] = true
		}
		for _, h := range s.Heads {
			for _, id := range h.Query {
				queries[id] = true
			}
		}
		k.projectionMu.RLock()
		for _, p := range k.ProjectionPopulations {
			if p.Scope != nil && *p.Scope == compositionScope && len(p.Prototype) == 1 && (p.Prototype[0]*64 == 24 || p.Prototype[0]*64 == 26 || (s.Contract == NeuralCompositionMetricV6 && p.Prototype[0]*64 == 27)) {
				for _, u := range p.Units {
					scalarSources[u.NodeID] = true
				}
			}
		}
		k.projectionMu.RUnlock()
	}
	return func(a, b NodeID) bool { return units[a] || units[b] || (scalarSources[a] && queries[b]) }
}

type NObservation struct {
	OutcomeError float64          `json:"outcome_error,omitempty"`
	Input        CompositionInput `json:"input"`
	Outcome      float64          `json:"outcome"`
	ObservedAt   time.Time        `json:"observed_at"`
	EpisodeID    string           `json:"episode_id"`
}
type NResolutionLesson struct {
	Fit   []NObservation `json:"fit"`
	Check []NObservation `json:"check"`
}
type NFeature struct {
	Node    NodeID   `json:"node"`
	Kind    int      `json:"kind"`
	Parents []NodeID `json:"parents,omitempty"`
}
type NWitness struct {
	OutcomeError     float64   `json:"outcome_error,omitempty"`
	MeasurementError []float64 `json:"measurement_error,omitempty"`
	X                []float64 `json:"x"`
	Query            []NodeID  `json:"query"`
	Outcome          float64   `json:"outcome"`
	EpisodeID        string    `json:"episode_id"`
}
type NTrial struct {
	ReadoutMode          string   `json:"readout_mode,omitempty"`
	FitValid             *bool    `json:"fit_valid,omitempty"`
	RejectedBy           []string `json:"rejected_by,omitempty"`
	Kind                 string   `json:"kind"`
	Feature              NodeID   `json:"feature,omitempty"`
	CheckError           float64  `json:"check_error"`
	RetentionRegressions int      `json:"retention_regressions"`
	Eligible             bool     `json:"eligible"`
}
type NDiagnosis struct {
	RequiredPrecision *float64        `json:"required_precision,omitempty"`
	EstimatedAbsError *float64        `json:"estimated_abs_error,omitempty"`
	Stage             string          `json:"stage,omitempty"`
	Hypotheses        []string        `json:"hypotheses,omitempty"`
	Probes            []NProbeRequest `json:"probes,omitempty"`
	AliasPairs        int             `json:"alias_pairs,omitempty"`
	Query             []NodeID        `json:"query"`
	Cause             string          `json:"cause"`
	BeforeError       float64         `json:"before_error"`
	AfterError        float64         `json:"after_error"`
	Committed         bool            `json:"committed"`
	Grown             NodeID          `json:"grown,omitempty"`
	ResolutionBefore  float64         `json:"resolution_before,omitempty"`
	ResolutionAfter   float64         `json:"resolution_after,omitempty"`
	Trials            []NTrial        `json:"trials"`
}
type NHeadTopology struct {
	Query    []NodeID `json:"query"`
	Features []NodeID `json:"features"`
}
type NResolutionState struct {
	Contract  string          `json:"contract,omitempty"`
	Controls  []NodeID        `json:"controls,omitempty"`
	Heads     []NHeadTopology `json:"heads"`
	Sources   []NodeID        `json:"sources"`
	Features  []NFeature      `json:"features"`
	Witnesses []NWitness      `json:"witnesses"`
	Last      NDiagnosis      `json:"last"`
}
type NValue struct {
	Metric            *float64 `json:"metric,omitempty"`
	Activation        *float64 `json:"activation,omitempty"`
	QuantizationError *float64 `json:"quantization_error,omitempty"`
	Contribution      float64  `json:"contribution"`
	Node              NodeID   `json:"node"`
	Code              float64  `json:"code"`
	Resolution        float64  `json:"resolution"`
}
type NResolutionReadout struct {
	MetricReadout           bool     `json:"metric_readout,omitempty"`
	ObservedCheckMaxError   *float64 `json:"observed_check_max_error,omitempty"`
	EstimatedAbsError       *float64 `json:"estimated_abs_error,omitempty"`
	RequestedPrecision      *float64 `json:"requested_precision,omitempty"`
	Value                   float64  `json:"value"`
	ObservedCheckRMSE       float64  `json:"observed_check_rmse"`
	Units                   []NValue `json:"units"`
	IsCalibratedProbability bool     `json:"is_calibrated_probability"`
}

func validateNLesson(e CompositionExposure) error {
	if e.Resolution == nil {
		return nil
	}
	if len(e.Input.Words) != 1 || len(e.Input.Sequence) == 0 {
		return errors.New("N lesson needs a query and numerical sequence")
	}
	b := e.Resolution
	if len(b.Fit) < 4 || len(b.Check) < 4 || len(b.Fit) > 64 || len(b.Check) > 64 {
		return errors.New("N lesson requires bounded fit and independent check sets")
	}
	fit, check := map[string]bool{}, map[string]bool{}
	for set, rows := range [][]NObservation{b.Fit, b.Check} {
		for _, o := range rows {
			if !nFinite(o.OutcomeError) || o.OutcomeError < 0 || o.OutcomeError > 2 {
				return errors.New("invalid outcome uncertainty")
			}
			if err := o.Input.Validate(); err != nil {
				return err
			}
			if len(o.Input.Words) != 1 || !slices.Equal(o.Input.Words[0], e.Input.Words[0]) || len(o.Input.Sequence) == 0 || o.EpisodeID == "" || len(o.EpisodeID) > 128 || math.IsNaN(o.Outcome) || math.IsInf(o.Outcome, 0) || math.Abs(o.Outcome) > 1 || o.ObservedAt.IsZero() || o.ObservedAt.Before(o.Input.Scene.ObservedAt) || o.ObservedAt.After(e.After.ObservedAt) {
				return errors.New("invalid or not-yet-observed N evidence")
			}
			if set == 0 {
				fit[o.EpisodeID] = true
			} else {
				if fit[o.EpisodeID] {
					return errors.New("N check shares a fit episode")
				}
				check[o.EpisodeID] = true
			}
		}
	}
	if len(fit) < 2 || len(check) < 2 {
		return errors.New("N diagnosis needs independent episodes")
	}
	return nil
}

// An explicit sensor codec preserves six measurements at each observed lag.
// Source coordinates are scaffolding, not semantic identities or object slots.
func nInput(in CompositionInput) []float64 {
	x := make([]float64, nDimensions)
	for i, row := range in.Sequence {
		x[(len(in.Sequence)-1-i)*7+6] = row[6]
		for j := 0; j < 6; j++ {
			x[(len(in.Sequence)-1-i)*7+j] = row[j] * row[6]
		}
	}
	return x
}

type nCircuit struct {
	Meta   NFeature
	Axis   []float64
	Delta  float64
	Center float64
}
type nHead struct {
	Metric   bool
	MaxError float64
	Query    []NodeID
	W        []float64
	B, Error float64
	Known    bool
}
type nModel struct {
	MetricEnabled bool
	Circuits      []nCircuit
	Heads         []nHead
}

func nDot(a, b []float64) float64 {
	v := 0.
	for i, x := range a {
		v += x * b[i]
	}
	return v
}
func nCodes(m nModel, x []float64) ([]float64, []float64) {
	if m.MetricEnabled {
		e := nMetricEvaluate(m, x, nil)
		return e.Activation, e.Code
	}
	codes := make([]float64, len(m.Circuits))
	values := make([]float64, len(codes))
	index := map[NodeID]int{}
	for i, c := range m.Circuits {
		raw := 0.
		if c.Meta.Kind == 1 {
			raw = nDot(c.Axis, x)
		} else {
			raw = 1
			for _, id := range c.Meta.Parents {
				raw *= values[index[id]]
			}
		}
		codes[i] = math.Round((raw - c.Center) / c.Delta)
		values[i] = math.Tanh(codes[i])
		index[c.Meta.Node] = i
	}
	return values, codes
}
func nHeadIndex(m nModel, q []NodeID) int {
	for i, h := range m.Heads {
		if slices.Equal(h.Query, q) {
			return i
		}
	}
	return -1
}
func nPredict(m nModel, x []float64, q []NodeID) float64 {
	i := nHeadIndex(m, q)
	if i < 0 {
		return 0
	}
	v, _ := nCodes(m, x)
	if m.MetricEnabled && m.Heads[i].Metric {
		v = nMetricEvaluate(m, x, nil).Metric
	}
	return math.Max(-1, math.Min(1, m.Heads[i].B+nDot(m.Heads[i].W, v)))
}
func (k *KnowledgeBase) loadNModel(bank [compositionBanks][]NodeID, queries [][]NodeID) nModel {
	var m nModel
	s := k.BrainState.NResolution
	if s != nil {
		m.MetricEnabled = s.Contract == NeuralCompositionMetricV6
		for _, f := range s.Features {
			c := nCircuit{Meta: f, Axis: make([]float64, nDimensions)}
			c.Delta, _ = k.compositionParameter(bank[25], []NodeID{f.Node})
			c.Center, _ = k.compositionParameter(bank[24], []NodeID{f.Node})
			if f.Kind == 1 {
				for j, id := range s.Sources {
					if id != 0 {
						c.Axis[j], _ = k.compositionParameter([]NodeID{id}, []NodeID{f.Node})
					}
				}
			}
			m.Circuits = append(m.Circuits, c)
		}
	}
	for _, q := range queries {
		h := nHead{Query: slices.Clone(q), W: make([]float64, len(m.Circuits))}
		if len(q) > 0 && q[0] != 0 {
			h.B, h.Known = k.compositionParameter(bank[24], q)
			h.Error, _ = k.compositionParameter(bank[26], q)
			if m.MetricEnabled {
				v, _ := k.compositionParameter(bank[27], q)
				h.Metric = v == 1
				if len(s.Controls) > 0 {
					h.MaxError, _ = k.compositionParameter(s.Controls[:1], q)
				}
			}
			for i, c := range m.Circuits {
				h.W[i], _ = k.compositionParameter([]NodeID{c.Meta.Node}, q)
			}
		}
		m.Heads = append(m.Heads, h)
	}
	return m
}
func nClone(m nModel) nModel {
	out := nModel{MetricEnabled: m.MetricEnabled}
	for _, c := range m.Circuits {
		c.Axis = slices.Clone(c.Axis)
		c.Meta.Parents = slices.Clone(c.Meta.Parents)
		out.Circuits = append(out.Circuits, c)
	}
	for _, h := range m.Heads {
		h.W = slices.Clone(h.W)
		h.Query = slices.Clone(h.Query)
		out.Heads = append(out.Heads, h)
	}
	return out
}
func nError(m nModel, rows []NWitness) float64 {
	v := 0.
	for _, r := range rows {
		d := nPredict(m, r.X, r.Query) - r.Outcome
		v += d * d
	}
	if len(rows) == 0 {
		return 0
	}
	return v / float64(len(rows))
}

// Retention protects previously accepted, correct distinctions and aggregate
// observed error per query. Requiring every wrong or uncertain residual to
// improve individually freezes an imperfect model and is not understanding.
func nRegression(before, after nModel, rows []NWitness) int {
	type aggregate struct {
		query []NodeID
		a, b  float64
		n     int
	}
	groups := []aggregate{}
	violations := 0
	for _, r := range rows {
		av, bv := nPredict(before, r.X, r.Query), nPredict(after, r.X, r.Query)
		a, b := av-r.Outcome, bv-r.Outcome
		idx := -1
		for i, g := range groups {
			if slices.Equal(g.query, r.Query) {
				idx = i
				break
			}
		}
		if idx < 0 {
			groups = append(groups, aggregate{query: r.Query})
			idx = len(groups) - 1
		}
		groups[idx].a += a * a
		groups[idx].b += b * b
		groups[idx].n++
		hi := nHeadIndex(before, r.Query)
		if hi >= 0 && math.Abs(r.Outcome) == 1 && av*r.Outcome > 0 && math.Abs(av) > before.Heads[hi].Error+nTolerance && bv*r.Outcome <= 0 {
			violations++
		}
	}
	for _, g := range groups {
		if g.b/float64(g.n) > g.a/float64(g.n)+nTolerance {
			violations++
		}
	}
	return violations
}

// Small deterministic ridge solve, k<=32. No external fitted model or cache.
func nFit(m *nModel, q []NodeID, rows []NWitness) bool {
	hi := nHeadIndex(*m, q)
	if hi < 0 {
		m.Heads = append(m.Heads, nHead{Query: slices.Clone(q)})
		hi = len(m.Heads) - 1
	}
	if m.MetricEnabled {
		return nFitMetric(m, hi, rows)
	}
	n := len(m.Circuits) + 1
	a := make([][]float64, n)
	for i := range a {
		a[i] = make([]float64, n+1)
		a[i][i] = 1e-9
	}
	for _, r := range rows {
		v, _ := nCodes(*m, r.X)
		v = append(v, 1)
		for i, x := range v {
			a[i][n] += x * r.Outcome
			for j, y := range v {
				a[i][j] += x * y
			}
		}
	}
	for i := 0; i < n; i++ {
		pivot := i
		for j := i + 1; j < n; j++ {
			if math.Abs(a[j][i]) > math.Abs(a[pivot][i]) {
				pivot = j
			}
		}
		a[i], a[pivot] = a[pivot], a[i]
		if math.Abs(a[i][i]) < 1e-12 {
			return false
		}
		d := a[i][i]
		for j := i; j <= n; j++ {
			a[i][j] /= d
		}
		for k := 0; k < n; k++ {
			if k != i {
				d = a[k][i]
				for j := i; j <= n; j++ {
					a[k][j] -= d * a[i][j]
				}
			}
		}
	}
	h := m.Heads[hi]
	h.W = make([]float64, n-1)
	for i := range h.W {
		if math.Abs(a[i][n]) > 7.99 {
			return false
		}
		h.W[i] = a[i][n]
	}
	if math.Abs(a[n-1][n]) > 7.99 {
		return false
	}
	h.B = a[n-1][n]
	h.Known = true
	m.Heads[hi] = h
	return true
}
func nDelta(axis []float64, rows []NWitness) float64 {
	gap := math.Inf(1)
	for i, a := range rows {
		for _, b := range rows[:i] {
			if math.Abs(a.Outcome-b.Outcome) > nTolerance {
				d := math.Abs(nDot(axis, a.X) - nDot(axis, b.X))
				if d > 1e-9 {
					gap = math.Min(gap, d)
				}
			}
		}
	}
	if math.IsInf(gap, 1) {
		return 1
	}
	return math.Max(1e-9, math.Min(7.99, gap/16))
}

// The center is learned from response-conditioned observations. A new boundary
// along an old geometric axis can carry new information, so algebraic rank or
// cosine overlap cannot veto it. Growth still requires independent checks.
func nCenter(axis []float64, rows []NWitness) float64 {
	mean := 0.
	for _, r := range rows {
		mean += r.Outcome
	}
	mean /= float64(len(rows))
	sum, count := [2]float64{}, [2]int{}
	for _, r := range rows {
		i := 0
		if r.Outcome > mean {
			i = 1
		}
		sum[i] += nDot(axis, r.X)
		count[i]++
	}
	if count[0] == 0 || count[1] == 0 {
		return 0
	}
	return math.Max(-7.99, math.Min(7.99, (sum[0]/float64(count[0])+sum[1]/float64(count[1]))/2))
}
func nRedundant(m nModel, rows []NWitness) bool {
	last := len(m.Circuits) - 1
	if last <= 0 {
		return false
	}
	for i := 0; i < last; i++ {
		same, opposite := true, true
		for _, r := range rows {
			v, _ := nCodes(m, r.X)
			same = same && math.Abs(v[i]-v[last]) <= nTolerance
			opposite = opposite && math.Abs(v[i]+v[last]) <= nTolerance
		}
		if same || opposite {
			return true
		}
	}
	return false
}

func nScalarResolution(values []float64, rows []NWitness) (float64, float64) {
	mean := 0.
	for _, r := range rows {
		mean += r.Outcome
	}
	mean /= float64(len(rows))
	s := [2]float64{}
	count := [2]int{}
	gap := math.Inf(1)
	for i, r := range rows {
		j := 0
		if r.Outcome > mean {
			j = 1
		}
		s[j] += values[i]
		count[j]++
		for k, a := range rows[:i] {
			d := math.Abs(values[i] - values[k])
			if math.Abs(r.Outcome-a.Outcome) > nTolerance && d > 1e-9 {
				gap = math.Min(gap, d)
			}
		}
	}
	center := 0.
	if count[0] > 0 && count[1] > 0 {
		center = (s[0]/float64(count[0]) + s[1]/float64(count[1])) / 2
	}
	if math.IsInf(gap, 1) {
		return center, 1
	}
	return center, math.Max(1e-9, math.Min(7.99, gap/16))
}
func nBalancedWitnesses(rows []NWitness, heads []NHeadTopology, budget int) []NWitness {
	if len(heads) == 0 {
		return nil
	}
	quota := budget / len(heads)
	out := []NWitness{}
	for _, h := range heads {
		group := []NWitness{}
		for _, r := range rows {
			if slices.Equal(r.Query, h.Query) {
				duplicate := slices.ContainsFunc(group, func(a NWitness) bool { return a.Outcome == r.Outcome && slices.Equal(a.X, r.X) })
				if !duplicate {
					group = append(group, r)
				}
			}
		}
		sort.SliceStable(group, func(i, j int) bool { return group[i].Outcome < group[j].Outcome })
		n := min(quota, len(group))
		for i := 0; i < n; i++ {
			j := 0
			if n > 1 {
				j = i * (len(group) - 1) / (n - 1)
			}
			out = append(out, group[j])
		}
	}
	return out
}
func nConflict(rows []NWitness) bool {
	for i, a := range rows {
		for _, b := range rows[:i] {
			if slices.Equal(a.Query, b.Query) && slices.Equal(a.X, b.X) && math.Abs(a.Outcome-b.Outcome) > nTolerance {
				return true
			}
		}
	}
	return false
}
func (k *KnowledgeBase) newNNode(kind, index int, at time.Time) NodeID {
	v := NewNeuralVector([]float64{float64(kind) / 8, float64(index) / 512})
	n := k.createRepresentationPrototypeAt(v, at)
	field := NeuralCompositionResolutionV5
	if k.BrainState.NResolution != nil && k.BrainState.NResolution.Contract == NeuralCompositionMetricV6 {
		field = NeuralCompositionMetricV6
	}
	scope := ProjectionScope{Modality: "internal-numeric", Field: field}
	k.registerProjectionPopulation(ProjectionPopulation{Scope: &scope, Prototype: v.Values, Units: []PopulationUnit{{NodeID: n.ID, Activation: 1}}, Provenance: BootstrapProvenance{Origin: "diagnosed_numeric_resolution", Status: "validated_circuit"}})
	return n.ID
}

// Exact installation of a validated candidate, retaining temporal metadata.
// Using a clipped partial install would invalidate the candidate's checks.
func (k *KnowledgeBase) writeNParameter(from, to []NodeID, value, reliability float64, at time.Time) {
	k.Registry.mu.Lock()
	defer k.Registry.mu.Unlock()
	for _, a := range from {
		node := k.Registry.byID[a]
		if node.Synapses == nil {
			node.Synapses = map[NodeID]SynapseList{}
		}
		for _, b := range to {
			list := node.Synapses[b]
			for _, inh := range []bool{false, true} {
				s := list.FindDynamic(inh)
				if s == nil {
					s = &Synapse{TargetID: b, Inhibitory: inh}
					list = append(list, s)
				}
				target := math.Max(0, value) / compositionScale
				if inh {
					target = math.Max(0, -value) / compositionScale
				}
				delta := target - s.Dynamic.Weight
				s.Dynamic.Weight = target
				s.Dynamic.Eligibility = clamp01(.9*s.Dynamic.Eligibility + math.Abs(delta))
				s.Dynamic.Confidence = math.Min(.95, s.Dynamic.Confidence+.02*reliability*(1-s.Dynamic.Confidence))
				s.Dynamic.Frequency++
				s.Dynamic.LastActivation = at
				s.Dynamic.LastModification = at
				s.Dynamic.LastEligibilityUpdate = at
				syncSynapseLegacyState(s)
			}
			node.Synapses[b] = list
		}
	}
}
func (k *KnowledgeBase) newNQuery(v []float64, at time.Time) []NodeID {
	p := ProjectionPopulation{Scope: cloneProjectionScope(&writtenNumericScope), Prototype: slices.Clone(v)}
	for j := 0; j < 4; j++ {
		node := k.createRepresentationPrototypeAt(projectionPrototype(NewNeuralVector(v), j, 4), at)
		p.Units = append(p.Units, PopulationUnit{NodeID: node.ID, Activation: 1})
	}
	k.registerProjectionPopulation(p)
	return writtenPopulationIDs(p)
}

func (k *KnowledgeBase) learnNResolution(bank [compositionBanks][]NodeID, query []NodeID, e CompositionExposure, at time.Time) error {
	if e.Resolution == nil {
		return errors.New("v5 requires an explicit diagnosed numerical lesson")
	}
	if k.BrainState.NResolution == nil {
		k.BrainState.NResolution = &NResolutionState{Sources: make([]NodeID, nDimensions)}
	}
	state := k.BrainState.NResolution
	queries := [][]NodeID{slices.Clone(query)}
	for _, h := range state.Heads {
		if !slices.ContainsFunc(queries, func(q []NodeID) bool { return slices.Equal(q, h.Query) }) {
			queries = append(queries, slices.Clone(h.Query))
		}
	}
	for _, w := range state.Witnesses {
		if !slices.ContainsFunc(queries, func(q []NodeID) bool { return slices.Equal(q, w.Query) }) {
			queries = append(queries, slices.Clone(w.Query))
		}
	}
	before := k.loadNModel(bank, queries)
	convert := func(rows []NObservation) []NWitness {
		out := []NWitness{}
		for _, o := range rows {
			out = append(out, NWitness{X: nInput(o.Input), Query: slices.Clone(query), Outcome: o.Outcome, EpisodeID: o.EpisodeID})
		}
		return out
	}
	fit, check := convert(e.Resolution.Fit), convert(e.Resolution.Check)
	for _, w := range state.Witnesses {
		if slices.Equal(w.Query, query) {
			fit = append(fit, w)
		}
	}
	d := NDiagnosis{Query: slices.Clone(query), Cause: "insufficient-predictive-evidence", BeforeError: nError(before, check)}
	d.AfterError = d.BeforeError
	if nConflict(append(slices.Clone(fit), check...)) {
		d.Cause = "conflicting-observations"
		if query[0] != 0 {
			err := 0.
			for _, r := range append(slices.Clone(fit), check...) {
				err = math.Max(err, math.Abs(nPredict(before, r.X, query)-r.Outcome))
			}
			k.writeNParameter(bank[26], query, math.Min(7.99, err), e.Reliability, at)
		}
		state.Last = d
		k.recordNDiagnosis(d, at)
		return nil
	}
	best := nClone(before)
	bestErr := d.BeforeError
	consider := func(kind string, feature NodeID, m nModel) {
		valid := nFit(&m, query, fit)
		err := nError(m, check)
		reg := nRegression(before, m, state.Witnesses)
		redundant := len(m.Circuits) > len(before.Circuits) && nRedundant(m, append(slices.Clone(fit), check...))
		eligible := valid && !redundant && reg == 0 && err+nTolerance < bestErr
		d.Trials = append(d.Trials, NTrial{Kind: kind, Feature: feature, CheckError: err, RetentionRegressions: reg, Eligible: eligible})
		if eligible {
			best = m
			bestErr = err
			d.Cause = kind
		}
	}
	consider("readout", 0, nClone(before))
	for i, c := range before.Circuits {
		if c.Meta.Kind == 1 {
			delta := nDelta(c.Axis, fit)
			if delta < c.Delta/2 {
				m := nClone(before)
				m.Circuits[i].Delta = delta
				consider("resolution", c.Meta.Node, m)
			}
		}
	}
	if len(before.Circuits) < maxNFeatures {
		axis := make([]float64, nDimensions)
		meanX := make([]float64, nDimensions)
		for _, r := range fit {
			for j, x := range r.X {
				meanX[j] += x / float64(len(fit))
			}
		}
		for _, r := range fit {
			residual := r.Outcome - nPredict(best, r.X, r.Query)
			for j, x := range r.X {
				axis[j] += residual * (x - meanX[j])
			}
		}

		norm := math.Sqrt(nDot(axis, axis))
		if norm > 1e-9 {
			for j := range axis {
				axis[j] /= norm
			}
			m := nClone(before)
			center := nCenter(axis, fit)
			m.Circuits = append(m.Circuits, nCircuit{Meta: NFeature{Kind: 1}, Axis: axis, Delta: nDelta(axis, fit), Center: center})
			for i := range m.Heads {
				m.Heads[i].W = append(m.Heads[i].W, 0)
			}
			kind := "missing-direction"
			for _, old := range before.Circuits {
				if old.Meta.Kind == 1 {
					parallel := math.Abs(nDot(old.Axis, axis))
					if math.Abs(parallel-1) < 1e-8 {
						kind = "missing-boundary"
					}
				}
			}
			consider(kind, 0, m)
		}
		type product struct {
			a, b  int
			score float64
		}
		products := []product{}
		productValues := make([][]float64, len(fit))
		productResidual := make([]float64, len(fit))
		for i, r := range fit {
			productValues[i], _ = nCodes(before, r.X)
			productResidual[i] = r.Outcome - nPredict(best, r.X, r.Query)
		}
		for i := range before.Circuits {
			for j := i; j < len(before.Circuits); j++ {
				parents := []NodeID{before.Circuits[i].Meta.Node, before.Circuits[j].Meta.Node}
				duplicate := slices.ContainsFunc(before.Circuits, func(c nCircuit) bool { return c.Meta.Kind == 2 && slices.Equal(c.Meta.Parents, parents) })
				if duplicate {
					continue
				}
				score := 0.
				for row, v := range productValues {
					score += productResidual[row] * v[i] * v[j]
				}
				products = append(products, product{i, j, math.Abs(score)})
			}
		}
		sort.SliceStable(products, func(i, j int) bool { return products[i].score > products[j].score })
		for _, p := range products[:min(4, len(products))] {
			if p.score < 1e-9 {
				continue
			}
			m := nClone(before)
			raw := make([]float64, len(fit))
			for i, v := range productValues {
				raw[i] = v[p.a] * v[p.b]
			}
			center, delta := nScalarResolution(raw, fit)
			m.Circuits = append(m.Circuits, nCircuit{Meta: NFeature{Kind: 2, Parents: []NodeID{before.Circuits[p.a].Meta.Node, before.Circuits[p.b].Meta.Node}}, Delta: delta, Center: center})
			for i := range m.Heads {
				m.Heads[i].W = append(m.Heads[i].W, 0)
			}
			consider("missing-relation", 0, m)
		}
	}
	if bestErr+nTolerance < d.BeforeError {
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
					if math.Abs(w) > 1e-12 {
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
		k.writeNParameter(bank[24], query, head.B, e.Reliability, at)
		k.writeNParameter(bank[26], query, math.Sqrt(bestErr), e.Reliability, at)
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
			state.Witnesses = nBalancedWitnesses(state.Witnesses, state.Heads, maxNWitnesses)
		}
	}
	state.Last = d
	k.recordNDiagnosis(d, at)
	return nil
}
func (k *KnowledgeBase) recordNDiagnosis(d NDiagnosis, at time.Time) {
	for _, p := range []*map[string]float64{&k.BrainState.PredictionState, &k.BrainState.CuriosityState, &k.BrainState.PlasticityState} {
		if *p == nil {
			*p = map[string]float64{}
		}
	}
	s := k.BrainState.PredictionState
	s["error_n_before"] = d.BeforeError
	s["error_n_after"] = d.AfterError
	s["n_diagnoses"]++
	s["n_committed"] = 0
	if d.Committed {
		s["n_committed"] = 1
	}
	s["last_error"] = math.Min(1, math.Sqrt(d.AfterError))
	s["last_update_unix"] = float64(at.UnixNano())
	k.BrainState.CuriosityState["n_unresolved_error"] = s["last_error"]
	k.BrainState.PlasticityState["n_validated_gain"] = math.Max(0, d.BeforeError-d.AfterError)
	k.BrainState.InquiryState.Sequence++
	k.BrainState.InquiryState.LastUpdated = at
	k.BrainState.InquiryState.OutcomePredictionError = s["last_error"]
	k.BrainState.InquiryState.Pending = !d.Committed && d.AfterError > nTolerance
}
func (k *KnowledgeBase) readNResolution(bank [compositionBanks][]NodeID, query []NodeID, in CompositionInput) CompositionReadout {
	m := k.loadNModel(bank, [][]NodeID{query})
	if m.MetricEnabled {
		return k.readNMetric(m, query, in)
	}
	v := nPredict(m, nInput(in), query)
	values, codes := nCodes(m, nInput(in))
	out := CompositionReadout{Status: "untrained-n-resolution", SceneAt: in.Scene.ObservedAt, Coordinates: []float64{0, 0, 0, 0, 4 * v, 4}, NResolution: &NResolutionReadout{Value: v}}
	if len(m.Heads) == 0 || !m.Heads[0].Known {
		return out
	}
	err := m.Heads[0].Error
	out.NResolution.ObservedCheckRMSE = err
	for i, c := range m.Circuits {
		out.NResolution.Units = append(out.NResolution.Units, NValue{Node: c.Meta.Node, Code: codes[i], Resolution: c.Delta, Contribution: m.Heads[0].W[i] * values[i]})
	}
	out.Confidence = math.Max(0, math.Min(1, math.Abs(v)-err))
	out.Accepted = math.Abs(v) > err+nTolerance
	out.Status = "numeric-prediction"
	if !out.Accepted {
		out.Status = "uncertain-numerical-state"
	}
	return out
}
