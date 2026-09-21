package runtime

import (
	"strings"
	"testing"
	"time"
)

func TestBuildInquiryProposalDoesNotAuthorize(t *testing.T) {
	now := time.Date(2026, 9, 21, 13, 0, 0, 0, time.UTC)
	agenda := InquiryAgenda{
		BrainIdentity: BrainIdentity,
		Sequence: 7,
		CreatedAt: now,
		Selected: &InquiryEvaluation{
			CandidateID: "focus",
			Action: InquiryFocus,
			InformationValue: .8,
			Score: .9,
		},
	}
	proposal, err := BuildInquiryProposal(agenda, "determine object boundary", now)
	if err != nil {
		t.Fatal(err)
	}
	if proposal.RequiresAuthorization {
		t.Fatal("focus should not require authorization")
	}
	if proposal.Intent != "inquiry:focus" {
		t.Fatalf("unexpected intent %q", proposal.Intent)
	}
	if proposal.Sequence != 7 || proposal.BrainIdentity != BrainIdentity {
		t.Fatal("proposal identity was not preserved")
	}
}

func TestPrepareInquiryExecutionRequiresExplicitApprovalForAuthorizedInquiry(t *testing.T) {
	now := time.Date(2026, 9, 21, 13, 0, 0, 0, time.UTC)
	agenda := InquiryAgenda{
		BrainIdentity: BrainIdentity,
		Sequence: 8,
		CreatedAt: now,
		Selected: &InquiryEvaluation{
			CandidateID: "safe-manipulation",
			Action: InquirySafeManipulation,
			InformationValue: .7,
			Score: .6,
			RequiresAuthorization: true,
		},
	}
	proposal, err := BuildInquiryProposal(agenda, "test object response", now)
	if err != nil {
		t.Fatal(err)
	}

	boundary := SafetyBoundary{Now: func() time.Time { return now }}
	expiry := now.Add(time.Minute)

	pending, decision, err := PrepareInquiryExecution(boundary, proposal, "", expiry)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Authorized {
		t.Fatal("inquiry unexpectedly authorized without approver")
	}
	if pending.Request.Authorized() {
		t.Fatal("pending inquiry contains an authorized execution request")
	}

	execution, decision, err := PrepareInquiryExecution(boundary, proposal, "human:operator", expiry)
	if err != nil {
		t.Fatal(err)
	}
	if !decision.Authorized {
		t.Fatal("explicit approver did not authorize inquiry")
	}
	if err := ValidateInquiryExecution(execution, now); err != nil {
		t.Fatal(err)
	}
	if execution.Request.RequestID != "inquiry-8-safe-manipulation" {
		t.Fatalf("unexpected request ID %q", execution.Request.RequestID)
	}
}

func TestValidateInquiryExecutionRejectsMismatchedRequest(t *testing.T) {
	now := time.Date(2026, 9, 21, 13, 0, 0, 0, time.UTC)
	execution := InquiryExecution{
		Proposal: InquiryProposal{
			BrainIdentity: BrainIdentity,
			Sequence: 9,
			CandidateID: "focus",
			Action: InquiryFocus,
		},
		Request: ExecutionRequest{
			RequestID: "inquiry-9-reobserve",
			BrainIdentity: BrainIdentity,
			Expiry: now.Add(time.Minute),
			authorized: true,
		},
	}
	err := ValidateInquiryExecution(execution, now)
	if err == nil || !strings.Contains(err.Error(), "identity mismatch") {
		t.Fatalf("expected identity mismatch, got %v", err)
	}
}
