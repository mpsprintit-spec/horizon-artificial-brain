package knowledge

// V3 repairs coverage of conditional terminal plans, without changing MUSE's
// BCE assessment, reward regression, BMPS equations or the terminal selector.
// Every sample is a real terminal branch verified by the external environment.
// The observation GRU, actor, class/world belief and V7 synapses are frozen.
import "math"

func terminalReadoutParameterV3(s *CognitiveModelState, i int) bool {
	l, n := s.Layout, s.Config.NeuralMeta
	in := func(t CognitiveTensor) bool { return i >= t.Offset && i < t.Offset+t.Rows*t.Cols }
	if in(l.Assessment) || in(l.AssessmentBias) || in(s.NeuralMeta.Reward) || in(s.NeuralMeta.RewardBias) {
		return true
	}
	if in(l.WI) {
		col := (i - l.WI.Offset) % l.WI.Cols
		return col == n.PlanMarker || col >= n.HypothesisOffset && col < n.HypothesisOffset+n.Classes || col >= n.ActionOffset+n.Computations && col < n.ActionOffset+s.Config.Actions
	}
	return false
}
func cognitiveTerminalReadoutObjectiveV3(s *CognitiveModelState, w []float64, b CognitiveTrainingBatch) (CognitiveLearningResult, []float64) {
	c, l := s.Config, s.Layout
	stats := CognitiveLearningResult{Episodes: len(b.Episodes)}
	g := make([]float64, len(w))
	scale := 1 / float64(len(b.Episodes))
	for _, e := range b.Episodes {
		h := make([]float64, c.Hidden)
		for _, f := range e.Frames {
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
		}
		f := e.Frames[len(e.Frames)-1]
		plan := cognitiveGRU(c, l, w, metaPlanFeatures(c, f.Features, *e.VerifiedTarget, f.Action), h)
		dp := make([]float64, c.Hidden)
		logits := mat(w, l.Assessment, plan.h)
		addBias(logits, w, l.AssessmentBias)
		ds := make([]float64, 5)
		label := 0.
		if e.SuccessAt == len(e.Frames)-1 {
			label = 1
		}
		// The probed plan terminates immediately. Thus all five temporal quantiles
		// share this verified outcome; no eventual policy success is substituted.
		for q, z := range logits {
			stats.Assessment += scale * metaBCE(z, label)
			ds[q] = scale * c.AssessmentWeight * (sigmoid(z) - label)
		}
		cognitiveLinearBackward(g, dp, w, plan.h, ds, l.Assessment, l.AssessmentBias)
		z := mat(w, s.NeuralMeta.Reward, plan.h)
		addBias(z, w, s.NeuralMeta.RewardBias)
		pred := math.Tanh(z[0])
		d := pred - f.Reward
		stats.Reward += scale * .5 * d * d
		cognitiveLinearBackward(g, dp, w, plan.h, []float64{scale * c.NeuralMeta.RewardWeight * d * (1 - pred*pred)}, s.NeuralMeta.Reward, s.NeuralMeta.RewardBias)
		cognitiveGRUBackward(c, l, w, g, plan, dp)
	}
	for i := range g {
		if !terminalReadoutParameterV3(s, i) {
			g[i] = 0
		}
	}
	stats.Loss = c.AssessmentWeight*stats.Assessment + c.NeuralMeta.RewardWeight*stats.Reward
	return stats, g
}
