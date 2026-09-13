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
// traces without requiring a semantic label. Patterns are selected by overlap
// between their early learned state and the predicted/actual population state.
// The operation is conservative and never removes a pattern.
func (p *PatternIndex) ReconsolidateFromState(predicted, actual map[NodeID]float64, predictionError float64) {
	if p == nil || predictionError <= 0 || len(predicted) == 0 { return }
	if predictionError > 1 { predictionError = 1 }
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, pattern := range p.patterns {
		if pattern == nil || len(pattern.Sequence) < 2 { continue }
		cueStrength := 0.0
		for i, step := range pattern.Sequence {
			if i >= 3 { break }
			level := predicted[step.NodeID]
			if level <= 0 { break }
			cueStrength += level
		}
		if cueStrength == 0 { continue }
		unexpected := 0.0
		for id, level := range actual {
			if level <= 0 || predicted[id] != 0 { continue }
			unexpected += level
		}
		if unexpected <= 0 { continue }
		adjustment := ReconsolidationGain * predictionError * clamp01(cueStrength/3) * clamp01(unexpected)
		if adjustment == 0 { continue }
		pattern.Weight = clamp01(pattern.Weight - adjustment)
		pattern.Confidence = clamp01(pattern.Confidence - adjustment*0.5)
	}
}
