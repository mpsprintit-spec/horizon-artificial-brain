package learning

import (
	"testing"
	"time"
)

func TestInquiryCandidateValidationRequiresIndependentSources(t *testing.T) {
	now := time.Date(2026, 9, 19, 14, 0, 0, 0, time.UTC)
	candidate, err := BuildInquiryCandidate(
		[]string{"air", "oxygen"},
		[]InquirySource{
			{Source: "source-a", Confidence: 0.90, Reputation: 0.90},
			{Source: "source-a", Confidence: 0.95, Reputation: 0.95},
		},
		"inquiry",
		now,
	)
	if err != nil {
		t.Fatalf("BuildInquiryCandidate: %v", err)
	}
	if len(candidate.Sources) != 1 {
		t.Fatalf("sources = %d, want duplicate source collapsed to 1", len(candidate.Sources))
	}
	if got := ValidateInquiry(candidate, DefaultLearningPolicy()); got != PromotionCandidate {
		t.Fatalf("decision = %q, want candidate with one independent source", got)
	}

	candidate, err = BuildInquiryCandidate(
		[]string{"air", "oxygen"},
		[]InquirySource{
			{Source: "source-a", Confidence: 0.90, Reputation: 0.90},
			{Source: "source-b", Confidence: 0.90, Reputation: 0.90},
		},
		"inquiry",
		now,
	)
	if err != nil {
		t.Fatalf("BuildInquiryCandidate with independent sources: %v", err)
	}
	if got := ValidateInquiry(candidate, DefaultLearningPolicy()); got != PromotionAccepted {
		t.Fatalf("decision = %q, want accepted with two independent sources", got)
	}
	if !candidate.Experience.Timestamp.Equal(now) {
		t.Fatalf("candidate timestamp = %v, want %v", candidate.Experience.Timestamp, now)
	}
}

func TestInquiryCandidateNeverMutatesBrainByConstruction(t *testing.T) {
	before := 0
	candidate, err := BuildInquiryCandidate(
		[]string{"unknown-concept"},
		[]InquirySource{{Source: "source-a", Confidence: 0.95, Reputation: 0.95}, {Source: "source-b", Confidence: 0.95, Reputation: 0.95}},
		"inquiry",
		time.Date(2026, 9, 19, 14, 1, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("BuildInquiryCandidate: %v", err)
	}
	if got := len(candidate.Experience.Sequence); got != 1 {
		t.Fatalf("candidate sequence length = %d, want 1", got)
	}
	if before != 0 {
		t.Fatal("candidate construction unexpectedly mutated external state")
	}
}
