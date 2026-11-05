package knowledge

// V2 extends V1's shared GRU, canonical synapses, optimizer and journal. The
// finite world decoder is trained on VERIFIED outcomes, never adapter beliefs.
// BMPS eqs.2-5 and its GP/EI optimizer are reused unchanged. MUSE uses its
// published realizable-rollout option (a), not unconstrained latent ascent.
import (
	"errors"
	"math"
	"slices"
	"time"
)

const CognitiveModelV2 = "internal-shared-gru-muse-bmps-v2"

type NeuralMetaConfig struct {
	Classes          int     `json:"classes"`
	Computations     int     `json:"computations"`
	Outcomes         int     `json:"outcomes"`
	ContextOffset    int     `json:"context_offset"`
	ContextSize      int     `json:"context_size"`
	ResultOffset     int     `json:"result_offset"`
	OperationOffset  int     `json:"operation_offset"`
	ActionOffset     int     `json:"action_offset"`
	HypothesisOffset int     `json:"hypothesis_offset"`
	PlanMarker       int     `json:"plan_marker"`
	StepIndex        int     `json:"step_index"`
	JointWeight      float64 `json:"joint_weight"`
	RewardWeight     float64 `json:"reward_weight"`
	PlanDepth        int     `json:"plan_depth"`
}

func (n NeuralMetaConfig) States() int {
	v := n.Classes
	for i := 0; i < n.Computations; i++ {
		v *= n.Outcomes
	}
	return v
}
func (n NeuralMetaConfig) Validate(c CognitiveModelConfig) error {
	if n.Classes < 2 || n.Classes > 16 || n.Computations < 1 || n.Computations > 8 || n.Outcomes < 2 || n.Outcomes > 8 || c.Actions != n.Computations+n.Classes+1 || n.PlanDepth < 1 || n.PlanDepth > 3 {
		return errors.New("invalid learned metalevel dimensions")
	}
	states := n.Classes
	for i := 0; i < n.Computations; i++ {
		if states > 4096/n.Outcomes {
			return errors.New("finite metalevel allocation budget exceeded")
		}
		states *= n.Outcomes
	}
	if !finiteNumber(n.JointWeight) || n.JointWeight <= 0 || !finiteNumber(n.RewardWeight) || n.RewardWeight <= 0 {
		return errors.New("invalid world learning weights")
	}
	seen := map[int]bool{}
	for _, span := range [][2]int{{n.ContextOffset, n.ContextSize}, {n.ResultOffset, n.Outcomes}, {n.OperationOffset, n.Computations}, {n.ActionOffset, c.Actions}, {n.HypothesisOffset, n.Classes}, {n.PlanMarker, 1}, {n.StepIndex, 1}} {
		if span[0] < 0 || span[1] < 1 || span[0] > c.Inputs-span[1] {
			return errors.New("invalid native metalevel feature layout")
		}
		for j := 0; j < span[1]; j++ {
			id := span[0] + j
			if seen[id] {
				return errors.New("overlapping native metalevel channels")
			}
			seen[id] = true
		}
	}
	return nil
}

type NeuralMetaState struct {
	JointNodes          []NodeID           `json:"joint_nodes"`
	RewardNodes         []NodeID           `json:"reward_nodes"`
	TemperatureNodes    []NodeID           `json:"temperature_nodes"`
	Joint               CognitiveTensor    `json:"joint"`
	JointBias           CognitiveTensor    `json:"joint_bias"`
	Reward              CognitiveTensor    `json:"reward"`
	RewardBias          CognitiveTensor    `json:"reward_bias"`
	CalibrationEvidence *CognitiveEvidence `json:"calibration_evidence,omitempty"`
}
type NeuralMetaInput struct {
	Diagnostics       bool        `json:"diagnostics,omitempty"`
	Greedy            bool        `json:"greedy,omitempty"`
	Candidate         *[3]float64 `json:"candidate,omitempty"` // explicit GP/EI experiment point; never changes neural parameters
	Mode              string      `json:"mode"`                // recurrent, muse, bmps, muse-bmps, adapter-bmps
	Costs             []float64   `json:"costs"`               // observable scheduler charges, not an outcome model
	Remaining         int         `json:"remaining"`
	NoFeedback        bool        `json:"no_feedback,omitempty"`
	UniformBelief     bool        `json:"uniform_belief,omitempty"` // explicit decoder ablation
	UniformLikelihood bool        `json:"uniform_likelihood,omitempty"`
}

func (x NeuralMetaInput) Validate(c CognitiveModelConfig) error {
	n := c.NeuralMeta
	if n == nil {
		return errors.New("native metalevel input requires V2")
	}
	switch x.Mode {
	case "recurrent", "muse", "bmps", "muse-bmps", "adapter-bmps":
	default:
		return errors.New("unknown native selector mode")
	}
	if len(x.Costs) != n.Computations || x.Remaining < 0 || x.Remaining > c.Horizon {
		return errors.New("invalid native scheduler contract")
	}
	for _, cost := range x.Costs {
		if !finiteNumber(cost) || cost <= 0 {
			return errors.New("invalid observable computation charge")
		}
	}
	if x.Candidate != nil {
		if x.Mode != "bmps" && x.Mode != "muse-bmps" {
			return errors.New("candidate weights require BMPS")
		}
		for _, v := range *x.Candidate {
			if !finiteNumber(v) || v < 0 || v > 1 {
				return errors.New("invalid GP/EI candidate point")
			}
		}
	}
	return nil
}
func metaTarget(n NeuralMetaConfig, state int) int { return state % n.Classes }
func metaOutcome(n NeuralMetaConfig, state, op int) int {
	state /= n.Classes
	for i := 0; i < op; i++ {
		state /= n.Outcomes
	}
	return state % n.Outcomes
}
func metaSoftmax(z []float64, temp float64) []float64 {
	p := make([]float64, len(z))
	m := math.Inf(-1)
	for _, v := range z {
		m = math.Max(m, v/temp)
	}
	s := 0.0
	for i, v := range z {
		p[i] = math.Exp(v/temp - m)
		s += p[i]
	}
	for i := range p {
		p[i] /= s
	}
	return p
}
func (k *KnowledgeBase) metaTemperatures(s *CognitiveModelState) []float64 {
	t := make([]float64, 6)
	for i, id := range s.NeuralMeta.TemperatureNodes {
		v, _ := k.compositionParameter([]NodeID{s.Layout.BiasInput}, []NodeID{id})
		t[i] = math.Exp(v)
	}
	return t
}
func metaDistribution(s *CognitiveModelState, w, h []float64, temp float64, uniform bool) []float64 {
	z := mat(w, s.NeuralMeta.Joint, h)
	addBias(z, w, s.NeuralMeta.JointBias)
	if uniform {
		clear(z)
	}
	return metaSoftmax(z, temp)
}
func metaPlanFeatures(c CognitiveModelConfig, x []float64, target, action int) []float64 {
	n := c.NeuralMeta
	out := append([]float64{}, x...)
	clear(out[n.ResultOffset : n.ResultOffset+n.Outcomes])
	clear(out[n.OperationOffset : n.OperationOffset+n.Computations])
	clear(out[n.ActionOffset : n.ActionOffset+c.Actions])
	clear(out[n.HypothesisOffset : n.HypothesisOffset+n.Classes])
	out[n.ActionOffset+action] = 1
	out[n.HypothesisOffset+target] = 1
	out[n.PlanMarker] = 1
	return out
}
func metaPlan(s *CognitiveModelState, w, h, x []float64, target, action int, temps []float64) ([]float64, float64) {
	f := cognitiveGRU(s.Config, s.Layout, w, metaPlanFeatures(s.Config, x, target, action), h)
	sa := mat(w, s.Layout.Assessment, f.h)
	addBias(sa, w, s.Layout.AssessmentBias)
	for i := range sa {
		sa[i] = sigmoid(sa[i] / temps[i+1])
	}
	reward := mat(w, s.NeuralMeta.Reward, f.h)
	addBias(reward, w, s.NeuralMeta.RewardBias)
	return sa, math.Tanh(reward[0])
}
func metaClassBelief(n NeuralMetaConfig, joint []float64) []float64 {
	p := make([]float64, n.Classes)
	for state, v := range joint {
		p[metaTarget(n, state)] += v
	}
	return p
}
func metaUtilities(s *CognitiveModelState, w, h, x, p, temps []float64, useMUSE bool) ([][]float64, [][]float64) {
	n := *s.Config.NeuralMeta
	u := make([][]float64, n.Classes)
	competence := make([][]float64, n.Classes)
	for y := 0; y < n.Classes; y++ {
		u[y] = make([]float64, n.Classes+1)
		competence[y] = make([]float64, n.Classes+1)
		for a := 0; a <= n.Classes; a++ {
			sa, reward := metaPlan(s, w, h, x, y, n.Computations+a, temps)
			competence[y][a] = sa[4]
			u[y][a] = reward
			if useMUSE && a < n.Classes {
				u[y][a] = sa[4]
			}
		}
	}
	return u, competence
}
func metaBelief(s *CognitiveModelState, joint, classes []float64, utilities [][]float64, input CognitiveInput) (MetaBelief, []int, error) {
	return metaBeliefSupport(s, joint, classes, utilities, input, false)
}

// Observed Bayesian conditioning creates exact zero-probability tuples. Their
// utility rows are immaterial and may be zero; legacy unconditioned callers
// retain their original strict support guard.
func metaBeliefSupport(s *CognitiveModelState, joint, classes []float64, utilities [][]float64, input CognitiveInput, observedZeros bool) (MetaBelief, []int, error) {
	n := *s.Config.NeuralMeta
	// The metalevel parameter is the tuple of observable computation results.
	// Irreducible goal noise is integrated into utility, NOT revealed by VPI.
	// Both tuple probabilities and conditional goal probabilities come from N.
	tuples := len(joint) / n.Classes
	b := MetaBelief{Probabilities: make([]float64, tuples), Utilities: make([][]float64, tuples), Horizon: s.Config.Horizon, Remaining: input.NeuralMeta.Remaining}
	for theta := 0; theta < tuples; theta++ {
		u := make([]float64, n.Classes+1)
		mass := 0.0
		for y := 0; y < n.Classes; y++ {
			p := joint[theta*n.Classes+y]
			mass += p
			for a := range u {
				u[a] += p * utilities[y][a]
			}
		}
		if mass <= 0 {
			if mass == 0 && observedZeros {
				b.Utilities[theta] = u
				continue
			}
			return b, nil, errors.New("neural tuple probability underflow; no oracle fallback")
		}
		for a := range u {
			u[a] /= mass
		}
		b.Probabilities[theta] = mass
		b.Utilities[theta] = u
	}
	mapping := []int{}
	for op := 0; op < n.Computations; op++ {
		if !input.Mask[op] {
			continue
		}
		likelihood := make([][]float64, tuples)
		groups := make([]int, tuples)
		for theta := 0; theta < tuples; theta++ {
			o := metaOutcome(n, theta*n.Classes, op)
			likelihood[theta] = make([]float64, n.Outcomes)
			likelihood[theta][o] = 1
			groups[theta] = o
		}
		if input.NeuralMeta.UniformLikelihood {
			for y := range likelihood {
				for o := range likelihood[y] {
					likelihood[y][o] = 1 / float64(n.Outcomes)
				}
			}
		}
		b.Computations = append(b.Computations, MetaComputation{Cost: input.NeuralMeta.Costs[op], Likelihood: likelihood, Groups: groups})
		mapping = append(mapping, op)
	}
	return b, mapping, b.Validate()
}
func metaNextFeatures(c CognitiveModelConfig, x []float64, op, outcome int, noFeedback bool) []float64 {
	n := c.NeuralMeta
	next := append([]float64{}, x...)
	// The public task/scheduler context remains visible in the environment.
	clear(next[n.ResultOffset : n.ResultOffset+n.Outcomes])
	clear(next[n.OperationOffset : n.OperationOffset+n.Computations])
	clear(next[n.ActionOffset : n.ActionOffset+c.Actions])
	clear(next[n.HypothesisOffset : n.HypothesisOffset+n.Classes])
	next[n.PlanMarker] = 0
	next[n.ActionOffset+op] = 1
	next[n.StepIndex] = math.Min(1, next[n.StepIndex]+1/float64(c.Horizon))
	if !noFeedback {
		next[n.ResultOffset+outcome] = 1
		next[n.OperationOffset+op] = 1
	}
	return next
}

// Published MUSE option (a): evaluate executable future observation branches
// generated by the learned decoder. No free latent vector is optimized and no
// simulated value is committed as a verified observation.
func metaMUSEValue(s *CognitiveModelState, w, h, x, temps []float64, mask []bool, noRec, noFeedback bool, depth int) (float64, int) {
	n := *s.Config.NeuralMeta
	joint := metaDistribution(s, w, h, temps[0], false)
	p := metaClassBelief(n, joint)
	_, q := metaUtilities(s, w, h, x, p, temps, true)
	best, a := -1.0, -1
	for answer := 0; answer < n.Classes; answer++ {
		id := n.Computations + answer
		if !mask[id] {
			continue
		}
		v := 0.0
		for y, py := range p {
			v += py * q[y][answer]
		}
		if v > best {
			best = v
			a = id
		}
	}
	if depth <= 0 {
		return best, a
	}
	for op := 0; op < n.Computations; op++ {
		if !mask[op] {
			continue
		}
		prob := make([]float64, n.Outcomes)
		for state, v := range joint {
			prob[metaOutcome(n, state, op)] += v
		}
		expected := 0.0
		nextMask := append([]bool{}, mask...)
		nextMask[op] = false
		for o, po := range prob {
			xx := metaNextFeatures(s.Config, x, op, o, noFeedback)
			hp := h
			if noRec {
				hp = make([]float64, len(h))
			}
			hh := cognitiveGRU(s.Config, s.Layout, w, xx, hp).h
			v, _ := metaMUSEValue(s, w, hh, xx, temps, nextMask, noRec, noFeedback, depth-1)
			expected += po * v
		}
		if expected > best {
			best = expected
			a = op
		}
	}
	return best, a
}
func (k *KnowledgeBase) cognitiveForwardV2(x CognitiveInput, s *CognitiveModelState, w []float64) (CognitiveReadout, error) {
	c, l := s.Config, s.Layout
	n := *c.NeuralMeta
	features := append([]float64{}, x.Features...)
	if x.NeuralMeta.NoFeedback {
		clear(features[n.ResultOffset : n.ResultOffset+n.Outcomes])
		clear(features[n.OperationOffset : n.OperationOffset+n.Computations])
	}
	hp := k.cognitiveHidden(s)
	if x.Reset || x.NoRecurrence {
		hp = make([]float64, c.Hidden)
	}
	h := cognitiveGRU(c, l, w, features, hp).h
	actor, _, v := cognitiveHeads(l, w, h, x.Mask)
	temps := k.metaTemperatures(s)
	joint := metaDistribution(s, w, h, temps[0], x.NeuralMeta.UniformBelief)
	p := metaClassBelief(n, joint)
	muse := x.NeuralMeta.Mode == "muse" || x.NeuralMeta.Mode == "muse-bmps"
	var u, q [][]float64
	if x.NeuralMeta.Mode != "recurrent" || x.NeuralMeta.Diagnostics {
		u, q = metaUtilities(s, w, h, features, p, temps, muse)
	}
	out := CognitiveReadout{Hidden: h, Policy: actor, Value: v, ParametersHash: k.cognitiveParameterDigest(w), ClassBelief: p, JointBelief: joint, Success: make([]float64, 5), AssessmentMeaning: "probability of selected immediate commit; on mental steps best immediate commit"}
	if x.NeuralMeta.Diagnostics {
		out.TerminalCompetence = q
		out.TerminalReward = make([][]float64, n.Classes)
		for y := 0; y < n.Classes; y++ {
			out.TerminalReward[y] = make([]float64, n.Classes+1)
			for a := 0; a <= n.Classes; a++ {
				_, out.TerminalReward[y][a] = metaPlan(s, w, h, features, y, n.Computations+a, temps)
			}
		}
	}
	out.Action = samplePolicy(actor, x.Draw)
	if x.NeuralMeta.Greedy {
		out.Action = maxIndex(actor)
	}
	switch x.NeuralMeta.Mode {
	case "muse":
		_, a := metaMUSEValue(s, w, h, features, temps, x.Mask, x.NoRecurrence, x.NeuralMeta.NoFeedback, n.PlanDepth)
		if a >= 0 {
			out.Action = a
		}
	case "bmps", "muse-bmps", "adapter-bmps":
		b, mapping, err := metaBelief(s, joint, p, u, x)
		if err != nil {
			return out, err
		}
		if x.NeuralMeta.Mode == "adapter-bmps" {
			if x.Belief == nil {
				return out, errors.New("explicit model-known comparator requires a supplied belief")
			}
			b = *x.Belief
			mapping = []int{}
			for op := 0; op < n.Computations; op++ {
				if x.Mask[op] {
					mapping = append(mapping, op)
				}
			}
			if len(b.Computations) != len(mapping) {
				return out, errors.New("adapter comparator action mismatch")
			}
		}
		weights := k.cognitiveBMPSWeights(s)
		if x.NeuralMeta.Candidate != nil {
			weights = BMPSWeights(*x.NeuralMeta.Candidate, c.Horizon)
		}
		d, err := SelectBMPS(b, weights)
		if err != nil {
			return out, err
		}
		out.BMPS = &d
		if d.Computation >= 0 {
			out.Action = mapping[d.Computation]
		} else {
			out.Action = n.Computations + d.TerminalAction
		}
		out.NativeLikelihood = b.Computations
	}
	if !x.Mask[out.Action] {
		return out, errors.New("native selector chose an invalid action")
	}
	answer := out.Action - n.Computations
	if answer < 0 || answer >= n.Classes {
		answer = maxIndex(p)
		if q != nil {
			scores := make([]float64, n.Classes)
			for y, py := range p {
				for a := range scores {
					scores[a] += py * q[y][a]
				}
			}
			answer = maxIndex(scores)
		}
	}
	if answer < n.Classes {
		for y, py := range p {
			sa, _ := metaPlan(s, w, h, features, y, n.Computations+answer, temps)
			for j := range out.Success {
				out.Success[j] += py * sa[j]
			}
		}
	}
	return out, nil
}

// Exact-array memoization is ONLY for the SHA receipt. Inference always uses
// a freshly decoded canonical synapse array. Nothing is trained in this cache.
func (k *KnowledgeBase) cognitiveParameterDigest(w []float64) string {
	k.cognitiveDigestMu.Lock()
	defer k.cognitiveDigestMu.Unlock()
	if !slices.Equal(w, k.cognitiveDigestWeights) {
		k.cognitiveDigestWeights = append(k.cognitiveDigestWeights[:0], w...)
		k.cognitiveDigestValue = CognitiveDigest(w)
	}
	return k.cognitiveDigestValue
}
func (k *KnowledgeBase) InitializeNeuralMetaExtension(s *CognitiveModelState, at time.Time) {
	n := s.Config.NeuralMeta
	ms := &NeuralMetaState{}
	coordinate := len(s.Layout.Inputs) + len(s.Layout.Hidden) + len(s.Layout.Gates) + len(s.Layout.Actions) + len(s.Layout.Success) + len(s.Layout.Value) + len(s.Layout.World) + 6
	alloc := func(count int) []NodeID {
		ids := make([]NodeID, count)
		for i := range ids {
			node := k.createRepresentationPrototypeAt(NewNeuralVector([]float64{float64(coordinate), 1}), at)
			node.Activation = .5
			node.Importance = 1
			ids[i] = node.ID
			scope := cognitiveScope
			k.projectionMu.Lock()
			k.ProjectionPopulations = append(k.ProjectionPopulations, ProjectionPopulation{Scope: &scope, Prototype: []float64{float64(coordinate), 1}, Units: []PopulationUnit{{NodeID: node.ID, Activation: .5}}})
			k.projectionMu.Unlock()
			coordinate++
		}
		return ids
	}
	ms.JointNodes = alloc(n.States())
	ms.RewardNodes = alloc(1)
	ms.TemperatureNodes = alloc(6)
	tensor := func(from, to []NodeID) CognitiveTensor {
		t := CognitiveTensor{Offset: len(s.Layout.Edges), Rows: len(to), Cols: len(from)}
		for _, b := range to {
			for _, a := range from {
				s.Layout.Edges = append(s.Layout.Edges, CognitiveEdge{a, b})
			}
		}
		return t
	}
	ms.Joint = tensor(s.Layout.Hidden, ms.JointNodes)
	ms.JointBias = tensor([]NodeID{s.Layout.BiasInput}, ms.JointNodes)
	ms.Reward = tensor(s.Layout.Hidden, ms.RewardNodes)
	ms.RewardBias = tensor([]NodeID{s.Layout.BiasInput}, ms.RewardNodes)
	s.NeuralMeta = ms
	for _, id := range ms.TemperatureNodes {
		k.adjustCompositionParameter([]NodeID{s.Layout.BiasInput}, []NodeID{id}, 0, 1, at, true)
	}
}
