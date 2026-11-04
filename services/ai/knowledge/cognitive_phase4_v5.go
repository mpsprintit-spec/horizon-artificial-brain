package knowledge

import (
	"errors"
	"math"
	"math/rand"
	"time"
)

const Phase4VersionV5 = "v5-phase4-unified-native-v1"
const Phase4FeaturesV5 = 58
const Phase4ParametersV5 = 1097

type Phase4SpecV5 struct {
	Version                                  string
	Seed                                     int64
	DropMUSE, DropVOC, DropActor             bool
	LearningRate, EntropyWeight, ValueWeight float64
}

func (s Phase4SpecV5) Validate() error {
	if !finiteNumber(s.LearningRate) || !finiteNumber(s.EntropyWeight) || !finiteNumber(s.ValueWeight) || s.Version != Phase4VersionV5 || s.LearningRate <= 0 || s.LearningRate > .01 || s.EntropyWeight < 0 || s.EntropyWeight > .1 || s.ValueWeight <= 0 || s.ValueWeight > 1 {
		return errors.New("invalid explicit V5 Phase IV spec")
	}
	return nil
}

type Phase4StateV5 struct {
	Contract                                     string `json:"coordinated_contract,omitempty"`
	Spec                                         Phase4SpecV5
	BaseParameters                               int
	Features, Hidden, Actions, Value             []NodeID
	WI, BI, Actor, ActorBias, Critic, CriticBias CognitiveTensor
	Steps, LearnedEpisodes                       uint64
	LastEvidence                                 *CognitiveEvidence
}
type Phase4DropV5 struct{ MUSE, VOC, Actor bool }
type Phase4FrameV5 struct {
	Features                                         []float64
	Mask                                             []bool
	Action                                           int
	Reward, Draw, BehaviorProbability, BehaviorValue float64
}
type Phase4EpisodeV5 struct {
	Frames      []Phase4FrameV5
	CaseID      string
	Acquisition *Phase4TerminalAcquisitionV5 `json:"acquisition,omitempty"`
}
type Phase4BatchV5 struct {
	Episodes       []Phase4EpisodeV5
	ParametersHash string
	Evidence       CognitiveEvidence
	Protocol       string `json:"protocol,omitempty"`
}
type Phase4PredictionV5 struct {
	Frozen           Phase2PredictionV5
	Features, Policy []float64
	Value            float64
	Action           int
	ParametersHash   string
	MetapolicyNS     int64
}
type Phase4LearningV5 struct {
	Episodes, Frames                                                                                                                                      int
	Policy, Value, Entropy, Loss, GradientNorm, ActorGradientNorm, CriticGradientNorm, AdvantageMean, AdvantageSD, AdvantageMin, AdvantageMax, MeanReturn float64
	Steps                                                                                                                                                 uint64
	ClippedParameters                                                                                                                                     int
}

func (k *KnowledgeBase) HasPhase4V5() bool {
	k.RLock()
	defer k.RUnlock()
	return k.BrainState.CognitiveModel != nil && k.BrainState.CognitiveModel.Phase4V5 != nil
}
func (k *KnowledgeBase) ValidatePhase4InitV5(spec Phase4SpecV5) error {
	k.RLock()
	defer k.RUnlock()
	s := k.BrainState.CognitiveModel
	if e := spec.Validate(); e != nil {
		return e
	}
	if s == nil || s.Phase2V5 == nil || s.Phase4V5 != nil || s.Config.Hidden != 32 || s.Config.Actions != 8 || s.Config.NeuralMeta.Classes != 4 || s.Config.NeuralMeta.Computations != 3 {
		return errors.New("Phase IV requires authentic 32hidden/8action Phase II/III canonical model")
	}
	return nil
}
func (k *KnowledgeBase) InitializePhase4V5(spec Phase4SpecV5, at time.Time) error {
	if e := k.ValidatePhase4InitV5(spec); e != nil {
		return e
	}
	k.Lock()
	defer k.Unlock()
	s := k.BrainState.CognitiveModel
	p := &Phase4StateV5{Spec: spec, BaseParameters: len(s.Layout.Edges)}
	count := 0
	units := func(n int) []NodeID {
		ids := []NodeID{}
		for i := 0; i < n; i++ {
			node := k.createRepresentationPrototypeAt(NewNeuralVector([]float64{float64(2000000 + count), 1}), at)
			count++
			node.Activation = .5
			node.Importance = 1
			ids = append(ids, node.ID)
			scope := cognitiveScope
			k.projectionMu.Lock()
			k.ProjectionPopulations = append(k.ProjectionPopulations, ProjectionPopulation{Scope: &scope, Prototype: []float64{float64(1999999 + count), 1}, Units: []PopulationUnit{{NodeID: node.ID, Activation: .5}}})
			k.projectionMu.Unlock()
		}
		return ids
	}
	p.Features = units(58)
	p.Hidden = units(16)
	p.Actions = units(8)
	p.Value = units(1)
	tensor := func(from, to []NodeID) CognitiveTensor {
		t := CognitiveTensor{Offset: len(s.Layout.Edges), Rows: len(to), Cols: len(from)}
		for _, b := range to {
			for _, a := range from {
				s.Layout.Edges = append(s.Layout.Edges, CognitiveEdge{a, b})
			}
		}
		return t
	}
	p.WI = tensor(p.Features, p.Hidden)
	p.BI = tensor([]NodeID{s.Layout.BiasInput}, p.Hidden)
	p.Actor = tensor(p.Hidden, p.Actions)
	p.ActorBias = tensor([]NodeID{s.Layout.BiasInput}, p.Actions)
	p.Critic = tensor(p.Hidden, p.Value)
	p.CriticBias = tensor([]NodeID{s.Layout.BiasInput}, p.Value)
	rng := rand.New(rand.NewSource(spec.Seed))
	for i, edge := range s.Layout.Edges[p.BaseParameters:] {
		idx := p.BaseParameters + i
		v := 0.
		if idx >= p.WI.Offset && idx < p.BI.Offset {
			v = rng.NormFloat64() / math.Sqrt(58)
		}
		if idx >= p.Actor.Offset && idx < p.ActorBias.Offset || idx >= p.Critic.Offset && idx < p.CriticBias.Offset {
			v = rng.NormFloat64() * .1 / math.Sqrt(16)
		}
		k.adjustCompositionParameter([]NodeID{edge.From}, []NodeID{edge.To}, v, 1, at, true)
	}
	s.FirstMoment = append(s.FirstMoment, make([]float64, 1097)...)
	s.SecondMoment = append(s.SecondMoment, make([]float64, 1097)...)
	s.Phase4V5 = p
	return nil
}
func phase4DropFeatures(x []float64, d Phase4DropV5) {
	if d.MUSE {
		clear(x[32:42])
	}
	if d.VOC {
		clear(x[42:45])
	}
	if d.Actor {
		clear(x[49:58])
	}
}
func phase4Forward(p *Phase4StateV5, w, x []float64, mask []bool) ([]float64, []float64, float64) {
	h := mat(w, p.WI, x)
	addBias(h, w, p.BI)
	for i := range h {
		h[i] = math.Tanh(h[i])
	}
	logits := mat(w, p.Actor, h)
	addBias(logits, w, p.ActorBias)
	max := -math.MaxFloat64
	for a, v := range logits {
		if mask[a] {
			max = math.Max(max, v)
		}
	}
	z := 0.
	for a := range logits {
		if mask[a] {
			logits[a] = math.Exp(logits[a] - max)
			z += logits[a]
		} else {
			logits[a] = 0
		}
	}
	for a := range logits {
		logits[a] /= z
	}
	v := mat(w, p.Critic, h)
	addBias(v, w, p.CriticBias)
	return h, logits, v[0]
}
func (k *KnowledgeBase) PreviewPhase4V5(req Phase2ProbeV5, drop Phase4DropV5) (Phase4PredictionV5, error) {
	return k.previewPhase4V5(req, drop, nil)
}
func (k *KnowledgeBase) PreviewPhase4DiagnosticV5(req Phase2ProbeV5, drop Phase4DropV5, do Phase3InterventionV5) (Phase4PredictionV5, error) {
	if !do.OfflineOnly {
		return Phase4PredictionV5{}, errors.New("offline diagnostic required")
	}
	return k.previewPhase4V5(req, drop, &do)
}
func (k *KnowledgeBase) previewPhase4V5(req Phase2ProbeV5, drop Phase4DropV5, do *Phase3InterventionV5) (Phase4PredictionV5, error) {
	out := Phase4PredictionV5{}
	if len(req.Trace) == 0 || req.RawTemperature || req.ConditionObserved || req.CategoricalAblation {
		return out, errors.New("candidate uses native frozen temperature, observed history, no oracle conditioning")
	}
	req.Trace = append([]CognitiveInput(nil), req.Trace...)
	for i := range req.Trace {
		if req.Trace[i].NeuralMeta == nil {
			return out, errors.New("missing causal meta input")
		}
		n := *req.Trace[i].NeuralMeta
		n.Diagnostics = true
		n.Mode = "muse-bmps"
		req.Trace[i].NeuralMeta = &n
	}
	var e error
	out.Frozen, e = k.previewPhase3V5(req, do)
	if e != nil {
		return out, e
	}
	k.RLock()
	defer k.RUnlock()
	s := k.BrainState.CognitiveModel
	if s == nil || s.Phase4V5 == nil {
		return out, errors.New("no learned Phase IV N circuit")
	}
	p := s.Phase4V5
	if p.Contract != "" {
		return out, errors.New("use explicit coordinated V5 inference")
	}
	w, e := k.cognitiveWeights(s)
	if e != nil {
		return out, e
	}
	start := time.Now()
	x := make([]float64, 58)
	r := out.Frozen.Readout
	copy(x[:32], r.Hidden)
	for a := 0; a < 5; a++ {
		for y, q := range r.ClassBelief {
			x[32+a] += q * r.TerminalCompetence[y][a]
			x[37+a] += q * r.TerminalReward[y][a]
		}
	}
	last := req.Trace[len(req.Trace)-1]
	j := 0
	for op := 0; op < 3; op++ {
		if last.Mask[op] {
			x[42+op] = r.BMPS.VOC[j]
			j++
		}
		x[45+op] = last.NeuralMeta.Costs[op]
	}
	x[48] = float64(last.NeuralMeta.Remaining)
	actor := mat(w, s.Layout.Actor, r.Hidden)
	addBias(actor, w, s.Layout.ActorBias)
	copy(x[49:57], actor)
	x[57] = r.Value
	phase4DropFeatures(x, Phase4DropV5{p.Spec.DropMUSE || drop.MUSE, p.Spec.DropVOC || drop.VOC, p.Spec.DropActor || drop.Actor})
	_, out.Policy, out.Value = phase4Forward(p, w, x, last.Mask)
	out.Action = samplePolicy(out.Policy, last.Draw)
	if last.NeuralMeta.Greedy {
		out.Action = maxIndex(out.Policy)
	}
	out.Features = x
	out.ParametersHash = k.cognitiveParameterDigest(w)
	out.MetapolicyNS = time.Since(start).Nanoseconds()
	return out, nil
}
func (k *KnowledgeBase) ValidatePhase4BatchV5(b Phase4BatchV5) error {
	if k.HasCoordinatedV5() {
		return errors.New("coordinated state cannot accept historical head-only updates")
	}
	k.RLock()
	defer k.RUnlock()
	s := k.BrainState.CognitiveModel
	if s == nil || s.Phase4V5 == nil {
		return errors.New("missing Phase IV")
	}
	w, e := k.cognitiveWeights(s)
	if e != nil {
		return e
	}
	if b.ParametersHash != CognitiveDigest(w) || len(b.Episodes) < 1 || len(b.Episodes) > 64 {
		return errors.New("not current on-policy batch")
	}
	if e = b.Evidence.Validate(); e != nil {
		return e
	}
	if b.Evidence.Digest != CognitiveDigest(b.Episodes) {
		return errors.New("outcome evidence digest differs")
	}
	if b.Protocol != "" && b.Protocol != Phase4TerminalProtocolV5 {
		return errors.New("unknown Phase IV training protocol")
	}
	if b.Protocol == Phase4TerminalProtocolV5 {
		return validatePhase4TerminalBatchV5(s, w, b)
	}
	for _, ep := range b.Episodes {
		if ep.Acquisition != nil {
			return errors.New("acquisition requires explicit terminal protocol")
		}
		if ep.CaseID == "" || len(ep.Frames) < 1 || len(ep.Frames) > 4 {
			return errors.New("invalid episode")
		}
		used := [3]bool{}
		for t, f := range ep.Frames {
			if len(f.Features) != 58 || len(f.Mask) != 8 || f.Action < 0 || f.Action > 7 || !f.Mask[f.Action] || !finiteNumber(f.Reward) || f.Draw < 0 || f.Draw >= 1 || !finiteNumber(f.Draw) {
				return errors.New("invalid frame")
			}
			for _, v := range f.Features {
				if !finiteNumber(v) {
					return errors.New("nonfinite feature")
				}
			}
			for op := 0; op < 3; op++ {
				if f.Mask[op] != (!used[op] && t < 3) {
					return errors.New("nonphysical mask")
				}
			}
			for a := 3; a < 8; a++ {
				if !f.Mask[a] {
					return errors.New("terminal mask cannot force answering")
				}
			}
			if f.Features[48] != float64(3-t) {
				return errors.New("budget not causal")
			}
			x := append([]float64(nil), f.Features...)
			phase4DropFeatures(x, Phase4DropV5{s.Phase4V5.Spec.DropMUSE, s.Phase4V5.Spec.DropVOC, s.Phase4V5.Spec.DropActor})
			if CognitiveDigest(x) != CognitiveDigest(f.Features) {
				return errors.New("ablated training features present")
			}
			_, prob, v := phase4Forward(s.Phase4V5, w, f.Features, f.Mask)
			if samplePolicy(prob, f.Draw) != f.Action || math.Abs(prob[f.Action]-f.BehaviorProbability) > 1e-12 || math.Abs(v-f.BehaviorValue) > 1e-12 {
				return errors.New("off-policy trajectory or value mismatch")
			}
			if t < len(ep.Frames)-1 {
				if f.Action >= 3 || f.Reward != -f.Features[45+f.Action] {
					return errors.New("nonobserved mental outcome/cost")
				}
				used[f.Action] = true
			} else if f.Action < 3 || f.Reward != 0 && f.Reward != 1 && f.Reward != .35 || f.Action == 7 && f.Reward != .35 || f.Action != 7 && f.Reward == .35 {
				return errors.New("invalid observed terminal reward")
			}
		}
	}
	return nil
}

// Returns and advantages are detached exactly as in native recurrent actor-critic.
// Frozen 58dim feature tensors receive no gradient; only these six new N tensors do.
func phase4Objective(p *Phase4StateV5, w []float64, b Phase4BatchV5, advantages, returns [][]float64) (Phase4LearningV5, []float64) {
	return phase4ExplorationObjective(p, w, b, advantages, returns, nil)
}

func phase4BehaviorPolicy(prob []float64, mask []bool, exploration float64) []float64 {
	if exploration == 0 {
		return prob
	}
	legal := 0
	for _, allowed := range mask {
		if allowed {
			legal++
		}
	}
	floor := exploration / float64(legal)
	behavior := make([]float64, len(prob))
	for a, allowed := range mask {
		if allowed {
			behavior[a] = (1-exploration)*prob[a] + floor
		}
	}
	return behavior
}

func phase4ExplorationObjective(p *Phase4StateV5, w []float64, b Phase4BatchV5, advantages, returns, exploration [][]float64) (Phase4LearningV5, []float64) {
	return phase4WeightedObjective(p, w, b, advantages, returns, exploration, nil)
}

type phase4VisitationWeights struct {
	before, through [][]float64
}

func phase4WeightedObjective(p *Phase4StateV5, w []float64, b Phase4BatchV5, advantages, returns, exploration [][]float64, weights *phase4VisitationWeights) (Phase4LearningV5, []float64) {
	st := Phase4LearningV5{Episodes: len(b.Episodes), AdvantageMin: math.Inf(1), AdvantageMax: math.Inf(-1)}
	g := make([]float64, len(w))
	norm := 1. / float64(len(b.Episodes))
	advSum, advSq := 0., 0.
	for ei, ep := range b.Episodes {
		for t, f := range ep.Frames {
			h, prob, v := phase4Forward(p, w, f.Features, f.Mask)
			adv, ret := advantages[ei][t], returns[ei][t]
			weight, entropyWeight := 1., 1.
			if weights != nil {
				weight, entropyWeight = weights.through[ei][t], weights.before[ei][t]
			}
			behavior, policyAdv := prob, adv
			if exploration != nil && exploration[ei][t] != 0 {
				eps := exploration[ei][t]
				behavior = phase4BehaviorPolicy(prob, f.Mask, eps)
				// Differentiate log q for q=(1-eps)*pi+eps*uniform.
				policyAdv *= (1 - eps) * prob[f.Action] / behavior[f.Action]
			}
			policyAdv *= weight
			st.Frames++
			advSum += adv
			advSq += adv * adv
			st.AdvantageMin = math.Min(st.AdvantageMin, adv)
			st.AdvantageMax = math.Max(st.AdvantageMax, adv)
			st.MeanReturn += ret
			st.Policy -= norm * weight * adv * math.Log(math.Max(1e-15, behavior[f.Action]))
			el := 0.
			for _, q := range prob {
				if q > 0 {
					el += q * math.Log(q)
				}
			}
			st.Entropy += norm * entropyWeight * el
			dy := make([]float64, 8)
			for a, q := range prob {
				if q > 0 {
					dy[a] = norm * (policyAdv*q + entropyWeight*p.Spec.EntropyWeight*q*(math.Log(q)-el))
					if a == f.Action {
						dy[a] -= norm * policyAdv
					}
				}
			}
			dh := make([]float64, 16)
			cognitiveLinearBackward(g, dh, w, h, dy, p.Actor, p.ActorBias)
			delta := v - ret
			st.Value += norm * weight * .5 * delta * delta
			cognitiveLinearBackward(g, dh, w, h, []float64{norm * weight * p.Spec.ValueWeight * delta}, p.Critic, p.CriticBias)
			for j := range dh {
				dh[j] *= 1 - h[j]*h[j]
			}
			cognitiveLinearBackward(g, make([]float64, 58), w, f.Features, dh, p.WI, p.BI)
		}
	}
	st.Loss = st.Policy + p.Spec.ValueWeight*st.Value + p.Spec.EntropyWeight*st.Entropy
	st.AdvantageMean = advSum / float64(st.Frames)
	st.AdvantageSD = math.Sqrt(math.Max(0, advSq/float64(st.Frames)-st.AdvantageMean*st.AdvantageMean))
	st.MeanReturn /= float64(st.Frames)
	for i, v := range g {
		st.GradientNorm += v * v
		if i >= p.Actor.Offset && i < p.Critic.Offset {
			st.ActorGradientNorm += v * v
		}
		if i >= p.Critic.Offset {
			st.CriticGradientNorm += v * v
		}
	}
	st.GradientNorm = math.Sqrt(st.GradientNorm)
	st.ActorGradientNorm = math.Sqrt(st.ActorGradientNorm)
	st.CriticGradientNorm = math.Sqrt(st.CriticGradientNorm)
	return st, g
}
func phase4Targets(p *Phase4StateV5, w []float64, b Phase4BatchV5) ([][]float64, [][]float64) {
	a, r := make([][]float64, len(b.Episodes)), make([][]float64, len(b.Episodes))
	for ei, ep := range b.Episodes {
		a[ei] = make([]float64, len(ep.Frames))
		r[ei] = make([]float64, len(ep.Frames))
		ret := 0.
		for t := len(ep.Frames) - 1; t >= 0; t-- {
			f := ep.Frames[t]
			ret += f.Reward
			_, _, v := phase4Forward(p, w, f.Features, f.Mask)
			r[ei][t] = ret
			a[ei][t] = ret - v
		}
	}
	return a, r
}
func (k *KnowledgeBase) LearnPhase4V5(b Phase4BatchV5, at time.Time) (Phase4LearningV5, error) {
	if e := k.ValidatePhase4BatchV5(b); e != nil {
		return Phase4LearningV5{}, e
	}
	k.Lock()
	defer k.Unlock()
	s := k.BrainState.CognitiveModel
	p := s.Phase4V5
	if p.Contract != "" {
		return Phase4LearningV5{}, errors.New("use explicit coordinated V5 learning")
	}
	w, e := k.cognitiveWeights(s)
	if e != nil {
		return Phase4LearningV5{}, e
	}
	if CognitiveDigest(w) != b.ParametersHash {
		return Phase4LearningV5{}, errors.New("parameters changed")
	}
	a, ret := phase4Targets(p, w, b)
	st, g := phase4Objective(p, w, b, a, ret)
	if !finiteNumber(st.GradientNorm) {
		return st, errors.New("nonfinite gradient")
	}
	scale := 1.
	if st.GradientNorm > 1 {
		scale = 1 / st.GradientNorm
	}
	p.Steps++
	clock := float64(p.Steps)
	for i := p.BaseParameters; i < len(w); i++ {
		d := g[i] * scale
		s.FirstMoment[i] = .9*s.FirstMoment[i] + .1*d
		s.SecondMoment[i] = .999*s.SecondMoment[i] + .001*d*d
		delta := p.Spec.LearningRate * (s.FirstMoment[i] / (1 - math.Pow(.9, clock))) / (math.Sqrt(s.SecondMoment[i]/(1-math.Pow(.999, clock))) + 1e-8)
		v := w[i] - delta
		if math.Abs(v) > 7.9 {
			st.ClippedParameters++
			v = math.Copysign(7.9, v)
		}
		edge := s.Layout.Edges[i]
		k.adjustCompositionParameter([]NodeID{edge.From}, []NodeID{edge.To}, v, 1, at, false)
	}
	p.LearnedEpisodes += uint64(len(b.Episodes))
	ev := b.Evidence
	p.LastEvidence = &ev
	st.Steps = p.Steps
	return st, nil
}
