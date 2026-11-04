package knowledge

// Recompute the target features because BMPS's stop continuation uses the
// policy's sampling contract. Substituting unmixed logits on behavior features
// would correct toward a different policy. The observed trace and reward stay
// unchanged, and the behavior propensity remains in each frame.
func coordinatedTargetFramesV5(s *CognitiveModelState, w, temps []float64, b CoordinatedBatchV5) (Phase4BatchV5, error) {
	var target Phase4BatchV5
	for _, ep := range b.Episodes {
		frames := append([]Phase4FrameV5(nil), ep.Frames...)
		for t, history := range ep.Traces {
			trace := append([]CognitiveInput(nil), history...)
			for i := range trace {
				meta := *trace[i].NeuralMeta
				meta.Mode = "recurrent"
				trace[i].NeuralMeta = &meta
			}
			p, err := coordinatedForwardV5(s, w, temps, trace, Phase4DropV5{})
			if err != nil {
				return Phase4BatchV5{}, err
			}
			frames[t].Features = p.Features
		}
		target.Episodes = append(target.Episodes, Phase4EpisodeV5{Frames: frames})
	}
	return target, nil
}

// G_t = r_t + rho_(t+1)*G_(t+1). The prefix through t corrects the
// sampled state and action. The suffix inside G_t corrects future actions.
// No future outcome is evaluated on an action absent from the trace.
// Ratios, critic baseline, and features are detached for the actor score.
func phase4ImportanceTargets(p *Phase4StateV5, w []float64, b Phase4BatchV5) ([][]float64, [][]float64, phase4VisitationWeights) {
	advantages := make([][]float64, len(b.Episodes))
	returns := make([][]float64, len(b.Episodes))
	weights := phase4VisitationWeights{before: make([][]float64, len(b.Episodes)), through: make([][]float64, len(b.Episodes))}
	for i, ep := range b.Episodes {
		n := len(ep.Frames)
		advantages[i], returns[i] = make([]float64, n), make([]float64, n)
		weights.before[i], weights.through[i] = make([]float64, n), make([]float64, n)
		ratios := make([]float64, n)
		prefix := 1.
		for t, f := range ep.Frames {
			_, pi, value := phase4Forward(p, w, f.Features, f.Mask)
			ratios[t] = pi[f.Action] / f.BehaviorProbability
			weights.before[i][t] = prefix
			prefix *= ratios[t]
			weights.through[i][t] = prefix
			advantages[i][t] = -value
		}
		future := 0.
		for t := n - 1; t >= 0; t-- {
			ret := ep.Frames[t].Reward + future
			returns[i][t] = ret
			advantages[i][t] += ret
			future = ratios[t] * ret
		}
	}
	return advantages, returns, weights
}
