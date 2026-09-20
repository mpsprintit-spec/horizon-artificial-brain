package runtime

import (
	"testing"
	"time"
)

func TestBuildInquiryAgendaSelectsInformationSeekingCandidate(t *testing.T) {
	at := time.Date(2026, 9, 20, 5, 0, 0, 0, time.UTC)
	candidates := DefaultInquiryCandidates(.9)
	agenda, err := BuildInquiryAgenda(BrainIdentity, 7, at, candidates, DefaultInquiryPolicy())
	if err != nil {
		t.Fatalf("BuildInquiryAgenda() error = %v", err)
	}
	if agenda.Selected == nil {
		t.Fatal("agenda selected = nil")
	}
	if agenda.Selected.Action == InquiryVerbalQuestion {
		t.Fatal("verbal question became the default selection")
	}
	if agenda.Selected.Action != InquiryFocus {
		t.Fatalf("selected action = %q, want focus under the deterministic default policy", agenda.Selected.Action)
	}
	if len(agenda.Evaluations) != len(candidates) {
		t.Fatalf("evaluations = %d, want %d", len(agenda.Evaluations), len(candidates))
	}
}

func TestInquiryAgendaDoesNotAuthorizeExecution(t *testing.T) {
	agenda, err := BuildInquiryAgenda(
		BrainIdentity,
		1,
		time.Date(2026, 9, 20, 5, 0, 0, 0, time.UTC),
		[]InquiryCandidate{{
			ID: "safe",
			Action: InquirySafeManipulation,
			ExpectedInformationGain: 1,
			Cost: 0,
			Reversibility: 1,
			SocialFit: 1,
			PriorExperience: 1,
			RequiresAuthorization: true,
		}},
		DefaultInquiryPolicy(),
	)
	if err != nil {
		t.Fatalf("BuildInquiryAgenda() error = %v", err)
	}
	if agenda.Selected == nil || !agenda.Selected.RequiresAuthorization {
		t.Fatal("authorization requirement was lost")
	}
}
