package knowledge

import (
	"errors"
	"math"
	"time"
)

type VerifiedMentalResult struct {
	Operation int `json:"operation"`
	Outcome   int `json:"outcome"`
}

func validateVerifiedMentalEpisode(c CognitiveModelConfig, e CognitiveTrainingEpisode, policy bool) error {
	n := c.NeuralMeta
	if e.VerifiedTarget == nil || *e.VerifiedTarget < 0 || *e.VerifiedTarget >= n.Classes {
		return errors.New("V2 learning requires a post-outcome verified class")
	}
	seen := map[int]bool{}
	obtained := map[int]int{}
	for _, v := range e.VerifiedOperations {
		if v.Operation < 0 || v.Operation >= n.Computations || v.Outcome < 0 || v.Outcome >= n.Outcomes || seen[v.Operation] {
			return errors.New("invalid/repeated verified mental result")
		}
		seen[v.Operation] = true
		obtained[v.Operation] = v.Outcome
	}
	queried := map[int]bool{}
	for t, f := range e.Frames {
		if policy && f.NeuralMeta.Greedy {
			return errors.New("greedy evaluation is not an on-policy REINFORCE trajectory")
		}
		if policy && f.NeuralMeta.Mode != "recurrent" {
			return errors.New("REINFORCE requires on-policy actor trajectories; model/self learning may use other verified behaviour")
		}
		if f.Action < n.Computations {
			o, ok := obtained[f.Action]
			if !ok || queried[f.Action] || t+1 >= len(e.Frames) {
				return errors.New("verified operation lacks its actually executed result transition")
			}
			queried[f.Action] = true
			next := e.Frames[t+1].Features
			for j := 0; j < n.Computations; j++ {
				want := 0.0
				if j == f.Action {
					want = 1
				}
				if next[n.OperationOffset+j] != want {
					return errors.New("verified operation disagrees with returned observation")
				}
			}
			for j := 0; j < n.Outcomes; j++ {
				want := 0.0
				if j == o {
					want = 1
				}
				if next[n.ResultOffset+j] != want {
					return errors.New("verified result disagrees with returned observation")
				}
			}
		} else if t != len(e.Frames)-1 {
			return errors.New("mental trajectory continued after a terminal action")
		}
	}
	if len(queried) != len(obtained) {
		return errors.New("unqueried potential results are not supervised labels")
	}
	last := e.Frames[len(e.Frames)-1].Action
	if last < n.Computations {
		return errors.New("verified trajectory has no terminal outcome")
	}
	wantSuccess := -1
	if last-n.Computations == *e.VerifiedTarget {
		wantSuccess = len(e.Frames) - 1
	}
	if e.SuccessAt != wantSuccess {
		return errors.New("Bernoulli success label disagrees with verified terminal class")
	}
	return nil
}
func metaAllowed(n NeuralMetaConfig, e CognitiveTrainingEpisode) []bool {
	out := make([]bool, n.States())
	for state := range out {
		if metaTarget(n, state) != *e.VerifiedTarget {
			continue
		}
		out[state] = true
		for _, obs := range e.VerifiedOperations {
			if metaOutcome(n, state, obs.Operation) != obs.Outcome {
				out[state] = false
				break
			}
		}
	}
	return out
}
func metaBCE(logit, y float64) float64 {
	return math.Max(logit, 0) - y*logit + math.Log1p(math.Exp(-math.Abs(logit)))
}
func cognitiveBatchObjectiveV2(s *CognitiveModelState, w []float64, b CognitiveTrainingBatch, targets cognitiveTargets) (CognitiveLearningResult, []float64) {
	c, l := s.Config, s.Layout
	n := *c.NeuralMeta
	stats := CognitiveLearningResult{Episodes: len(b.Episodes)}
	g := make([]float64, len(w))
	scale := 1 / float64(len(b.Episodes))
	for ei, e := range b.Episodes {
		frames := make([]gruFrame, len(e.Frames))
		ps := make([][]float64, len(frames))
		values := make([]float64, len(frames))
		hp := make([]float64, c.Hidden)
		allowed := metaAllowed(n, e)
		for t, f := range e.Frames {
			if b.NoRecurrence {
				hp = make([]float64, c.Hidden)
			}
			x := append([]float64{}, f.Features...)
			if f.NeuralMeta.NoFeedback {
				clear(x[n.ResultOffset : n.ResultOffset+n.Outcomes])
				clear(x[n.OperationOffset : n.OperationOffset+n.Computations])
			}
			frames[t] = cognitiveGRU(c, l, w, x, hp)
			hp = frames[t].h
			ps[t], _, values[t] = cognitiveHeads(l, w, hp, f.Mask)
		}
		next := make([]float64, c.Hidden)
		for t := len(frames) - 1; t >= 0; t-- {
			f := e.Frames[t]
			h := frames[t].h
			dh := append([]float64{}, next...)
			p := ps[t]
			if b.LearnPolicy {
				adv := targets.Advantages[ei][t]
				stats.Policy -= scale * adv * math.Log(math.Max(1e-15, p[f.Action]))
				dy := make([]float64, c.Actions)
				ent := 0.0
				for _, q := range p {
					if q > 0 {
						ent += q * math.Log(q)
					}
				}
				stats.Entropy += scale * ent
				for a, q := range p {
					if q == 0 {
						continue
					}
					d := adv * q
					if a == f.Action {
						d -= adv
					}
					dy[a] = scale * (d + c.EntropyWeight*q*(math.Log(q)-ent))
				}
				cognitiveLinearBackward(g, dh, w, h, dy, l.Actor, l.ActorBias)
				delta := values[t] - targets.Returns[ei][t]
				stats.Value += scale * .5 * delta * delta
				cognitiveLinearBackward(g, dh, w, h, []float64{scale * c.ValueWeight * delta}, l.Critic, l.CriticBias)
			}
			joint := metaDistribution(s, w, h, 1, false)
			mass := 0.0
			for state, q := range joint {
				if allowed[state] {
					mass += q
				}
			}
			stats.Joint -= scale * math.Log(mass)
			dj := make([]float64, len(joint))
			for state, q := range joint {
				v := q
				if allowed[state] {
					v -= q / mass
				}
				dj[state] = scale * n.JointWeight * v
			}
			cognitiveLinearBackward(g, dh, w, h, dj, s.NeuralMeta.Joint, s.NeuralMeta.JointBias)
			plan := cognitiveGRU(c, l, w, metaPlanFeatures(c, f.Features, *e.VerifiedTarget, f.Action), h)
			dp := make([]float64, c.Hidden)
			logits := mat(w, l.Assessment, plan.h)
			addBias(logits, w, l.AssessmentBias)
			ds := make([]float64, 5)
			for q, z := range logits {
				limit := math.Ceil(float64(c.Horizon) * float64(q+1) / 5)
				y := 0.0
				if e.SuccessAt >= t && float64(e.SuccessAt-t+1) <= limit {
					y = 1
				}
				stats.Assessment += scale * metaBCE(z, y)
				ds[q] = scale * c.AssessmentWeight * (sigmoid(z) - y)
			}
			cognitiveLinearBackward(g, dp, w, plan.h, ds, l.Assessment, l.AssessmentBias)
			reward := mat(w, s.NeuralMeta.Reward, plan.h)
			addBias(reward, w, s.NeuralMeta.RewardBias)
			r := math.Tanh(reward[0])
			delta := r - f.Reward
			stats.Reward += scale * .5 * delta * delta
			cognitiveLinearBackward(g, dp, w, plan.h, []float64{scale * n.RewardWeight * delta * (1 - r*r)}, s.NeuralMeta.Reward, s.NeuralMeta.RewardBias)
			back := cognitiveGRUBackward(c, l, w, g, plan, dp)
			for j := range dh {
				dh[j] += back[j]
			}
			next = cognitiveGRUBackward(c, l, w, g, frames[t], dh)
			if b.NoRecurrence {
				clear(next)
			}
		}
	}
	stats.Loss = stats.Policy + c.ValueWeight*stats.Value + c.EntropyWeight*stats.Entropy + c.AssessmentWeight*stats.Assessment + n.JointWeight*stats.Joint + n.RewardWeight*stats.Reward
	return stats, g
}

// Temperature fitting preserves the categorical/Bernoulli likelihood targets.
// Only six positive calibration scales change; world/recurrent/policy weights
// remain frozen. Samples must come from a designated calibration split.
func (k *KnowledgeBase) CalibrateNeuralMeta(b CognitiveTrainingBatch, at time.Time) error {
	if b.ComponentV4 != "" && b.ComponentV4 != "world-decoder" {
		return errors.New("V4 calibration supports only world decoder")
	}
	if err := k.ValidateCognitiveBatch(b); err != nil {
		return err
	}
	k.Lock()
	defer k.Unlock()
	s := k.BrainState.CognitiveModel
	if s == nil || s.NeuralMeta == nil {
		return errors.New("calibration requires V2")
	}
	c, l := s.Config, s.Layout
	n := *c.NeuralMeta
	w, err := k.cognitiveWeights(s)
	if err != nil {
		return err
	}
	type sample struct {
		joint   []float64
		allowed []bool
		sa      []float64
		labels  []float64
	}
	samples := []sample{}
	for _, e := range b.Episodes {
		h := make([]float64, c.Hidden)
		allowed := metaAllowed(n, e)
		for t, f := range e.Frames {
			if b.NoRecurrence {
				clear(h)
			}
			x := append([]float64{}, f.Features...)
			if f.NeuralMeta.NoFeedback {
				clear(x[n.ResultOffset : n.ResultOffset+n.Outcomes])
				clear(x[n.OperationOffset : n.OperationOffset+n.Computations])
			}
			h = cognitiveGRU(c, l, w, x, h).h
			if b.TerminalReadoutOnly && t != len(e.Frames)-1 {
				continue
			}
			z := mat(w, s.NeuralMeta.Joint, h)
			addBias(z, w, s.NeuralMeta.JointBias)
			plan := cognitiveGRU(c, l, w, metaPlanFeatures(c, x, *e.VerifiedTarget, f.Action), h)
			sa := mat(w, l.Assessment, plan.h)
			addBias(sa, w, l.AssessmentBias)
			ys := make([]float64, 5)
			for q := range ys {
				if e.SuccessAt >= t && float64(e.SuccessAt-t+1) <= math.Ceil(float64(c.Horizon)*float64(q+1)/5) {
					ys[q] = 1
				}
			}
			samples = append(samples, sample{joint: z, allowed: allowed, sa: sa, labels: ys})
		}
	}
	logs := make([]float64, 6)
	for iteration := 0; iteration < 120; iteration++ {
		grad := make([]float64, 6)
		for _, row := range samples {
			T := math.Exp(logs[0])
			p := metaSoftmax(row.joint, T)
			mass, all, selected := 0.0, 0.0, 0.0
			for j, q := range p {
				all += q * row.joint[j] / T
				if row.allowed[j] {
					mass += q
					selected += q * row.joint[j] / T
				}
			}
			if mass <= 0 {
				return errors.New("calibration likelihood underflow")
			}
			grad[0] += selected/mass - all
			for q, z := range row.sa {
				scaled := z / math.Exp(logs[q+1])
				grad[q+1] += (row.labels[q] - sigmoid(scaled)) * scaled
			}
		}
		for j := range logs {
			if b.ComponentV4 == "world-decoder" && j != 0 {
				continue
			}
			if b.TerminalReadoutOnly && j == 0 {
				continue
			}
			logs[j] = math.Max(-2, math.Min(2, logs[j]-.05*grad[j]/float64(len(samples))))
		}
	}
	for j, id := range s.NeuralMeta.TemperatureNodes {
		if b.ComponentV4 == "world-decoder" && j != 0 {
			continue
		}
		if b.TerminalReadoutOnly && j == 0 {
			continue
		}
		k.adjustCompositionParameter([]NodeID{l.BiasInput}, []NodeID{id}, logs[j], 1, at, true)
	}
	ev := b.Evidence
	s.NeuralMeta.CalibrationEvidence = &ev
	return nil
}
