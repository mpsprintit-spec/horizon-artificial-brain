package execution

import (
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/plugin"
)

// InquiryExecution is the minimal execution boundary connecting an internally
// selected information-seeking candidate to an observed consequence.
type InquiryExecution struct {
	Candidate knowledge.InquiryCandidate
	Outcome   knowledge.InquiryOutcome
	Score     float64
}

// ExecuteInquiry selects one candidate from an experiential plugin, executes it,
// evaluates the observed consequence against its prediction, and records the
// resulting learning signal. No question text is generated.
func ExecuteInquiry(kb *knowledge.KnowledgeBase, p plugin.ExperientialPlugin, contextData string, now time.Time) (InquiryExecution, bool) {
	if kb == nil || p == nil {
		return InquiryExecution{}, false
	}

	candidates := p.Predict(contextData)
	policy := kb.BrainState.LearningPolicyState
	state := kb.BrainState.InquiryState
	candidate, score, ok := knowledge.SelectInformationSeekingCandidate(
		candidates,
		state.Uncertainty,
		policy.ExplorationBias,
		policy.RepeatObservationBias,
	)
	if !ok || candidate.ActionID == "" {
		return InquiryExecution{}, false
	}

	if now.IsZero() {
		now = time.Now().UTC()
	}
	kb.RecordInquirySelection(
		candidate.Sequence,
		candidate.ActionID,
		state.Uncertainty,
		score,
		score,
		now,
	)

	outcome := p.Execute(contextData)
	if outcome.At.IsZero() {
		outcome.At = now.UTC()
	}
	observedGain, predictionError := knowledge.EvaluateInquiryOutcome(candidate, outcome)
	kb.RecordInquiryOutcome(observedGain, predictionError, outcome.At)

	return InquiryExecution{
		Candidate: candidate,
		Outcome:   outcome,
		Score:     score,
	}, true
}
