package runtime

import (
	"testing"
	"time"
)

func TestBuildInquiryRequestDoesNotExecuteAction(t *testing.T) {
	rt := NewBrainRuntime(nil)
	at := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	cognitive, _, err := NewCognitiveOrchestrator(rt).ProcessObservation(
		Event{ID: "request-input", Cycles: 1, Timestamp: at},
		ObservationInput{Source: "vision-sensor", Modality: "vision", Tokens: []string{"cup"}},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if cognitive.InquiryAgenda == nil {
		t.Fatal("cognitive observation did not create an inquiry agenda")
	}
	agenda := *cognitive.InquiryAgenda
	request, err := rt.BuildInquiryRequest(agenda)
	if err != nil {
		t.Fatal(err)
	}
	if request.BrainIdentity != BrainIdentity {
		t.Fatalf("request brain identity = %q", request.BrainIdentity)
	}
	if request.RequestID == "" || request.Action == "" {
		t.Fatal("adapter request identity/action is missing")
	}
	if request.Sequence != agenda.Sequence {
		t.Fatalf("request sequence = %d, want %d", request.Sequence, agenda.Sequence)
	}
	if len(rt.actions) != 0 {
		t.Fatal("building an inquiry request must not register an executable action")
	}
	if !rt.brain.BrainState.InquiryState.Pending {
		t.Fatal("inquiry selection should remain pending until an observation returns")
	}
}

func TestProcessInquiryObservationClosesPreviousTrajectoryBeforePlanningNext(t *testing.T) {
	rt := NewBrainRuntime(nil)
	orch := NewCognitiveOrchestrator(rt)
	at := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)

	cognitiveInput, _, err := orch.ProcessObservation(
		Event{ID: "inquiry-input", Cycles: 1, Timestamp: at},
		ObservationInput{Source: "vision-sensor", Modality: "vision", Tokens: []string{"cup"}},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	initialAgenda := cognitiveInput.InquiryAgenda
	if initialAgenda == nil {
		t.Fatal("initial inquiry agenda was not planned")
	}
	request, err := orch.RequestInquiry(*initialAgenda)
	if err != nil {
		t.Fatal(err)
	}

	cognitive, _, err := orch.ProcessInquiryObservation(
		request,
		Event{ID: "inquiry-observation-1", Cycles: 1, Timestamp: at.Add(time.Second)},
		ObservationInput{Source: "vision-sensor", Modality: "vision", Tokens: []string{"cup"}},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if cognitive.InquiryAgenda == nil {
		t.Fatal("next inquiry agenda was not planned")
	}
	if cognitive.InquiryAgenda.Sequence <= initialAgenda.Sequence {
		t.Fatalf("next inquiry sequence = %d, want > %d", cognitive.InquiryAgenda.Sequence, initialAgenda.Sequence)
	}
	state := rt.brain.BrainState.InquiryState
	if !state.Pending {
		t.Fatal("next inquiry should be pending after feedback processing")
	}
	if state.ObservedInformationGain < 0 || state.ObservedInformationGain > 1 {
		t.Fatalf("observed information gain out of range: %v", state.ObservedInformationGain)
	}
	if state.OutcomePredictionError < 0 || state.OutcomePredictionError > 1 {
		t.Fatalf("outcome prediction error out of range: %v", state.OutcomePredictionError)
	}
	if state.Sequence != cognitive.InquiryAgenda.Sequence {
		t.Fatalf("persisted inquiry sequence = %d, want %d", state.Sequence, cognitive.InquiryAgenda.Sequence)
	}
}

func TestProcessInquiryObservationRejectsDifferentBrain(t *testing.T) {
	rt := NewBrainRuntime(nil)
	orch := NewCognitiveOrchestrator(rt)
	_, _, err := orch.ProcessInquiryObservation(
		InquiryRequest{BrainIdentity: "other-brain", RequestID: "inquiry-1", Action: InquiryWait},
		Event{ID: "bad-inquiry", Timestamp: time.Now().UTC()},
		ObservationInput{Source: "sensor", Modality: "vision", Tokens: []string{"x"}},
		nil,
	)
	if err == nil {
		t.Fatal("expected cross-brain inquiry request to be rejected")
	}
}
