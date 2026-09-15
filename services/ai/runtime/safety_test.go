package runtime

import (
	"strings"
	"testing"
	"time"
)

func TestGate8HighConfidenceCannotAuthorizeRiskyAction(t *testing.T) {
	now := time.Date(2026, 9, 15, 16, 0, 0, 0, time.UTC)
	boundary := SafetyBoundary{Now: func() time.Time { return now }}
	recommendation := Recommendation{
		RequestID: "risk-1", BrainIdentity: BrainIdentity, Intent: "high-impact action",
		Confidence: 0.999999, RiskLevel: RiskHigh, CreatedAt: now,
	}
	assessment, err := boundary.Assess(recommendation)
	if err != nil { t.Fatalf("assess: %v", err) }
	decision, err := boundary.Authorize(recommendation, assessment, "")
	if err != nil { t.Fatalf("authorize: %v", err) }
	if decision.Authorized { t.Fatal("high confidence bypassed required human authorization") }
	if !decision.RequiredApproval { t.Fatal("high-risk action did not require approval") }
}

func TestGate8AuthorizationRequiresMatchingRequestIdentity(t *testing.T) {
	boundary := SafetyBoundary{Now: func() time.Time { return time.Date(2026, 9, 15, 16, 0, 0, 0, time.UTC) }}
	recommendation := Recommendation{RequestID: "r1", BrainIdentity: BrainIdentity, Intent: "action", RiskLevel: RiskHigh}
	assessment, err := boundary.Assess(recommendation)
	if err != nil { t.Fatalf("assess: %v", err) }
	assessment.RequestID = "r2"
	if _, err := boundary.Authorize(recommendation, assessment, "operator"); err == nil || !strings.Contains(err.Error(), "request IDs") {
		t.Fatalf("expected identity mismatch, got %v", err)
	}
}

func TestGate8UnauthorizedCannotBecomeExecutionRequest(t *testing.T) {
	now := time.Date(2026, 9, 15, 17, 0, 0, 0, time.UTC)
	boundary := SafetyBoundary{Now: func() time.Time { return now }}
	recommendation := Recommendation{RequestID: "r3", BrainIdentity: BrainIdentity, Intent: "action", RiskLevel: RiskHigh, Confidence: 1}
	assessment, err := boundary.Assess(recommendation)
	if err != nil { t.Fatalf("assess: %v", err) }
	decision, err := boundary.Authorize(recommendation, assessment, "")
	if err != nil { t.Fatalf("authorize: %v", err) }
	if _, err := boundary.BuildExecutionRequest(recommendation, assessment, decision, now.Add(time.Minute)); err == nil || !strings.Contains(err.Error(), "not authorized") {
		t.Fatalf("expected authorization rejection, got %v", err)
	}
}

func TestGate8AuthorizedExecutionRequestCarriesPolicyBoundary(t *testing.T) {
	now := time.Date(2026, 9, 15, 18, 0, 0, 0, time.UTC)
	boundary := SafetyBoundary{Now: func() time.Time { return now }}
	recommendation := Recommendation{
		RequestID: "r4", BrainIdentity: BrainIdentity, Intent: "reversible action",
		Confidence: 0.2, EvidenceRefs: []string{"e1"}, Reversibility: true, RiskLevel: RiskMedium,
	}
	assessment, err := boundary.Assess(recommendation)
	if err != nil { t.Fatalf("assess: %v", err) }
	decision, err := boundary.Authorize(recommendation, assessment, "operator-1")
	if err != nil { t.Fatalf("authorize: %v", err) }
	request, err := boundary.BuildExecutionRequest(recommendation, assessment, decision, now.Add(time.Minute))
	if err != nil { t.Fatalf("build execution request: %v", err) }
	if request.IdempotencyKey != recommendation.RequestID { t.Fatalf("idempotency key: got %q want %q", request.IdempotencyKey, recommendation.RequestID) }
	if !request.RequiredApproval { t.Fatal("medium-risk execution request lost approval requirement") }
	if request.BrainIdentity != BrainIdentity { t.Fatal("execution request lost brain identity") }
}

func TestGate8ExpiredExecutionRequestRejected(t *testing.T) {
	now := time.Date(2026, 9, 15, 19, 0, 0, 0, time.UTC)
	boundary := SafetyBoundary{Now: func() time.Time { return now }}
	recommendation := Recommendation{RequestID: "r5", BrainIdentity: BrainIdentity, Intent: "action", RiskLevel: RiskLow}
	assessment, err := boundary.Assess(recommendation)
	if err != nil { t.Fatalf("assess: %v", err) }
	decision, err := boundary.Authorize(recommendation, assessment, "")
	if err != nil { t.Fatalf("authorize: %v", err) }
	if _, err := boundary.BuildExecutionRequest(recommendation, assessment, decision, now); err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("expected expiry rejection, got %v", err)
	}
}
