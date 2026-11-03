package knowledge

// V4 uses the existing categorical partial NLL and Chen REINFORCE/baseline
// objectives, with explicit disjoint optimizer masks. No inference rule or
// learned tensor is added. MUSE's V3 conditional readout remains protected.
import (
	"errors"
	"math"
)

func validateComponentV4(c CognitiveModelConfig, b CognitiveTrainingBatch) error {
	if b.ComponentV4 == "" {
		if b.EntropyWeightV4 != nil {
			return errors.New("entropy schedule requires V4 actor component")
		}
		return nil
	}
	if c.NeuralMeta == nil || b.TerminalReadoutOnly {
		return errors.New("V4 components require native V2 and cannot mix terminal repair")
	}
	switch b.ComponentV4 {
	case "world-decoder":
		if b.LearnPolicy || b.EntropyWeightV4 != nil {
			return errors.New("world-only fitting cannot train policy")
		}
	case "actor-critic", "actor-only":
		if !b.LearnPolicy {
			return errors.New("actor component requires verified on-policy episodes")
		}
	default:
		return errors.New("unknown V4 component mask")
	}
	if b.EntropyWeightV4 != nil && (!finiteNumber(*b.EntropyWeightV4) || *b.EntropyWeightV4 < 0 || *b.EntropyWeightV4 > 1) {
		return errors.New("invalid logged entropy coefficient")
	}
	return nil
}
func componentParameterV4(s *CognitiveModelState, component string, i int) bool {
	in := func(t CognitiveTensor) bool { return i >= t.Offset && i < t.Offset+t.Rows*t.Cols }
	if component == "world-decoder" {
		return in(s.NeuralMeta.Joint) || in(s.NeuralMeta.JointBias)
	}
	return in(s.Layout.Actor) || in(s.Layout.ActorBias) || (component == "actor-critic" && (in(s.Layout.Critic) || in(s.Layout.CriticBias)))
}

// Stable -log sum_{allowed} softmax(z). This is exactly the V2 partial
// categorical likelihood, evaluated as two log-sum-exp terms; no probability
// flooring, invented target distribution, or hidden target in inference.
func partialCategoricalV4(z []float64, allowed []bool) (float64, []float64) {
	ma, ms := math.Inf(-1), math.Inf(-1)
	for i, v := range z {
		ma = math.Max(ma, v)
		if allowed[i] {
			ms = math.Max(ms, v)
		}
	}
	a, s := 0., 0.
	for i, v := range z {
		a += math.Exp(v - ma)
		if allowed[i] {
			s += math.Exp(v - ms)
		}
	}
	g := make([]float64, len(z))
	for i, v := range z {
		g[i] = math.Exp(v-ma) / a
		if allowed[i] {
			g[i] -= math.Exp(v-ms) / s
		}
	}
	return ma + math.Log(a) - ms - math.Log(s), g
}
func cognitiveComponentObjectiveV4(s *CognitiveModelState, w []float64, b CognitiveTrainingBatch, targets cognitiveTargets) (CognitiveLearningResult, []float64) {
	c, l := s.Config, s.Layout
	g := make([]float64, len(w))
	stats := CognitiveLearningResult{Episodes: len(b.Episodes)}
	scale := 1 / float64(len(b.Episodes))
	beta := c.EntropyWeight
	if b.EntropyWeightV4 != nil {
		beta = *b.EntropyWeightV4
	}
	for ei, e := range b.Episodes {
		h := make([]float64, c.Hidden)
		allowed := metaAllowed(*c.NeuralMeta, e)
		for t, f := range e.Frames {
			if b.NoRecurrence {
				clear(h)
			}
			x := append([]float64{}, f.Features...)
			n := c.NeuralMeta
			if f.NeuralMeta.NoFeedback {
				clear(x[n.ResultOffset : n.ResultOffset+n.Outcomes])
				clear(x[n.OperationOffset : n.OperationOffset+n.Computations])
			}
			h = cognitiveGRU(c, l, w, x, h).h
			if b.ComponentV4 == "world-decoder" {
				z := mat(w, s.NeuralMeta.Joint, h)
				addBias(z, w, s.NeuralMeta.JointBias)
				loss, d := partialCategoricalV4(z, allowed)
				stats.Joint += scale * loss
				for j := range d {
					d[j] *= scale * n.JointWeight
				}
				cognitiveLinearBackward(g, make([]float64, c.Hidden), w, h, d, s.NeuralMeta.Joint, s.NeuralMeta.JointBias)
			} else {
				p, _, v := cognitiveHeads(l, w, h, f.Mask)
				adv := targets.Advantages[ei][t]
				stats.Policy -= scale * adv * math.Log(math.Max(1e-15, p[f.Action]))
				ent := 0.
				for _, q := range p {
					if q > 0 {
						ent += q * math.Log(q)
					}
				}
				stats.Entropy += scale * ent
				d := make([]float64, c.Actions)
				for a, q := range p {
					if q > 0 {
						d[a] = scale * (adv*q + beta*q*(math.Log(q)-ent))
						if a == f.Action {
							d[a] -= scale * adv
						}
					}
				}
				cognitiveLinearBackward(g, make([]float64, c.Hidden), w, h, d, l.Actor, l.ActorBias)
				delta := v - targets.Returns[ei][t]
				stats.Value += scale * .5 * delta * delta
				if b.ComponentV4 == "actor-critic" {
					cognitiveLinearBackward(g, make([]float64, c.Hidden), w, h, []float64{scale * c.ValueWeight * delta}, l.Critic, l.CriticBias)
				}
			}
		}
	}
	if b.ComponentV4 == "world-decoder" {
		stats.Loss = c.NeuralMeta.JointWeight * stats.Joint
	} else {
		stats.Loss = stats.Policy + beta*stats.Entropy
		if b.ComponentV4 == "actor-critic" {
			stats.Loss += c.ValueWeight * stats.Value
		}
	}
	return stats, g
}
