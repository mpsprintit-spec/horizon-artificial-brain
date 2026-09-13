package knowledge

// ReconsolidationGain limits how strongly a prediction error can modify a
// learned pattern in one transition. Reconsolidation is deliberately gradual:
// an unexpected state changes current strength but never erases history.
const ReconsolidationGain = 0.12

// ReconsolidateCompetition applies a bounded response to prediction error for
// patterns that share a cue. It is structural and evidence-aware; it does not
// infer semantic contradiction from language labels.
func (p *PatternIndex) ReconsolidateCompetition(cue []PatternStep, context []ContextFrame, predictionError float64) {
	if p == nil || len(cue) == 0 || predictionError <= 0 {
		return
	}
	if predictionError > 1 { predictionError = 1 }
	matches := p.CompetingPatterns(cue, context)
	if len(matches) == 0 { return }

	p.mu.Lock()
	defer p.mu.Unlock()
	for _, competition := range matches {
		pattern := p.patterns[competition.PatternID]
		if pattern == nil { continue }
		// Evidence support protects well-supported patterns from being
		// destabilized too aggressively by a single unexpected transition.
		protection := 0.5 + 0.5*competition.EvidenceSupport
		contradiction := competition.Contradiction
		adjustment := ReconsolidationGain * predictionError * protection
		if contradiction > 0 {
			adjustment *= 1 + contradiction
		}
		pattern.Weight = clamp01(pattern.Weight - adjustment)
		pattern.Confidence = clamp01(pattern.Confidence - adjustment*0.5)
	}
}
