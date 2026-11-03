package knowledge

import (
	"errors"
	"fmt"
	"math"
	"time"
)

type CognitiveTrainingFrame struct {
	NeuralMeta *NeuralMetaInput `json:"neural_meta,omitempty"`
	Features   []float64        `json:"features"`
	Mask       []bool           `json:"mask"`
	Action     int              `json:"action"`
	Reward     float64          `json:"reward"`
	Draw       float64          `json:"draw"`
	Behaviour  []float64        `json:"behaviour,omitempty"`
}
type CognitiveTrainingEpisode struct {
	VerifiedTarget     *int                     `json:"verified_target,omitempty"`
	VerifiedOperations []VerifiedMentalResult   `json:"verified_operations,omitempty"`
	Frames             []CognitiveTrainingFrame `json:"frames"`
	SuccessAt          int                      `json:"success_at"` // -1 = verified failure, otherwise first success
}
type CognitiveTrainingBatch struct {
	ComponentV4         string                     `json:"component_v4,omitempty"`
	EntropyWeightV4     *float64                   `json:"entropy_weight_v4,omitempty"`
	TerminalReadoutOnly bool                       `json:"terminal_readout_only,omitempty"`
	Episodes            []CognitiveTrainingEpisode `json:"episodes"`
	Evidence            CognitiveEvidence          `json:"evidence"`
	ParametersHash      string                     `json:"parameters_hash"`
	LearnPolicy         bool                       `json:"learn_policy"`
	NoRecurrence        bool                       `json:"no_recurrence,omitempty"`
}

func (b CognitiveTrainingBatch) Validate(c CognitiveModelConfig) error {
	if err := validateComponentV4(c, b); err != nil {
		return err
	}
	if b.TerminalReadoutOnly && (b.LearnPolicy || c.NeuralMeta == nil) {
		return errors.New("terminal readout fitting requires V2 verified auxiliary data, no policy loss")
	}
	if len(b.Episodes) == 0 {
		return errors.New("empty cognitive training batch")
	}
	if err := b.Evidence.Validate(); err != nil {
		return err
	}
	if b.Evidence.Digest != CognitiveDigest(b.Episodes) {
		return errors.New("feedback digest disagrees with episodes")
	}
	for _, e := range b.Episodes {
		if len(e.Frames) == 0 || len(e.Frames) > c.Horizon || e.SuccessAt < -1 || e.SuccessAt >= len(e.Frames) {
			return errors.New("invalid verified trajectory/horizon")
		}
		for _, f := range e.Frames {
			if err := (CognitiveInput{Features: f.Features, Mask: f.Mask, Draw: f.Draw, NeuralMeta: f.NeuralMeta}).Validate(c); err != nil {
				return err
			}
			if f.Action < 0 || f.Action >= c.Actions || !f.Mask[f.Action] || !finiteNumber(f.Reward) {
				return errors.New("invalid training action or outcome")
			}
			if b.TerminalReadoutOnly {
				n := c.NeuralMeta
				for j := n.Computations; j < c.Actions; j++ {
					if f.Features[n.ActionOffset+j] != 0 {
						return errors.New("readout repair requires inactive terminal-plan channels on real observations")
					}
				}
			}
			if b.LearnPolicy && len(f.Behaviour) != c.Actions {
				return errors.New("on-policy training requires recorded behaviour probabilities")
			}
		}
		if c.NeuralMeta != nil {
			if err := validateVerifiedMentalEpisode(c, e, b.LearnPolicy); err != nil {
				return err
			}
		}
	}
	return nil
}

type CognitiveLearningResult struct {
	Phase2Times       *Phase2UpdateTimesV5 `json:"phase2_times,omitempty"` // output-only timings, never persisted as neural state or replay input
	Joint             float64              `json:"joint,omitempty"`
	Reward            float64              `json:"reward,omitempty"`
	Loss              float64              `json:"loss"`
	Policy            float64              `json:"policy"`
	Value             float64              `json:"value"`
	Entropy           float64              `json:"entropy"`
	Assessment        float64              `json:"assessment"`
	World             float64              `json:"world"`
	GradientNorm      float64              `json:"gradient_norm"`
	ClippedParameters int                  `json:"clipped_parameters"`
	Episodes          int                  `json:"episodes"`
	OptimizerStep     uint64               `json:"optimizer_step"`
}
type cognitiveTargets struct{ Returns, Advantages [][]float64 }

func makeCognitiveTargets(c CognitiveModelConfig, l CognitiveLayout, w []float64, b CognitiveTrainingBatch, check bool) (cognitiveTargets, error) {
	targets := cognitiveTargets{Returns: make([][]float64, len(b.Episodes)), Advantages: make([][]float64, len(b.Episodes))}
	for ei, e := range b.Episodes {
		h := make([]float64, c.Hidden)
		vs := make([]float64, len(e.Frames))
		for t, f := range e.Frames {
			if b.NoRecurrence {
				h = make([]float64, c.Hidden)
			}
			x := f.Features
			if c.NeuralMeta != nil && f.NeuralMeta.NoFeedback {
				x = append([]float64{}, x...)
				n := c.NeuralMeta
				clear(x[n.ResultOffset : n.ResultOffset+n.Outcomes])
				clear(x[n.OperationOffset : n.OperationOffset+n.Computations])
			}
			h = cognitiveGRU(c, l, w, x, h).h
			p, _, v := cognitiveHeads(l, w, h, f.Mask)
			vs[t] = v
			if check && b.LearnPolicy {
				for a, q := range p {
					if !finiteNumber(f.Behaviour[a]) || math.Abs(q-f.Behaviour[a]) > 1e-8 {
						return targets, fmt.Errorf("trajectory is off-policy at episode %d frame %d", ei, t)
					}
				}
				if samplePolicy(p, f.Draw) != f.Action {
					return targets, errors.New("training action was not sampled from frozen behaviour")
				}
			}
		}
		targets.Returns[ei] = make([]float64, len(e.Frames))
		targets.Advantages[ei] = make([]float64, len(e.Frames))
		r := 0.0
		for t := len(e.Frames) - 1; t >= 0; t-- {
			r = e.Frames[t].Reward + c.Gamma*r
			targets.Returns[ei][t] = r
			targets.Advantages[ei][t] = r - vs[t]
		}
	}
	return targets, nil
}
func cognitiveLinearBackward(g, dh, w, h, dy []float64, t, bias CognitiveTensor) {
	for i, d := range dy {
		g[bias.Offset+i] += d
		for j, v := range h {
			idx := t.Offset + i*t.Cols + j
			g[idx] += d * v
			dh[j] += w[idx] * d
		}
	}
}
func cognitiveGRUBackward(c CognitiveModelConfig, l CognitiveLayout, w, g []float64, f gruFrame, dh []float64) []float64 {
	h := c.Hidden
	dx := make([]float64, 3*h)
	du := make([]float64, 3*h)
	prev := make([]float64, h)
	for i, d := range dh {
		dn := d * (1 - f.z[i])
		dz := d * (f.hp[i] - f.n[i])
		an := dn * (1 - f.n[i]*f.n[i])
		dx[2*h+i] = an
		du[2*h+i] = an * f.r[i]
		dx[i] = an * f.u[i] * f.r[i] * (1 - f.r[i])
		du[i] = dx[i]
		dx[h+i] = dz * f.z[i] * (1 - f.z[i])
		du[h+i] = dx[h+i]
		prev[i] += d * f.z[i]
	}
	for i := range dx {
		g[l.BI.Offset+i] += dx[i]
		g[l.BH.Offset+i] += du[i]
		for j, x := range f.x {
			g[l.WI.Offset+i*c.Inputs+j] += dx[i] * x
		}
		for j, x := range f.hp {
			idx := l.WH.Offset + i*h + j
			g[idx] += du[i] * x
			prev[j] += du[i] * w[idx]
		}
	}
	return prev
}
func cognitiveBatchObjective(c CognitiveModelConfig, l CognitiveLayout, w []float64, b CognitiveTrainingBatch, targets cognitiveTargets) (CognitiveLearningResult, []float64) {
	stats := CognitiveLearningResult{Episodes: len(b.Episodes)}
	g := make([]float64, len(w))
	norm := 1 / float64(len(b.Episodes))
	for ei, e := range b.Episodes {
		frames := make([]gruFrame, len(e.Frames))
		ps := make([][]float64, len(frames))
		sas := make([][]float64, len(frames))
		values := make([]float64, len(frames))
		h := make([]float64, c.Hidden)
		for t, f := range e.Frames {
			if b.NoRecurrence {
				h = make([]float64, c.Hidden)
			}
			frames[t] = cognitiveGRU(c, l, w, f.Features, h)
			h = frames[t].h
			ps[t], sas[t], values[t] = cognitiveHeads(l, w, h, f.Mask)
		}
		next := make([]float64, c.Hidden)
		for t := len(frames) - 1; t >= 0; t-- {
			f := e.Frames[t]
			h := frames[t].h
			dh := append([]float64{}, next...)
			p, sa := ps[t], sas[t]
			if b.LearnPolicy {
				adv := targets.Advantages[ei][t]
				stats.Policy -= norm * adv * math.Log(math.Max(1e-15, p[f.Action]))
				dy := make([]float64, c.Actions)
				el := 0.0
				for _, q := range p {
					if q > 0 {
						el += q * math.Log(q)
					}
				}
				stats.Entropy += norm * el
				for a, q := range p {
					if q == 0 {
						continue
					}
					d := adv * q
					if a == f.Action {
						d -= adv
					}
					dy[a] = norm * (d + c.EntropyWeight*q*(math.Log(q)-el))
				}
				cognitiveLinearBackward(g, dh, w, h, dy, l.Actor, l.ActorBias)
				delta := values[t] - targets.Returns[ei][t]
				stats.Value += norm * .5 * delta * delta
				cognitiveLinearBackward(g, dh, w, h, []float64{norm * c.ValueWeight * delta}, l.Critic, l.CriticBias)
			}
			dy := make([]float64, 5)
			for q, prob := range sa {
				limit := math.Ceil(float64(c.Horizon) * float64(q+1) / 5)
				y := 0.0
				if e.SuccessAt >= t && float64(e.SuccessAt-t+1) <= limit {
					y = 1
				}
				stats.Assessment -= norm * (y*math.Log(math.Max(prob, 1e-15)) + (1-y)*math.Log(math.Max(1-prob, 1e-15)))
				dy[q] = norm * c.AssessmentWeight * (prob - y)
			}
			cognitiveLinearBackward(g, dh, w, h, dy, l.Assessment, l.AssessmentBias)
			if t+1 < len(frames) {
				pred := cognitiveDecode(l, w, h, f.Action, c.Actions)
				xx := append(append([]float64{}, h...), make([]float64, c.Actions)...)
				xx[len(h)+f.Action] = 1
				dWorld := make([]float64, c.Inputs)
				for i, v := range pred {
					delta := v - e.Frames[t+1].Features[i]
					stats.World += norm * .5 * delta * delta / float64(c.Inputs)
					dWorld[i] = norm * c.WorldWeight * delta * (1 - v*v) / float64(c.Inputs)
				}
				extra := make([]float64, len(xx))
				cognitiveLinearBackward(g, extra, w, xx, dWorld, l.Decoder, l.DecoderBias)
				for j := range dh {
					dh[j] += extra[j]
				}
			}
			next = cognitiveGRUBackward(c, l, w, g, frames[t], dh)
			if b.NoRecurrence {
				clear(next)
			}
		}
	}
	stats.Loss = stats.Policy + c.ValueWeight*stats.Value + c.EntropyWeight*stats.Entropy + c.AssessmentWeight*stats.Assessment + c.WorldWeight*stats.World
	return stats, g
}
func (k *KnowledgeBase) ValidateCognitiveBatch(b CognitiveTrainingBatch) error {
	k.RLock()
	defer k.RUnlock()
	s := k.BrainState.CognitiveModel
	if s == nil {
		return errors.New("cognitive model not initialized")
	}
	if err := b.Validate(s.Config); err != nil {
		return err
	}
	w, err := k.cognitiveWeights(s)
	if err != nil {
		return err
	}
	if CognitiveDigest(w) != b.ParametersHash {
		return errors.New("stale cognitive training parameter hash")
	}
	_, err = makeCognitiveTargets(s.Config, s.Layout, w, b, true)
	return err
}
func (k *KnowledgeBase) LearnCognitiveBatch(b CognitiveTrainingBatch, at time.Time) (CognitiveLearningResult, error) {
	if err := k.ValidateCognitiveBatch(b); err != nil {
		return CognitiveLearningResult{}, err
	}
	k.Lock()
	defer k.Unlock()
	s := k.BrainState.CognitiveModel
	w, err := k.cognitiveWeights(s)
	if err != nil {
		return CognitiveLearningResult{}, err
	}
	if CognitiveDigest(w) != b.ParametersHash {
		return CognitiveLearningResult{}, errors.New("cognitive parameters changed before update")
	}
	targets, err := makeCognitiveTargets(s.Config, s.Layout, w, b, true)
	if err != nil {
		return CognitiveLearningResult{}, err
	}
	var stats CognitiveLearningResult
	var g []float64
	if b.ComponentV4 != "" {
		stats, g = cognitiveComponentObjectiveV4(s, w, b, targets)
	} else if b.TerminalReadoutOnly {
		stats, g = cognitiveTerminalReadoutObjectiveV3(s, w, b)
	} else if s.NeuralMeta != nil {
		stats, g = cognitiveBatchObjectiveV2(s, w, b, targets)
	} else {
		stats, g = cognitiveBatchObjective(s.Config, s.Layout, w, b, targets)
	}
	sum := 0.0
	for _, d := range g {
		if !finiteNumber(d) {
			return stats, errors.New("nonfinite neural gradient")
		}
		sum += d * d
	}
	stats.GradientNorm = math.Sqrt(sum)
	scale := 1.0
	if stats.GradientNorm > 1 {
		scale = 1 / stats.GradientNorm
	}
	s.OptimizerStep++
	t := float64(s.OptimizerStep)
	for i, d := range g {
		if b.ComponentV4 != "" && !componentParameterV4(s, b.ComponentV4, i) {
			continue
		}
		if b.TerminalReadoutOnly && !terminalReadoutParameterV3(s, i) {
			continue
		}
		d *= scale
		s.FirstMoment[i] = .9*s.FirstMoment[i] + .1*d
		s.SecondMoment[i] = .999*s.SecondMoment[i] + .001*d*d
		delta := s.Config.LearningRate * (s.FirstMoment[i] / (1 - math.Pow(.9, t))) / (math.Sqrt(s.SecondMoment[i]/(1-math.Pow(.999, t))) + 1e-8)
		v := w[i] - delta
		if math.Abs(v) > 7.9 {
			stats.ClippedParameters++
			v = math.Copysign(7.9, v)
		}
		e := s.Layout.Edges[i]
		k.adjustCompositionParameter([]NodeID{e.From}, []NodeID{e.To}, v, 1, at, false)
	}
	s.LearnedEpisodes += uint64(len(b.Episodes))
	ev := b.Evidence
	s.LastEvidence = &ev
	stats.OptimizerStep = s.OptimizerStep
	return stats, nil
}
