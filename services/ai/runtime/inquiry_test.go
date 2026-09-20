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


func TestBuildInquiryAgendaFromCognitionUsesCurrentNeuralUncertainty(t *testing.T) {
	at := time.Date(2026, 9, 20, 5, 0, 0, 0, time.UTC)
	interpretation := CognitiveInterpretation{
		State: CognitiveState{
			BrainIdentity: BrainIdentity,
			Sequence: 42,
			Resonance: .9,
			PredictionError: .2,
		},
		Answer: Answer{
			Uncertainty: Uncertainty{
				Level: .1,
				PredictionError: .2,
				Reason: "low-uncertainty",
			},
		},
	}
	agenda, err := BuildInquiryAgendaFromCognition(interpretation, at)
	if err != nil {
		t.Fatalf("BuildInquiryAgendaFromCognition() error = %v", err)
	}
	if agenda.BrainIdentity != BrainIdentity || agenda.Sequence != 42 {
		t.Fatalf("agenda identity/sequence = %q/%d, want %q/42", agenda.BrainIdentity, agenda.Sequence, BrainIdentity)
	}
	if !agenda.CreatedAt.Equal(at) {
		t.Fatalf("agenda timestamp = %v, want %v", agenda.CreatedAt, at)
	}
	for _, evaluation := range agenda.Evaluations {
		if evaluation.InformationValue > .1+.000001 {
			t.Fatalf("candidate %q used uncertainty outside cognitive answer: %v", evaluation.Action, evaluation.InformationValue)
		}
	}
}

func TestBuildInquiryAgendaFromCognitionRejectsMissingBrainIdentity(t *testing.T) {
	_, err := BuildInquiryAgendaFromCognition(CognitiveInterpretation{
		Answer: Answer{Uncertainty: Uncertainty{Level: .8}},
	}, time.Date(2026, 9, 20, 5, 0, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("expected missing brain identity error")
	}
}
