package knowledge

import "time"

// PatternCompetition describes competing learned traces that share a cue but
// diverge in their continuation. It is an observational signal only: it does
// not assign semantic relation kinds and does not delete either pattern.
type PatternCompetition struct {
	PatternID       PatternID
	SharedSteps     int
	DivergenceAt    int
	Strength        float64
	EvidenceSupport float64
	Contradiction   float64
}

// CompetingPatterns finds learned temporal traces that share the supplied cue
// and then diverge. Competition is structural: node order, learned strength,
// evidence independence/reliability, and contradiction metadata are used.
func (p *PatternIndex) CompetingPatterns(cue []PatternStep, context []ContextFrame) []PatternCompetition {
	cue = normalizeSequence(cue)
	context = normalizeContext(context)
	if len(cue) == 0 || p == nil {
		return nil
	}
	p.mu.RLock()
	defer p.mu.RUnlock()

	out := make([]PatternCompetition, 0)
	for _, pattern := range p.patterns {
		if len(pattern.Sequence) <= len(cue) {
			continue
		}
		shared := 0
		for i := 0; i < len(cue) && i < len(pattern.Sequence); i++ {
			if pattern.Sequence[i].NodeID != cue[i].NodeID {
				break
			}
			shared++
		}
		if shared != len(cue) {
			continue
		}
		strength := clamp01(pattern.Weight) * clamp01(pattern.Confidence)
		support := EvidenceReliability(pattern.Evidence)
		if len(pattern.Evidence) > 0 {
			support *= clamp01(float64(EvidenceIndependence(pattern.Evidence)) / 4)
		}
		contradiction := 0.0
		for _, evidence := range pattern.Evidence {
			if evidence.ContradictionSet != "" {
				value := evidence.Reliability
				if value <= 0 { value = 0.5 }
				if value > contradiction { contradiction = value }
			}
		}
		_ = context
		out = append(out, PatternCompetition{
			PatternID: pattern.ID,
			SharedSteps: shared,
			DivergenceAt: shared,
			Strength: strength,
			EvidenceSupport: support,
			Contradiction: contradiction,
		})
	}
	return out
}

// CompetingAtNextStep returns patterns whose learned continuation differs at
// the first step after a complete cue. The timestamp is intentionally accepted
// so callers can keep structural competition inside the same temporal event
// boundary; it is not used to manufacture a semantic contradiction.
func (p *PatternIndex) CompetingAtNextStep(cue []PatternStep, context []ContextFrame, now time.Time) []PatternCompetition {
	if now.IsZero() { now = time.Now().UTC() }
	return p.CompetingPatterns(cue, context)
}
