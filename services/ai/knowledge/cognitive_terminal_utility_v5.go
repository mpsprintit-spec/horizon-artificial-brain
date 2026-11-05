package knowledge

import (
	"encoding/hex"
	"errors"
	"math"
	"time"
)

// TerminalUtilityProtocolV5 is opt-in offline learning from uniformly
// assigned observed answer counts. It does not alter inference selection.
const TerminalUtilityProtocolV5 = "v5-observed-terminal-utility-v1"

type TerminalAnswerCountsV5 struct {
	Observed, Correct int
}
type TerminalUtilityCaseV5 struct {
	Trace   []CognitiveInput
	Answers [4]TerminalAnswerCountsV5
}
type TerminalUtilityBatchV5 struct {
	Protocol, Objective, DatasetSHA256 string
	Cases                              []TerminalUtilityCaseV5
}
type terminalUtilityRowV5 struct {
	F []float64
	U [5]float64
}

func terminalUtilityRowsV5(s *CognitiveModelState, w, temps []float64, b *TerminalUtilityBatchV5) ([]terminalUtilityRowV5, error) {
	rows := make([]terminalUtilityRowV5, len(b.Cases))
	for i, c := range b.Cases {
		p, e := coordinatedForwardV5(s, w, temps, c.Trace, Phase4DropV5{})
		if e != nil {
			return nil, e
		}
		n := 0
		for _, a := range c.Answers {
			n += a.Observed
		}
		rows[i].F = p.Features
		rows[i].U[4] = .35
		for a, c := range c.Answers {
			rows[i].U[a] = 4 * float64(c.Correct) / float64(n)
		}
	}
	return rows, nil
}
func validateTerminalUtilityV5(s *CognitiveModelState, w, temps []float64, b CoordinatedBatchV5) error {
	t := b.TerminalUtility
	if s == nil || s.Phase4V5 == nil || s.NeuralMeta == nil || s.Config.NeuralMeta == nil || len(b.Episodes) != 0 || t == nil ||
		t.Protocol != TerminalUtilityProtocolV5 || (t.Objective != "policy_value" && t.Objective != "logscore") ||
		len(t.Cases) < 1 || len(t.Cases) > 128 || b.ParametersHash != CognitiveDigest(w) || len(t.DatasetSHA256) != 64 {
		return errors.New("invalid terminal utility batch")
	}
	if _, e := hex.DecodeString(t.DatasetSHA256); e != nil {
		return errors.New("invalid terminal utility dataset digest")
	}
	if e := b.Evidence.Validate(); e != nil {
		return e
	}
	if b.Evidence.Digest != CognitiveDigest(t) {
		return errors.New("terminal utility evidence mismatch")
	}
	for _, c := range t.Cases {
		if len(c.Trace) < 1 || len(c.Trace) > s.Config.Horizon {
			return errors.New("invalid terminal observed history")
		}
		used := make([]bool, s.Config.NeuralMeta.Computations)
		for step, x := range c.Trace {
			if err := validateCoordinatedInputV5(x, s.Config); err != nil {
				return err
			}
			if x.NeuralMeta == nil || x.NeuralMeta.Mode != "recurrent" || x.NoRecurrence ||
				x.NeuralMeta.NoFeedback || x.Reset != (step == 0) ||
				x.NeuralMeta.Remaining != s.Config.Horizon-1-step {
				return errors.New("terminal utility requires ordinary complete observed recurrent history")
			}
			if step > 0 {
				op := -1
				for a := range used {
					if x.Features[s.Config.NeuralMeta.OperationOffset+a] == 1 {
						if op >= 0 {
							return errors.New("ambiguous terminal operation history")
						}
						op = a
					}
				}
				if op < 0 || used[op] {
					return errors.New("missing or repeated terminal operation history")
				}
				used[op] = true
			}
		}
		mask := c.Trace[len(c.Trace)-1].Mask
		if len(mask) != 8 {
			return errors.New("invalid terminal mask")
		}
		for a, legal := range mask {
			if legal != (a >= 3) {
				return errors.New("terminal utility requires all terminal actions and no computation")
			}
		}
		n := 0
		for _, a := range c.Answers {
			if a.Observed < 0 || a.Observed > 4096 || a.Correct < 0 || a.Correct > a.Observed {
				return errors.New("invalid observed answer counts")
			}
			n += a.Observed
		}
		if n < 1 || n > 4096 {
			return errors.New("invalid uniformly assigned trial total")
		}
		// The successful outcomes must come from mutually exclusive labels,
		// but only the executed action's aggregate successes are recorded.
		if _, _, _, e := coordinatedStateV5(s, w, temps, c.Trace); e != nil {
			return e
		}
	}
	return nil
}

func terminalUtilityObjectiveV5(p *Phase4StateV5, w []float64, rows []terminalUtilityRowV5, kind string) (float64, []float64) {
	g := make([]float64, len(w))
	norm := 1 / float64(len(rows))
	loss := 0.
	for _, r := range rows {
		h, pi, _ := phase4Forward(p, w, r.F, []bool{false, false, false, true, true, true, true, true})
		sumU, mean, el := 0., 0., 0.
		for a, u := range r.U {
			q := pi[a+3]
			sumU += u
			mean += q * u
			if q > 0 {
				el += q * math.Log(q)
			}
			if kind == "logscore" {
				loss -= norm * u * math.Log(math.Max(q, 1e-15))
			}
		}
		if kind == "policy_value" {
			loss -= norm * mean
		}
		loss += norm * p.Spec.EntropyWeight * el
		dz := make([]float64, 8)
		for a, u := range r.U {
			q := pi[a+3]
			switch kind {
			case "policy_value":
				dz[a+3] = -q * (u - mean)
			case "logscore":
				dz[a+3] = q*sumU - u
			default:
				panic("unknown outcome objective")
			}
			dz[a+3] = norm * (dz[a+3] + p.Spec.EntropyWeight*q*(math.Log(math.Max(q, 1e-15))-el))
		}
		dh := make([]float64, 16)
		cognitiveLinearBackward(g, dh, w, h, dz, p.Actor, p.ActorBias)
		for j := range dh {
			dh[j] *= 1 - h[j]*h[j]
		}
		cognitiveLinearBackward(g, make([]float64, 58), w, r.F, dh, p.WI, p.BI)
	}
	return loss, g
}
func (k *KnowledgeBase) learnTerminalUtilityV5(s *CognitiveModelState, w, temps []float64, b CoordinatedBatchV5, at time.Time) (CoordinatedLearningV5, error) {
	out := CoordinatedLearningV5{}
	rows, e := terminalUtilityRowsV5(s, w, temps, b.TerminalUtility)
	if e != nil {
		return out, e
	}
	p := s.Phase4V5
	loss, g := terminalUtilityObjectiveV5(p, w, rows, b.TerminalUtility.Objective)
	norm := 0.
	for _, v := range g {
		if !finiteNumber(v) {
			return out, errors.New("nonfinite terminal utility gradient")
		}
		norm += v * v
	}
	norm = math.Sqrt(norm)
	scale := 1 / math.Max(1, norm)
	p.Steps++
	clock := float64(p.Steps)
	for i := p.BaseParameters; i < len(w); i++ {
		d := g[i] * scale
		s.FirstMoment[i] = .9*s.FirstMoment[i] + .1*d
		s.SecondMoment[i] = .999*s.SecondMoment[i] + .001*d*d
		v := w[i] - p.Spec.LearningRate*(s.FirstMoment[i]/(1-math.Pow(.9, clock)))/(math.Sqrt(s.SecondMoment[i]/(1-math.Pow(.999, clock)))+1e-8)
		v = math.Max(-7.9, math.Min(7.9, v))
		edge := s.Layout.Edges[i]
		k.adjustCompositionParameter([]NodeID{edge.From}, []NodeID{edge.To}, v, 1, at, false)
		out.UpdatedHead++
	}
	p.Contract = CoordinatedContractV5
	p.LearnedEpisodes += uint64(len(rows))
	ev := b.Evidence
	p.LastEvidence = &ev
	out.Policy = Phase4LearningV5{Episodes: len(rows), Frames: len(rows), Loss: loss, Policy: loss, GradientNorm: norm, ActorGradientNorm: norm, Steps: p.Steps}
	return out, nil
}
