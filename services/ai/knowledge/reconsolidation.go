package knowledge

// ReconsolidationGain limits how strongly a prediction error can modify a
// learned pattern in one transition. Reconsolidation is deliberately gradual:
// an unexpected state changes current strength but never erases history.
const ReconsolidationGain = 0.12

// ReconsolidateCompetition applies a bounded response to prediction error for
// patterns that share a cue. It is structural and evidence-aware; it does not
// infer semantic contradiction from language labels.
func (p *PatternIndex) ReconsolidateCompetition(cue []PatternStep, context []ContextFrame, predictionError float64) {
	if p == nil || len(cue) == 0 || predictionError <= 0 { return }
	if predictionError > 1 { predictionError = 1 }
	matches := p.CompetingPatterns(cue, context)
	if len(matches) == 0 { return }
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, competition := range matches {
		pattern := p.patterns[competition.PatternID]
		if pattern == nil { continue }
		protection := 0.5 + 0.5*competition.EvidenceSupport
		adjustment := ReconsolidationGain * predictionError * protection
		if competition.Contradiction > 0 { adjustment *= 1 + competition.Contradiction }
		pattern.Weight = clamp01(pattern.Weight - adjustment)
		pattern.Confidence = clamp01(pattern.Confidence - adjustment*0.5)
	}
}

// ReconsolidateFromState connects an observed prediction error to learned
// traces without requiring a semantic label. A trace is adapted only when its
// learned result was part of the predicted state and that result failed to
// occur. This prevents an unrelated unexpected population from weakening every
// trace that merely shares some active nodes.
func (p *PatternIndex) ReconsolidateFromState(predicted, actual map[NodeID]float64, predictionError float64) {
	if p == nil || predictionError <= 0 || len(predicted) == 0 { return }
	if predictionError > 1 { predictionError = 1 }
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, pattern := range p.patterns {
		if pattern == nil || len(pattern.Sequence) < 2 { continue }

		predictedResult := clamp01(predicted[pattern.Result])
		actualResult := clamp01(actual[pattern.Result])
		resultMismatch := predictedResult - actualResult
		if resultMismatch <= 0 {
			continue
		}

		// Measure support from the learned cue, excluding the result itself.
		// For distributed outcomes this remains valid even when the result is
		// the final unit of a multi-node population.
		cueStrength := 0.0
		for _, step := range pattern.Sequence {
			if step.NodeID == pattern.Result {
				break
			}
			level := clamp01(predicted[step.NodeID])
			if level <= 0 {
				break
			}
			cueStrength += level
		}
		if cueStrength <= 0 {
			continue
		}

		unexpected := 0.0
		for id, level := range actual {
			if level <= 0 || predicted[id] != 0 {
				continue
			}
			unexpected += level
		}
		unexpectedFactor := 0.5 + 0.5*clamp01(unexpected)
		adjustment := ReconsolidationGain *
			predictionError *
			clamp01(cueStrength/3) *
			resultMismatch *
			unexpectedFactor
		if adjustment <= 0 {
			continue
		}
		pattern.Weight = clamp01(pattern.Weight - adjustment)
		pattern.Confidence = clamp01(pattern.Confidence - adjustment*0.5)
	}
}
