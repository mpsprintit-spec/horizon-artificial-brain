package runtime

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// RiskLevel is a policy-side classification. It is not a cognitive signal and
// must never be inferred as permission to execute an action.
type RiskLevel string

const (
	RiskLow      RiskLevel = "low"
	RiskMedium   RiskLevel = "medium"
	RiskHigh     RiskLevel = "high"
	RiskCritical RiskLevel = "critical"
)

type Recommendation struct {
	RequestID      string
	BrainIdentity  string
	Intent         string
	Confidence     float64
	EvidenceRefs   []string
	Reversibility  bool
	RiskLevel      RiskLevel
	CreatedAt      time.Time
}

type RiskAssessment struct {
	RequestID     string
	RiskLevel     RiskLevel
	Reasons       []string
	RequiresHuman bool
}

type AuthorizationDecision struct {
	RequestID       string
	Authorized      bool
	Approver        string
	DecisionAt      time.Time
	Reason          string
	RequiredApproval bool
}

type ExecutionRequest struct {
	RequestID      string
	BrainIdentity  string
	Intent         string
	RiskLevel      RiskLevel
	Confidence     float64
	EvidenceRefs   []string
	Reversibility  bool
	RequiredApproval bool
	Expiry         time.Time
	IdempotencyKey string
}

// SafetyBoundary keeps recommendation, risk, authorization and execution as
// separate stages. Cognition can produce a recommendation, but it cannot
// directly authorize or execute it.
type SafetyBoundary struct {
	Now func() time.Time
}

func (s SafetyBoundary) Assess(recommendation Recommendation) (RiskAssessment, error) {
	if recommendation.RequestID == "" || recommendation.BrainIdentity == "" || strings.TrimSpace(recommendation.Intent) == "" {
		return RiskAssessment{}, errors.New("recommendation is missing required identity or intent")
	}
	level := recommendation.RiskLevel
	if level == "" {
		level = RiskMedium
	}
	assessment := RiskAssessment{RequestID: recommendation.RequestID, RiskLevel: level}
	switch level {
	case RiskLow:
		assessment.Reasons = []string{"low-risk policy classification"}
	case RiskMedium:
		assessment.Reasons = []string{"medium-risk action requires policy review"}
		assessment.RequiresHuman = true
	case RiskHigh, RiskCritical:
		assessment.Reasons = []string{"high-impact or irreversible action requires explicit human authorization"}
		assessment.RequiresHuman = true
	default:
		return RiskAssessment{}, fmt.Errorf("unknown risk level %q", level)
	}
	return assessment, nil
}

// Authorize requires an explicit authorization decision. Confidence is
// deliberately never used as an authorization predicate.
func (s SafetyBoundary) Authorize(recommendation Recommendation, assessment RiskAssessment, approver string) (AuthorizationDecision, error) {
	if recommendation.RequestID == "" || assessment.RequestID != recommendation.RequestID {
		return AuthorizationDecision{}, errors.New("recommendation and risk assessment request IDs do not match")
	}
	decision := AuthorizationDecision{
		RequestID: recommendation.RequestID,
		DecisionAt: s.now(),
		RequiredApproval: assessment.RequiresHuman,
	}
	if assessment.RequiresHuman {
		if strings.TrimSpace(approver) == "" {
			decision.Reason = "explicit human authorization is required"
			return decision, nil
		}
		decision.Authorized = true
		decision.Approver = approver
		decision.Reason = "explicit human authorization supplied"
		return decision, nil
	}
	decision.Authorized = true
	decision.Reason = "policy permits low-risk execution"
	return decision, nil
}

func (s SafetyBoundary) BuildExecutionRequest(recommendation Recommendation, assessment RiskAssessment, authorization AuthorizationDecision, expiry time.Time) (ExecutionRequest, error) {
	if !authorization.Authorized {
		return ExecutionRequest{}, errors.New("execution is not authorized")
	}
	if recommendation.RequestID != assessment.RequestID || recommendation.RequestID != authorization.RequestID {
		return ExecutionRequest{}, errors.New("execution request identity mismatch")
	}
	if expiry.IsZero() {
		return ExecutionRequest{}, errors.New("execution request expiry is required")
	}
	if !expiry.After(s.now()) {
		return ExecutionRequest{}, errors.New("execution request is expired")
	}
	return ExecutionRequest{
		RequestID: recommendation.RequestID,
		BrainIdentity: recommendation.BrainIdentity,
		Intent: recommendation.Intent,
		RiskLevel: assessment.RiskLevel,
		Confidence: recommendation.Confidence,
		EvidenceRefs: append([]string(nil), recommendation.EvidenceRefs...),
		Reversibility: recommendation.Reversibility,
		RequiredApproval: authorization.RequiredApproval,
		Expiry: expiry,
		IdempotencyKey: recommendation.RequestID,
	}, nil
}

func (s SafetyBoundary) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}
