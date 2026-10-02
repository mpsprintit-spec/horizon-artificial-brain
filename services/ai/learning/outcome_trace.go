package learning

import (
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// LearnOutcomeTrace reinforces each observed target-to-outcome transition in
// the existing dynamic neural substrate. Distinct outcomes remain distinct
// connections; repeated or contradictory evidence therefore changes relative
// predictive strength instead of replacing prior evidence.
func (l *LearningUnit) LearnOutcomeTrace(targets, outcomes []knowledge.NodeID, weight, reliability float64, evidence knowledge.ExperienceEvidence, now time.Time) {
	if l == nil || l.Kb == nil || l.Kb.Registry == nil {
		return
	}
	if len(targets) == 0 || len(outcomes) == 0 {
		return
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	weight = clampLearning01(weight)
	reliability = clampLearning01(reliability)
	if weight == 0 {
		weight = 0.5
	}
	if reliability == 0 {
		reliability = 0.5
	}

	for _, targetID := range targets {
		target := l.Kb.Registry.GetByID(targetID)
		if target == nil {
			continue
		}
		for _, outcomeID := range outcomes {
			outcome := l.Kb.Registry.GetByID(outcomeID)
			if outcome == nil || outcome.ID == target.ID {
				continue
			}
			l.Kb.ConnectAt(target, outcome, weight, reliability, false, now)

			// Preserve provenance in the canonical Pattern substrate as part of
			// the same learned transition. The dynamic synapse carries adaptive
			// predictive strength; the pattern trace retains the evidence that
			// produced that transition. No parallel inquiry-specific evidence
			// store is introduced.
			if l.Kb.Patterns != nil {
				sequence := []knowledge.PatternStep{
					{NodeID: target.ID, Position: 0, Activation: 1},
					{NodeID: outcome.ID, Position: 1, Activation: 1},
				}
				l.Kb.Patterns.LearnTraceWithEvidence(
					sequence,
					nil,
					outcome.ID,
					weight,
					reliability,
					evidence,
				)
			}
		}
	}

}

func clampLearning01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
