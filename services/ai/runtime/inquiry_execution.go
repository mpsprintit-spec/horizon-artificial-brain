package runtime

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/activation"
)

// InquiryProposal is the safety-neutral bridge between cognitive inquiry
// selection and the policy/authorization boundary. Creating a proposal does
// not mutate the Brain and does not authorize execution.
type InquiryProposal struct {
	BrainIdentity string
	Sequence     uint64
	CandidateID  string
	Action       InquiryAction
	Intent       string
	InformationTarget string
	Confidence   float64
	Reversibility bool
	RiskLevel    RiskLevel
	RequiresAuthorization bool
	CreatedAt    time.Time
}

// InquiryExecution is the only runtime representation that can cross the
// execution boundary. It contains an authorized ExecutionRequest plus the
// originating inquiry identity so the executor can reject mismatched work.
type InquiryExecution struct {
	Proposal InquiryProposal
	Request  ExecutionRequest
	// Prediction is the neural expectation captured immediately before
	// external inquiry execution. It is provenance, not a second memory store.
	Prediction activation.Prediction
	PredictionCapturedAt time.Time
}

// BuildInquiryProposal converts one selected inquiry evaluation into a
// safety-neutral proposal. It never executes the selected action.
func BuildInquiryProposal(agenda InquiryAgenda, informationTarget string, at time.Time) (InquiryProposal, error) {
	if agenda.BrainIdentity == "" {
		return InquiryProposal{}, errors.New("inquiry agenda brain identity is required")
	}
	if agenda.Selected == nil {
		return InquiryProposal{}, errors.New("inquiry agenda has no selected candidate")
	}
	selected := *agenda.Selected
	if selected.CandidateID == "" || selected.Action == "" {
		return InquiryProposal{}, errors.New("selected inquiry candidate is incomplete")
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	action := selected.Action
	reversible := true
	risk := RiskLow
	if selected.RequiresAuthorization {
		risk = RiskMedium
		reversible = false
	}
	return InquiryProposal{
		BrainIdentity: agenda.BrainIdentity,
		Sequence: agenda.Sequence,
		CandidateID: selected.CandidateID,
		Action: action,
		Intent: fmt.Sprintf("inquiry:%s", action),
		InformationTarget: strings.TrimSpace(informationTarget),
		Confidence: clamp01(1 - selected.InformationValue),
		Reversibility: reversible,
		RiskLevel: risk,
		RequiresAuthorization: selected.RequiresAuthorization,
		CreatedAt: at.UTC(),
	}, nil
}

// InquiryRecommendation translates a proposal into the existing safety
// boundary. It does not bypass risk assessment or authorization.
func InquiryRecommendation(proposal InquiryProposal) (Recommendation, error) {
	if proposal.BrainIdentity == "" || proposal.Sequence == 0 {
		return Recommendation{}, errors.New("inquiry proposal identity is incomplete")
	}
	if proposal.CandidateID == "" || proposal.Action == "" {
		return Recommendation{}, errors.New("inquiry proposal candidate is incomplete")
	}
	requestID := fmt.Sprintf("inquiry-%d-%s", proposal.Sequence, proposal.CandidateID)
	return Recommendation{
		RequestID: requestID,
		BrainIdentity: proposal.BrainIdentity,
		Intent: proposal.Intent,
		Confidence: proposal.Confidence,
		EvidenceRefs: []string{fmt.Sprintf("inquiry-sequence:%d", proposal.Sequence), "candidate:" + proposal.CandidateID},
		Reversibility: proposal.Reversibility,
		RiskLevel: proposal.RiskLevel,
		CreatedAt: proposal.CreatedAt,
	}, nil
}

// PrepareInquiryExecution performs risk assessment and records the explicit
// authorization decision. It never executes an external/device action.
func PrepareInquiryExecution(boundary SafetyBoundary, proposal InquiryProposal, approver string, expiry time.Time) (InquiryExecution, AuthorizationDecision, error) {
	recommendation, err := InquiryRecommendation(proposal)
	if err != nil {
		return InquiryExecution{}, AuthorizationDecision{}, err
	}
	assessment, err := boundary.Assess(recommendation)
	if err != nil {
		return InquiryExecution{}, AuthorizationDecision{}, err
	}
	decision, err := boundary.Authorize(recommendation, assessment, approver)
	if err != nil {
		return InquiryExecution{}, AuthorizationDecision{}, err
	}
	if !decision.Authorized {
		return InquiryExecution{Proposal: proposal}, decision, nil
	}
	request, err := boundary.BuildExecutionRequest(recommendation, assessment, decision, expiry)
	if err != nil {
		return InquiryExecution{}, decision, err
	}
	return InquiryExecution{Proposal: proposal, Request: request}, decision, nil
}

// ValidateInquiryExecution is an execution-side guard. It checks that the
// authorized request still belongs to the proposal and has not expired.
// It does not execute the action and does not mutate the Brain.
func ValidateInquiryExecution(execution InquiryExecution, now time.Time) error {
	if execution.Proposal.BrainIdentity == "" || execution.Request.BrainIdentity == "" {
		return errors.New("inquiry execution brain identity is required")
	}
	if execution.Proposal.BrainIdentity != execution.Request.BrainIdentity {
		return errors.New("inquiry execution brain identity mismatch")
	}
	expectedID := fmt.Sprintf("inquiry-%d-%s", execution.Proposal.Sequence, execution.Proposal.CandidateID)
	if execution.Request.RequestID != expectedID {
		return errors.New("inquiry execution request identity mismatch")
	}
	if !execution.Request.Authorized() {
		return errors.New("inquiry execution is not authorized")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if !execution.Request.Expiry.After(now.UTC()) {
		return errors.New("inquiry execution request is expired")
	}
	return nil
}


// CaptureInquiryPrediction binds an authorized inquiry to the exact neural
// prediction that existed immediately before external execution.
func (r *BrainRuntime) CaptureInquiryPrediction(execution InquiryExecution, at time.Time) (InquiryExecution, error) {
	if r == nil || r.activation == nil {
		return InquiryExecution{}, errors.New("brain runtime is not initialized")
	}
	if err := ValidateInquiryExecution(execution, at); err != nil {
		return InquiryExecution{}, err
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	execution.Prediction = r.activation.PredictionSnapshot()
	execution.PredictionCapturedAt = at.UTC()
	return execution, nil
}
