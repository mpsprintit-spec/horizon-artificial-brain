package knowledge

// Explicit experimental binary-payoff contract. No old selector or replay path
// changes. Reuses MUSE assessment, finite BMPS and the existing GRU backward.
import (
	"errors"
	"math"
	"slices"
	"time"
)

const CoordinatedFidelityV5 = "scientific-fidelity-v5"

type FidelityLearningV5 struct {
	AssessmentOnly   bool
	FreezeRecurrence bool
}

// Research acquisition only. No actor score is fitted on these mixed-policy
// records. Conditional competence learns the actually selected outcome.
func FidelityAssessmentPolicyV5(pi []float64, mask []bool) []float64 {
	return phase4BehaviorPolicy(pi, mask, .5)
}

func CoordinatedEvidenceDigestV5(b CoordinatedBatchV5) string {
	if b.Fidelity == nil {
		return CognitiveDigest(b.Episodes)
	}
	return CognitiveDigest(struct {
		Episodes []CoordinatedEpisodeV5
		Options  FidelityLearningV5
	}{b.Episodes, *b.Fidelity})
}

func validateFidelityContractV5(s *CognitiveModelState, b CoordinatedBatchV5) error {
	if b.Fidelity != nil && (s == nil || s.Config.NeuralMeta == nil || s.Config.Gamma != 1) {
		return errors.New("fidelity requires the native undiscounted metalevel model")
	}
	if b.Fidelity != nil && b.TerminalUtility != nil {
		return errors.New("fidelity and historical terminal objectives cannot mix")
	}
	for _, ep := range b.Episodes {
		if b.Fidelity != nil && b.Fidelity.AssessmentOnly && len(ep.Frames) != 1 {
			return errors.New("assessment acquisition records one terminal outcome")
		}
		for t, trace := range ep.Traces {
			if b.Fidelity != nil && ((!b.Fidelity.AssessmentOnly && len(trace) != t+1) || len(trace) == 0 || !trace[0].Reset) {
				return errors.New("fidelity requires complete reset-to-terminal observed episodes")
			}
			if b.Fidelity != nil && b.Fidelity.AssessmentOnly {
				if len(trace[len(trace)-1].Mask) != s.Config.Actions {
					return errors.New("fidelity terminal mask shape mismatch")
				}
				for _, legal := range trace[len(trace)-1].Mask[:3] {
					if legal {
						return errors.New("assessment acquisition must be terminal-only")
					}
				}
			}
			for j, x := range trace {
				if b.Fidelity != nil && len(x.Features) != s.Config.Inputs {
					return errors.New("fidelity observation shape mismatch")
				}
				if b.Fidelity != nil && x.Reset != (j == 0) {
					return errors.New("fidelity reset boundary disagrees with observed episode")
				}
				if b.Fidelity != nil && j == 0 {
					n := s.Config.NeuralMeta
					for _, span := range [][2]int{{n.ResultOffset, n.Outcomes}, {n.OperationOffset, n.Computations}, {n.ActionOffset, n.Computations}, {n.StepIndex, 1}} {
						for _, value := range x.Features[span[0] : span[0]+span[1]] {
							if value != 0 {
								return errors.New("fidelity root contains unrecorded mental history")
							}
						}
					}
				}
				fidelity := x.NeuralMeta != nil && x.NeuralMeta.Mode == CoordinatedFidelityV5
				if fidelity != (b.Fidelity != nil) {
					return errors.New("fidelity learning options and observed mode disagree")
				}
			}
		}
	}
	return nil
}

func fidelityBMPSWeightsV5(s *CognitiveModelState) [4]float64 {
	// These are the initialization and the exact search-history reconstruction
	// used by InitializeCognitiveModel / FitCognitiveBMPS, respectively.
	weights := [4]float64{1.0 / 3, 1.0 / 3, 1.0 / 3, 1}
	if len(s.Search) > 0 {
		best := 0
		for i := range s.Search {
			if s.Search[i].Return > s.Search[best].Return {
				best = i
			}
		}
		weights = BMPSWeights(s.Search[best].Point, s.Config.Horizon)
	}
	return weights
}

func fidelityForwardV5(s *CognitiveModelState, w, temps []float64, trace []CognitiveInput, drop Phase4DropV5) (CoordinatedPredictionV5, error) {
	out := CoordinatedPredictionV5{}
	if drop.Actor {
		return out, errors.New("legacy actor ablation is not a fidelity-mode mechanism")
	}
	for _, x := range trace {
		if x.NeuralMeta == nil || x.NeuralMeta.Mode != CoordinatedFidelityV5 {
			return out, errors.New("mixed selector contracts on fidelity history")
		}
	}
	h, x, joint, err := coordinatedStateV5(s, w, temps, trace)
	if err != nil {
		return out, err
	}
	last := trace[len(trace)-1]
	if !slices.Contains(last.Mask[3:], true) {
		return out, errors.New("fidelity requires a legal terminal continuation")
	}
	classes := metaClassBelief(*s.Config.NeuralMeta, joint)
	utilities, competence := metaUtilities(s, w, h, x, classes, temps, true)
	f, comp, u := make([]float64, 58), make([]float64, 5), make([]float64, 5)
	copy(f, h)
	for y, py := range classes {
		utilities[y][4] = .35
		for a := 0; a < 4; a++ {
			comp[a] += py * competence[y][a]
			u[a] += py * utilities[y][a]
		}
	}
	u[4] = .35
	copy(f[32:37], comp)
	copy(f[37:42], u)
	copy(f[45:48], last.NeuralMeta.Costs)
	f[48] = float64(last.NeuralMeta.Remaining) / float64(s.Config.Horizon-1)
	belief, mapping, err := metaBeliefSupport(s, joint, classes, utilities, last, len(trace) > 1 && !last.NeuralMeta.NoFeedback)
	if err != nil {
		return out, err
	}
	// BMPS's bound is over legal terminal utilities, not an override of the
	// executing actor. Its ideal-stop assumption is explicit in the output.
	for i, values := range belief.Utilities {
		legal := []float64{}
		for a, value := range values {
			if last.Mask[a+3] {
				legal = append(legal, value)
			}
		}
		belief.Utilities[i] = legal
	}
	bmps, err := SelectBMPS(belief, fidelityBMPSWeightsV5(s))
	if err != nil {
		return out, err
	}
	voc := make([]float64, 3)
	if last.NeuralMeta.Remaining > 0 {
		for i, op := range mapping {
			voc[op] = bmps.VOC[i]
			f[42+op] = voc[op]
			f[49+op] = bmps.Features[i].Myopic
			f[52+op] = bmps.Features[i].Perfect
			f[55+op] = bmps.Features[i].Subset
		}
	}
	if drop.MUSE {
		clear(f[32:42])
	}
	if drop.VOC {
		clear(f[42:45])
		clear(f[49:58])
	}
	_, pi, value := phase4Forward(s.Phase4V5, w, f, last.Mask)
	action := samplePolicy(pi, last.Draw)
	if last.NeuralMeta.Greedy {
		action = maxIndex(pi)
	}
	terminalMask := append([]bool(nil), last.Mask...)
	clear(terminalMask[:3])
	stopFeatures := append([]float64(nil), f...)
	clear(stopFeatures[42:45])
	clear(stopFeatures[49:58])
	out.Phase4PredictionV5 = Phase4PredictionV5{Features: f, Policy: pi, Value: value, Action: action, ParametersHash: CognitiveDigest(w)}
	out.Competence, out.TerminalUtility, out.VOC, out.BMPS = comp, u, voc, &bmps
	out.PredictedJoint = joint
	out.ContinuationUtility = coordinatedTerminalValueV5(s, w, stopFeatures, u, terminalMask, last.NeuralMeta.Greedy, 0)
	return out, nil
}

func fidelityRecurrentParameterV5(s *CognitiveModelState, i int) bool {
	if coordinatedMUSEParameterV5(s, i) {
		return false
	}
	for _, tensor := range []CognitiveTensor{s.Layout.WI, s.Layout.WH, s.Layout.BI, s.Layout.BH} {
		if i >= tensor.Offset && i < tensor.Offset+tensor.Rows*tensor.Cols {
			return true
		}
	}
	return false
}

func fidelityMUSEParameterV5(s *CognitiveModelState, i int) bool {
	for _, tensor := range []CognitiveTensor{s.NeuralMeta.Reward, s.NeuralMeta.RewardBias} {
		if i >= tensor.Offset && i < tensor.Offset+tensor.Rows*tensor.Cols {
			return false
		}
	}
	return coordinatedMUSEParameterV5(s, i)
}

// Proper Bernoulli mixture likelihood for only the selected observed answer.
// The signed general-reward head is preserved for legacy/nonbinary tasks.
func fidelityMUSEObjectiveV5(s *CognitiveModelState, w, temps []float64, b CoordinatedBatchV5) (float64, []float64, error) {
	g, loss := make([]float64, len(w)), 0.
	norm := 1 / float64(len(b.Episodes))
	for _, ep := range b.Episodes {
		t := len(ep.Frames) - 1
		frame := ep.Frames[t]
		if frame.Action == 7 {
			continue
		}
		h, x, joint, err := coordinatedStateV5(s, w, temps, ep.Traces[t])
		if err != nil {
			return 0, nil, err
		}
		classes := metaClassBelief(*s.Config.NeuralMeta, joint)
		plans, z := make([]gruFrame, 4), make([]float64, 4)
		lp, ln := make([]float64, 4), make([]float64, 4)
		for y, py := range classes {
			plans[y] = cognitiveGRU(s.Config, s.Layout, w, metaPlanFeatures(s.Config, x, y, frame.Action), h)
			logits := mat(w, s.Layout.Assessment, plans[y].h)
			addBias(logits, w, s.Layout.AssessmentBias)
			z[y] = logits[4] / temps[5]
			lp[y], ln[y] = math.Log(py)-metaBCE(z[y], 1), math.Log(py)-metaBCE(z[y], 0)
		}
		logP, logN := fidelityLogSumV5(lp), fidelityLogSumV5(ln)
		for y := range classes {
			den, sign := logN, 1.
			if frame.Reward == 1 {
				den, sign = logP, -1
			}
			d := sign * norm * math.Exp(math.Log(classes[y])-metaBCE(z[y], 1)-metaBCE(z[y], 0)-den) / temps[5]
			dp, ds := make([]float64, s.Config.Hidden), make([]float64, 5)
			ds[4] = s.Config.AssessmentWeight * d
			cognitiveLinearBackward(g, dp, w, plans[y].h, ds, s.Layout.Assessment, s.Layout.AssessmentBias)
			cognitiveGRUBackward(s.Config, s.Layout, w, g, plans[y], dp)
		}
		if frame.Reward == 1 {
			loss -= norm * logP
		} else {
			loss -= norm * logN
		}
	}
	for i := range g {
		if !fidelityMUSEParameterV5(s, i) {
			g[i] = 0
		}
	}
	return loss, g, nil
}

func fidelityLogSumV5(x []float64) float64 {
	m := slices.Max(x)
	z := 0.
	for _, v := range x {
		z += math.Exp(v - m)
	}
	return m + math.Log(z)
}

// Exact BPTT of the actor/critic surrogate with detached auxiliary features,
// returns and advantages. Recompute only the GRU block, preserve actual input
// histories. No world labels or gradients through imagined outcomes are used.
func fidelityPolicyObjectiveV5(s *CognitiveModelState, w []float64, b CoordinatedBatchV5, adv, returns [][]float64) (Phase4LearningV5, []float64) {
	p := s.Phase4V5
	g := make([]float64, len(w))
	st := Phase4LearningV5{Episodes: len(b.Episodes), AdvantageMin: math.Inf(1), AdvantageMax: math.Inf(-1)}
	scale := 1 / float64(len(b.Episodes))
	advSq := 0.
	for ei, ep := range b.Episodes {
		for t, frame := range ep.Frames {
			h := make([]float64, s.Config.Hidden)
			trace := ep.Traces[t]
			cache := make([]gruFrame, len(trace))
			for j, x := range trace {
				if x.Reset || x.NoRecurrence {
					clear(h)
				}
				features := append([]float64(nil), x.Features...)
				if x.NeuralMeta.NoFeedback {
					n := s.Config.NeuralMeta
					clear(features[n.ResultOffset : n.ResultOffset+n.Outcomes])
					clear(features[n.OperationOffset : n.OperationOffset+n.Computations])
				}
				cache[j] = cognitiveGRU(s.Config, s.Layout, w, features, h)
				h = cache[j].h
			}
			x := append([]float64(nil), frame.Features...)
			copy(x, h)
			hh, pi, value := phase4Forward(p, w, x, frame.Mask)
			logits := mat(w, p.Actor, hh)
			addBias(logits, w, p.ActorBias)
			legal := []float64{}
			for a, yes := range frame.Mask {
				if yes {
					legal = append(legal, logits[a])
				}
			}
			logZ := fidelityLogSumV5(legal)
			ent := 0.
			for a, q := range pi {
				if frame.Mask[a] {
					ent += q * (logits[a] - logZ)
				}
			}
			advantage, delta := adv[ei][t], value-returns[ei][t]
			st.Policy -= scale * advantage * (logits[frame.Action] - logZ)
			st.Entropy += scale * ent
			st.Value += scale * .5 * delta * delta
			dy := make([]float64, 8)
			for a, q := range pi {
				if frame.Mask[a] {
					dy[a] = scale * (advantage*q + p.Spec.EntropyWeight*q*(logits[a]-logZ-ent))
					if a == frame.Action {
						dy[a] -= scale * advantage
					}
				}
			}
			dh := make([]float64, 16)
			cognitiveLinearBackward(g, dh, w, hh, dy, p.Actor, p.ActorBias)
			cognitiveLinearBackward(g, dh, w, hh, []float64{scale * p.Spec.ValueWeight * delta}, p.Critic, p.CriticBias)
			for j := range dh {
				dh[j] *= 1 - hh[j]*hh[j]
			}
			dx := make([]float64, 58)
			cognitiveLinearBackward(g, dx, w, x, dh, p.WI, p.BI)
			back := append([]float64(nil), dx[:32]...)
			for j := len(cache) - 1; j >= 0; j-- {
				back = cognitiveGRUBackward(s.Config, s.Layout, w, g, cache[j], back)
				if trace[j].Reset || trace[j].NoRecurrence {
					clear(back)
				}
			}
			st.Frames++
			st.AdvantageMean += advantage
			advSq += advantage * advantage
			st.AdvantageMin = math.Min(st.AdvantageMin, advantage)
			st.AdvantageMax = math.Max(st.AdvantageMax, advantage)
			st.MeanReturn += returns[ei][t]
		}
	}
	st.Loss = st.Policy + p.Spec.ValueWeight*st.Value + p.Spec.EntropyWeight*st.Entropy
	st.AdvantageMean /= float64(st.Frames)
	st.MeanReturn /= float64(st.Frames)
	st.AdvantageSD = math.Sqrt(math.Max(0, advSq/float64(st.Frames)-st.AdvantageMean*st.AdvantageMean))
	for i := range g {
		allowed := i >= p.BaseParameters || !b.Fidelity.FreezeRecurrence && fidelityRecurrentParameterV5(s, i)
		if !allowed {
			g[i] = 0
		}
		st.GradientNorm += g[i] * g[i]
		if i >= p.Actor.Offset && i < p.Critic.Offset {
			st.ActorGradientNorm += g[i] * g[i]
		}
		if i >= p.Critic.Offset {
			st.CriticGradientNorm += g[i] * g[i]
		}
	}
	st.GradientNorm = math.Sqrt(st.GradientNorm)
	st.ActorGradientNorm = math.Sqrt(st.ActorGradientNorm)
	st.CriticGradientNorm = math.Sqrt(st.CriticGradientNorm)
	return st, g
}

func (k *KnowledgeBase) learnFidelityV5(s *CognitiveModelState, w, temps []float64, b CoordinatedBatchV5, at time.Time) (CoordinatedLearningV5, error) {
	out := CoordinatedLearningV5{}
	assessment, mg, err := fidelityMUSEObjectiveV5(s, w, temps, b)
	if err != nil {
		return out, err
	}
	out.AssessmentLoss = assessment
	g := make([]float64, len(w))
	if !b.Fidelity.AssessmentOnly {
		policy := Phase4BatchV5{}
		for _, ep := range b.Episodes {
			policy.Episodes = append(policy.Episodes, Phase4EpisodeV5{Frames: ep.Frames})
		}
		a, r := phase4Targets(s.Phase4V5, w, policy)
		out.Policy, g = fidelityPolicyObjectiveV5(s, w, b, a, r)
	}
	for i := range g {
		if !finiteNumber(g[i]) || !finiteNumber(mg[i]) {
			return out, errors.New("nonfinite fidelity gradient")
		}
		out.MUSEGradientNorm += mg[i] * mg[i]
		if fidelityRecurrentParameterV5(s, i) {
			out.RecurrentGradientNorm += g[i] * g[i]
		}
	}
	out.MUSEGradientNorm = math.Sqrt(out.MUSEGradientNorm)
	out.RecurrentGradientNorm = math.Sqrt(out.RecurrentGradientNorm)
	if !b.Fidelity.AssessmentOnly {
		s.Phase4V5.Steps++
	}
	s.OptimizerStep++
	pgScale, mgScale := 1/math.Max(1, out.Policy.GradientNorm), 1/math.Max(1, out.MUSEGradientNorm)
	for i := range w {
		muse := fidelityMUSEParameterV5(s, i)
		recurrent := !b.Fidelity.AssessmentOnly && !b.Fidelity.FreezeRecurrence && fidelityRecurrentParameterV5(s, i)
		head := !b.Fidelity.AssessmentOnly && i >= s.Phase4V5.BaseParameters
		if !muse && !recurrent && !head {
			continue
		}
		d, clock, lr := g[i]*pgScale, float64(s.OptimizerStep), s.Config.LearningRate
		if muse {
			d = mg[i] * mgScale
			out.UpdatedMUSE++
		} else if recurrent {
			out.UpdatedRecurrent++
		} else {
			clock = float64(s.Phase4V5.Steps)
			lr = s.Phase4V5.Spec.LearningRate
			out.UpdatedHead++
		}
		s.FirstMoment[i] = .9*s.FirstMoment[i] + .1*d
		s.SecondMoment[i] = .999*s.SecondMoment[i] + .001*d*d
		delta := lr * (s.FirstMoment[i] / (1 - math.Pow(.9, clock))) / (math.Sqrt(s.SecondMoment[i]/(1-math.Pow(.999, clock))) + 1e-8)
		v := math.Max(-7.9, math.Min(7.9, w[i]-delta))
		edge := s.Layout.Edges[i]
		k.adjustCompositionParameter([]NodeID{edge.From}, []NodeID{edge.To}, v, 1, at, false)
	}
	s.Phase4V5.Contract = CoordinatedContractV5
	s.Phase4V5.LearnedEpisodes += uint64(len(b.Episodes))
	ev := b.Evidence
	s.Phase4V5.LastEvidence = &ev
	out.Policy.Steps = s.Phase4V5.Steps
	return out, nil
}
