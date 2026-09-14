package knowledge

import (
	"sort"
	"time"
)

// PatternCompetition is a structural competition signal between learned
// temporal traces. It records a trace's continuation, its relative support,
// and the other learned continuations it competes with. It does not assign
// semantic relation kinds and it never deletes a competing trace.
type PatternCompetition struct {
	PatternID        PatternID
	SharedSteps      int
	DivergenceAt     int
	Continuation     NodeID
	Strength         float64
	EvidenceSupport  float64
	Contradiction    float64
	ContextFit       float64
	CompetitiveScore float64
	OpponentIDs      []PatternID
}

// CompetingPatterns finds learned traces that share the complete cue and
// actually compete at the next temporal step. Competition is pairwise: two
// traces are competitors only when their next learned NodeID differs. The
// score combines learned strength, provenance support, contradiction load,
// and contextual fit. No semantic ontology is introduced.
func (p *PatternIndex) CompetingPatterns(cue []PatternStep, context []ContextFrame) []PatternCompetition {
	cue = normalizeSequence(cue)
	context = normalizeContext(context)
	if len(cue) == 0 || p == nil {
		return nil
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	type candidate struct {
		pattern       *PatternSynapse
		continuation  NodeID
		strength      float64
		support       float64
		contradiction float64
		contextFit    float64
		score         float64
	}

	candidates := make([]candidate, 0)
	for _, pattern := range p.patterns {
		if len(pattern.Sequence) <= len(cue) {
			continue
		}
		if !prefixMatches(pattern.Sequence, cue) {
			continue
		}

		continuation := pattern.Sequence[len(cue)].NodeID
		strength := clamp01(pattern.Weight) * clamp01(pattern.Confidence)
		support := EvidenceReliability(pattern.Evidence)
		if len(pattern.Evidence) > 0 {
			support *= clamp01(float64(EvidenceIndependence(pattern.Evidence)) / 4)
		}
		contradiction := ContradictionLoad(pattern.Evidence, contradictionSetFor(pattern.Evidence))
		contextFit := contextSimilarity(pattern.Context, context)

		// Contradiction reduces support; it does not remove the trace. Context
		// is a contextual prior, while learned strength remains the dominant
		// signal. The constants are bounded and continuous.
		score := clamp01(strength*.55 + support*.20 + contextFit*.15 - contradiction*.10)
		candidates = append(candidates, candidate{
			pattern: pattern, continuation: continuation, strength: strength,
			support: support, contradiction: contradiction, contextFit: contextFit,
			score: score,
		})
	}

	out := make([]PatternCompetition, 0)
	for _, current := range candidates {
		opponents := make([]PatternID, 0)
		for _, other := range candidates {
			if other.pattern.ID == current.pattern.ID || other.continuation == current.continuation {
				continue
			}
			opponents = append(opponents, other.pattern.ID)
		}
		sort.Slice(opponents, func(i, j int) bool { return opponents[i] < opponents[j] })
		if len(opponents) == 0 {
			continue
		}

		bestOpponent := 0.0
		for _, other := range candidates {
			if other.pattern.ID == current.pattern.ID || other.continuation == current.continuation {
				continue
			}
			if other.score > bestOpponent {
				bestOpponent = other.score
			}
		}
		margin := clamp01(current.score - bestOpponent + 0.5)
		competitiveScore := clamp01(current.score*.75 + margin*.25)

		out = append(out, PatternCompetition{
			PatternID: current.pattern.ID,
			SharedSteps: len(cue),
			DivergenceAt: len(cue),
			Continuation: current.continuation,
			Strength: current.strength,
			EvidenceSupport: current.support,
			Contradiction: current.contradiction,
			ContextFit: current.contextFit,
			CompetitiveScore: competitiveScore,
			OpponentIDs: opponents,
		})
	}

	// This API is a ranking surface: callers must receive the same order for
	// the same neural state regardless of Go map iteration order. PatternID is
	// the stable tie-breaker.
	sort.Slice(out, func(i, j int) bool {
		if out[i].CompetitiveScore == out[j].CompetitiveScore {
			return out[i].PatternID < out[j].PatternID
		}
		return out[i].CompetitiveScore > out[j].CompetitiveScore
	})
	return out
}

// CompetingAtNextStep keeps the temporal API explicit. The timestamp marks the
// event boundary for callers; learned traces themselves remain persistent and
// are not mutated merely because competition is evaluated.
func (p *PatternIndex) CompetingAtNextStep(cue []PatternStep, context []ContextFrame, now time.Time) []PatternCompetition {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return p.CompetingPatterns(cue, context)
}

func prefixMatches(sequence, cue []PatternStep) bool {
	if len(sequence) <= len(cue) {
		return false
	}
	for i := range cue {
		if sequence[i].NodeID != cue[i].NodeID {
			return false
		}
	}
	return true
}

func contradictionSetFor(evidence []ExperienceEvidence) string {
	for _, e := range evidence {
		if e.ContradictionSet != "" {
			return e.ContradictionSet
		}
	}
	return ""
}
