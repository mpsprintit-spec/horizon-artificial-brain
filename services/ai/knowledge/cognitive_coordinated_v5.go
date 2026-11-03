package knowledge

// V5 coordinated contract. One final actor; world/observation GRU are stable
// during this bounded repair. MUSE learns selected observed terminal outcomes.
// BMPS integrates predicted outcomes against this same actor's continuation.
import (
	"errors"
	"math"
	"time"
)

const CoordinatedContractV5 = "v5-coordinated-payoff-continuation-v1"

type CoordinatedPredictionV5 struct {
	Phase4PredictionV5
	Competence, TerminalUtility, VOC []float64
	ContinuationUtility              float64
}
type CoordinatedEpisodeV5 struct {
	Traces [][]CognitiveInput // observed prefix for every actual decision
	Frames []Phase4FrameV5
}
type CoordinatedBatchV5 struct {
	Episodes       []CoordinatedEpisodeV5
	ParametersHash string
	Evidence       CognitiveEvidence
}
type CoordinatedLearningV5 struct {
	Policy                                       Phase4LearningV5
	AssessmentLoss, RewardLoss, MUSEGradientNorm float64
	UpdatedHead, UpdatedMUSE                     int
}

func coordinatedStateV5(s *CognitiveModelState, w, temps []float64, trace []CognitiveInput) ([]float64, []float64, []float64, error) {
	if s == nil || s.Phase4V5 == nil || len(trace) == 0 || len(trace) > s.Config.Horizon {
		return nil, nil, nil, errors.New("coordinated V5 requires canonical Phase IV and bounded observed trace")
	}
	c, n := s.Config, *s.Config.NeuralMeta
	h := make([]float64, c.Hidden)
	observed := map[int]int{}
	var x []float64
	for _, f := range trace {
		if e := f.Validate(c); e != nil {
			return nil, nil, nil, e
		}
		if f.Belief != nil || f.NeuralMeta == nil || f.NeuralMeta.Mode == "adapter-bmps" || f.NeuralMeta.Candidate != nil || f.Regulate {
			return nil, nil, nil, errors.New("coordinated inference cannot accept external belief, overrides or regulation")
		}
		if f.Reset || f.NoRecurrence {
			clear(h)
		}
		for j := n.HypothesisOffset; j < n.HypothesisOffset+n.Classes; j++ {
			if f.Features[j] != 0 {
				return nil, nil, nil, errors.New("oracle/plan hypothesis forbidden on observed trace")
			}
		}
		if f.Features[n.PlanMarker] != 0 {
			return nil, nil, nil, errors.New("plan marker forbidden on actual observations")
		}
		for a := n.Computations; a < c.Actions; a++ {
			if f.Features[n.ActionOffset+a] != 0 {
				return nil, nil, nil, errors.New("terminal-plan hint forbidden on observation")
			}
		}
		if f.NeuralMeta.Remaining == 0 {
			for op := 0; op < n.Computations; op++ {
				if f.Mask[op] {
					return nil, nil, nil, errors.New("operation beyond budget")
				}
			}
		}
		x = append([]float64(nil), f.Features...)
		if f.NeuralMeta.NoFeedback {
			clear(x[n.ResultOffset : n.ResultOffset+n.Outcomes])
			clear(x[n.OperationOffset : n.OperationOffset+n.Computations])
		} else {
			op, o := -1, -1
			for j := 0; j < n.Computations; j++ {
				if x[n.OperationOffset+j] == 1 {
					if op >= 0 {
						return nil, nil, nil, errors.New("ambiguous operation")
					}
					op = j
				}
			}
			for j := 0; j < n.Outcomes; j++ {
				if x[n.ResultOffset+j] == 1 {
					if o >= 0 {
						return nil, nil, nil, errors.New("ambiguous result")
					}
					o = j
				}
			}
			if (op >= 0) != (o >= 0) {
				return nil, nil, nil, errors.New("incomplete operation feedback")
			}
			if op >= 0 {
				if prior, ok := observed[op]; ok && prior != o {
					return nil, nil, nil, errors.New("conflicting observed history")
				}
				observed[op] = o
			}
		}
		h = cognitiveGRU(c, s.Layout, w, x, h).h
	}
	logits, e := phase2LogitsV5(s, w, h, false)
	if e != nil {
		return nil, nil, nil, e
	}
	joint := metaSoftmax(logits, temps[0])
	mass := 0.
	for state := range joint {
		for op, o := range observed {
			if metaOutcome(n, state, op) != o {
				joint[state] = 0
				break
			}
		}
		mass += joint[state]
	}
	if mass <= 0 {
		return nil, nil, nil, errors.New("zero predictive support for observations")
	}
	for i := range joint {
		joint[i] /= mass
	}
	return h, x, joint, nil
}
func coordinatedFeaturesV5(s *CognitiveModelState, w, temps, h, x, joint []float64, remaining int) ([]float64, []float64, []float64) {
	c, n := s.Config, *s.Config.NeuralMeta
	classes := metaClassBelief(n, joint)
	f := make([]float64, 58)
	copy(f, h)
	comp := make([]float64, 5)
	u := make([]float64, 5)
	for y, py := range classes {
		for a := 0; a < 4; a++ {
			sa, r := metaPlan(s, w, h, x, y, 3+a, temps)
			comp[a] += py * sa[4]
			u[a] += py * r
		}
	}
	// This is the contractual known payoff, not a confidence threshold or bonus.
	u[4] = .35
	copy(f[32:37], comp)
	copy(f[37:42], u)
	f[48] = float64(remaining) / float64(c.Horizon-1)
	return f, comp, u
}
func coordinatedTerminalValueV5(s *CognitiveModelState, w, f, u []float64, greedy bool) float64 {
	mask := []bool{false, false, false, true, true, true, true, true}
	_, p, _ := phase4Forward(s.Phase4V5, w, f, mask)
	if greedy {
		return u[maxIndex(p)-3]
	}
	v := 0.
	for a := 3; a < 8; a++ {
		v += p[a] * u[a-3]
	}
	return v
}
func coordinatedForwardV5(s *CognitiveModelState, w, temps []float64, trace []CognitiveInput, drop Phase4DropV5) (CoordinatedPredictionV5, error) {
	out := CoordinatedPredictionV5{}
	h, x, joint, e := coordinatedStateV5(s, w, temps, trace)
	if e != nil {
		return out, e
	}
	last := trace[len(trace)-1]
	n := *s.Config.NeuralMeta
	f, comp, u := coordinatedFeaturesV5(s, w, temps, h, x, joint, last.NeuralMeta.Remaining)
	copy(f[45:48], last.NeuralMeta.Costs)
	if drop.MUSE {
		clear(f[32:42])
	}
	base := coordinatedTerminalValueV5(s, w, f, u, last.NeuralMeta.Greedy)
	voc := make([]float64, 3)
	// Finite Bayesian one-step metalevel value, with a realizable neural
	// continuation instead of max utility of a different terminal selector.
	for op := 0; op < 3; op++ {
		if !last.Mask[op] || last.NeuralMeta.Remaining == 0 {
			continue
		}
		prob := make([]float64, n.Outcomes)
		for state, p := range joint {
			prob[metaOutcome(n, state, op)] += p
		}
		future := 0.
		for o, po := range prob {
			if po == 0 {
				continue
			}
			next := last
			next.Features = metaNextFeatures(s.Config, x, op, o, last.NeuralMeta.NoFeedback)
			next.Mask = append([]bool(nil), last.Mask...)
			next.Mask[op] = false
			next.Reset = false
			meta := *last.NeuralMeta
			meta.Remaining--
			next.NeuralMeta = &meta
			tt := append(append([]CognitiveInput(nil), trace...), next)
			hh, xx, jj, err := coordinatedStateV5(s, w, temps, tt)
			if err != nil {
				return out, err
			}
			ff, _, uu := coordinatedFeaturesV5(s, w, temps, hh, xx, jj, meta.Remaining)
			copy(ff[45:48], meta.Costs)
			if drop.MUSE {
				clear(ff[32:42])
			}
			future += po * coordinatedTerminalValueV5(s, w, ff, uu, last.NeuralMeta.Greedy)
		}
		voc[op] = future - base - last.NeuralMeta.Costs[op]
		f[42+op] = voc[op]
	}
	if drop.VOC {
		clear(f[42:45])
	} // old actor/critic slots49:58 intentionally unused
	_, pi, value := phase4Forward(s.Phase4V5, w, f, last.Mask)
	action := samplePolicy(pi, last.Draw)
	if last.NeuralMeta.Greedy {
		action = maxIndex(pi)
	}
	out.Phase4PredictionV5 = Phase4PredictionV5{Features: f, Policy: pi, Value: value, Action: action, ParametersHash: CognitiveDigest(w)}
	out.Competence = comp
	out.TerminalUtility = u
	out.VOC = voc
	out.ContinuationUtility = base
	return out, nil
}
func (k *KnowledgeBase) PreviewCoordinatedV5(trace []CognitiveInput, drop Phase4DropV5) (CoordinatedPredictionV5, error) {
	k.RLock()
	defer k.RUnlock()
	s := k.BrainState.CognitiveModel
	if s == nil || s.Phase4V5 == nil || s.Config.Hidden != 32 || s.Config.Actions != 8 || s.Config.Horizon != 4 {
		return CoordinatedPredictionV5{}, errors.New("explicit initialized V5 circuit required")
	}
	temps := k.metaTemperatures(s)
	w, e := k.cognitiveWeights(s)
	if e != nil {
		return CoordinatedPredictionV5{}, e
	}
	return coordinatedForwardV5(s, w, temps, trace, drop)
}
func coordinatedMUSEParameterV5(s *CognitiveModelState, i int) bool {
	in := func(t CognitiveTensor) bool { return i >= t.Offset && i < t.Offset+t.Rows*t.Cols }
	l := s.Layout
	if in(l.Assessment) {
		return (i-l.Assessment.Offset)/l.Assessment.Cols == 4
	}
	if in(l.AssessmentBias) {
		return i-l.AssessmentBias.Offset == 4
	}
	return in(s.NeuralMeta.Reward) || in(s.NeuralMeta.RewardBias) || in(l.WI) && terminalReadoutParameterV3(s, i)
}
func coordinatedMUSEObjectiveV5(s *CognitiveModelState, w, temps []float64, b CoordinatedBatchV5) (float64, float64, []float64, error) {
	g := make([]float64, len(w))
	assessment, reward := 0., 0.
	norm := 1 / float64(len(b.Episodes))
	l, c := s.Layout, s.Config
	for _, ep := range b.Episodes {
		t := len(ep.Frames) - 1
		f := ep.Frames[t]
		if f.Action == 7 {
			continue
		}
		h, x, joint, e := coordinatedStateV5(s, w, temps, ep.Traces[t])
		if e != nil {
			return 0, 0, nil, e
		}
		classes := metaClassBelief(*c.NeuralMeta, joint)
		plans := make([]gruFrame, 4)
		sa, rs := make([]float64, 4), make([]float64, 4)
		pc, pr := 0., 0.
		for y, py := range classes {
			plans[y] = cognitiveGRU(c, l, w, metaPlanFeatures(c, x, y, f.Action), h)
			z := mat(w, l.Assessment, plans[y].h)
			addBias(z, w, l.AssessmentBias)
			sa[y] = sigmoid(z[4] / temps[5])
			rr := mat(w, s.NeuralMeta.Reward, plans[y].h)
			addBias(rr, w, s.NeuralMeta.RewardBias)
			rs[y] = math.Tanh(rr[0])
			pc += py * sa[y]
			pr += py * rs[y]
		}
		pc = math.Max(1e-12, math.Min(1-1e-12, pc))
		label := f.Reward
		assessment += norm * (-label*math.Log(pc) - (1-label)*math.Log(1-pc))
		reward += norm * .5 * (pr - label) * (pr - label)
		for y, py := range classes {
			dp := make([]float64, c.Hidden)
			ds := make([]float64, 5)
			ds[4] = norm * c.AssessmentWeight * (pc - label) / (pc * (1 - pc)) * py * sa[y] * (1 - sa[y]) / temps[5]
			cognitiveLinearBackward(g, dp, w, plans[y].h, ds, l.Assessment, l.AssessmentBias)
			dr := norm * c.NeuralMeta.RewardWeight * (pr - label) * py * (1 - rs[y]*rs[y])
			cognitiveLinearBackward(g, dp, w, plans[y].h, []float64{dr}, s.NeuralMeta.Reward, s.NeuralMeta.RewardBias)
			cognitiveGRUBackward(c, l, w, g, plans[y], dp)
		}
	}
	for i := range g {
		if !coordinatedMUSEParameterV5(s, i) {
			g[i] = 0
		}
	}
	return assessment, reward, g, nil
}
func validateCoordinatedBatchV5(s *CognitiveModelState, w, temps []float64, b CoordinatedBatchV5) error {
	if s == nil || s.Phase4V5 == nil || len(b.Episodes) == 0 || len(b.Episodes) > 128 || CognitiveDigest(w) != b.ParametersHash {
		return errors.New("invalid/stale coordinated batch")
	}
	if e := b.Evidence.Validate(); e != nil {
		return e
	}
	if b.Evidence.Digest != CognitiveDigest(b.Episodes) {
		return errors.New("coordinated evidence mismatch")
	}
	for _, ep := range b.Episodes {
		if len(ep.Frames) == 0 || len(ep.Frames) > 4 || len(ep.Traces) != len(ep.Frames) {
			return errors.New("coordinated trajectory shape")
		}
		for t, f := range ep.Frames {
			tr := ep.Traces[t]
			if len(tr) == 0 {
				return errors.New("missing observed prefix")
			}
			last := tr[len(tr)-1]
			if e := last.Validate(s.Config); e != nil {
				return e
			}
			if last.NeuralMeta.Greedy {
				return errors.New("greedy traces cannot train on-policy")
			}
			if t > 0 {
				prior := ep.Traces[t-1]
				if len(tr) != len(prior)+1 || CognitiveDigest(tr[:len(prior)]) != CognitiveDigest(prior) {
					return errors.New("trace history changed")
				}
				prev := ep.Frames[t-1]
				if prev.Action >= 3 {
					return errors.New("continued terminal episode")
				}
				n := s.Config.NeuralMeta
				op, o := prev.Action, -1
				for j := 0; j < n.Outcomes; j++ {
					if last.Features[n.ResultOffset+j] == 1 {
						o = j
					}
				}
				if o < 0 || last.Features[n.OperationOffset+op] != 1 || last.Mask[op] || last.NeuralMeta.Remaining != prior[len(prior)-1].NeuralMeta.Remaining-1 {
					return errors.New("missing actual operation feedback")
				}
				if prev.Reward != -prior[len(prior)-1].NeuralMeta.Costs[op] {
					return errors.New("operation payoff differs from cost")
				}
			}
			if len(f.Mask) != 8 || CognitiveDigest(f.Mask) != CognitiveDigest(last.Mask) || f.Action < 0 || f.Action >= 8 || !f.Mask[f.Action] {
				return errors.New("illegal action")
			}
			for a := 3; a < 8; a++ {
				if !f.Mask[a] {
					return errors.New("terminal mask cannot force answers")
				}
			}
			p, e := coordinatedForwardV5(s, w, temps, tr, Phase4DropV5{})
			if e != nil {
				return e
			}
			if CognitiveDigest(p.Features) != CognitiveDigest(f.Features) || p.Action != f.Action || math.Abs(p.Policy[f.Action]-f.BehaviorProbability) > 1e-12 || math.Abs(p.Value-f.BehaviorValue) > 1e-12 || f.Draw != last.Draw {
				return errors.New("stale behavior/features")
			}
			if t == len(ep.Frames)-1 {
				if f.Action < 3 || f.Action == 7 && f.Reward != .35 || f.Action != 7 && f.Reward != 0 && f.Reward != 1 {
					return errors.New("terminal payoff invalid")
				}
			} else if f.Action >= 3 {
				return errors.New("invalid nonterminal action")
			}
		}
	}
	return nil
}
func (k *KnowledgeBase) ValidateCoordinatedBatchV5(b CoordinatedBatchV5) error {
	k.RLock()
	defer k.RUnlock()
	s := k.BrainState.CognitiveModel
	if s == nil || s.Phase4V5 == nil || s.NeuralMeta == nil {
		return errors.New("no cognitive circuit")
	}
	temps := k.metaTemperatures(s)
	w, e := k.cognitiveWeights(s)
	if e != nil {
		return e
	}
	return validateCoordinatedBatchV5(s, w, temps, b)
}
func (k *KnowledgeBase) LearnCoordinatedV5(b CoordinatedBatchV5, at time.Time) (CoordinatedLearningV5, error) {
	k.Lock()
	defer k.Unlock()
	var out CoordinatedLearningV5
	s := k.BrainState.CognitiveModel
	if s == nil || s.Phase4V5 == nil || s.NeuralMeta == nil {
		return out, errors.New("no circuit")
	}
	temps := k.metaTemperatures(s)
	w, e := k.cognitiveWeights(s)
	if e != nil {
		return out, e
	}
	if e = validateCoordinatedBatchV5(s, w, temps, b); e != nil {
		return out, e
	}
	p := s.Phase4V5
	policy := Phase4BatchV5{}
	for _, ep := range b.Episodes {
		policy.Episodes = append(policy.Episodes, Phase4EpisodeV5{Frames: ep.Frames})
	}
	a, r := phase4Targets(p, w, policy)
	var g []float64
	out.Policy, g = phase4Objective(p, w, policy, a, r)
	assessment, reward, mg, e := coordinatedMUSEObjectiveV5(s, w, temps, b)
	if e != nil {
		return out, e
	}
	out.AssessmentLoss = assessment
	out.RewardLoss = reward
	for _, x := range mg {
		out.MUSEGradientNorm += x * x
	}
	out.MUSEGradientNorm = math.Sqrt(out.MUSEGradientNorm)
	for _, x := range append(append([]float64(nil), g...), mg...) {
		if !finiteNumber(x) {
			return out, errors.New("nonfinite coordinated gradient")
		}
	}
	p.Steps++
	s.OptimizerStep++
	headScale, museScale := 1., 1.
	if out.Policy.GradientNorm > 1 {
		headScale = 1 / out.Policy.GradientNorm
	}
	if out.MUSEGradientNorm > 1 {
		museScale = 1 / out.MUSEGradientNorm
	}
	for i := range w {
		isHead := i >= p.BaseParameters
		isMuse := coordinatedMUSEParameterV5(s, i)
		if !isHead && !isMuse {
			continue
		}
		d, clock, lr := g[i]*headScale, float64(p.Steps), p.Spec.LearningRate
		if isMuse {
			d = mg[i] * museScale
			clock = float64(s.OptimizerStep)
			lr = s.Config.LearningRate
			out.UpdatedMUSE++
		} else {
			out.UpdatedHead++
		}
		s.FirstMoment[i] = .9*s.FirstMoment[i] + .1*d
		s.SecondMoment[i] = .999*s.SecondMoment[i] + .001*d*d
		delta := lr * (s.FirstMoment[i] / (1 - math.Pow(.9, clock))) / (math.Sqrt(s.SecondMoment[i]/(1-math.Pow(.999, clock))) + 1e-8)
		v := math.Max(-7.9, math.Min(7.9, w[i]-delta))
		edge := s.Layout.Edges[i]
		k.adjustCompositionParameter([]NodeID{edge.From}, []NodeID{edge.To}, v, 1, at, false)
	}
	p.Contract = CoordinatedContractV5
	p.LearnedEpisodes += uint64(len(b.Episodes))
	ev := b.Evidence
	p.LastEvidence = &ev
	out.Policy.Steps = p.Steps
	return out, nil
}

func (k *KnowledgeBase) HasCoordinatedV5() bool {
	k.RLock()
	defer k.RUnlock()
	s := k.BrainState.CognitiveModel
	return s != nil && s.Phase4V5 != nil && s.Phase4V5.Contract == CoordinatedContractV5
}
