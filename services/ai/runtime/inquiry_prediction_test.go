package runtime

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestInquiryPredictionUsesLearnedActionTargetsWithoutExecution(t *testing.T) {
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	brain := knowledge.NewBrain()
	rt := NewBrainRuntime(brain)

	target := brain.Store("target")
	outcome := brain.Store("outcome")

	if err := rt.RegisterActionBinding(ActionBinding{
		RequestID: "inquiry-1-focus",
		BrainIdentity: BrainIdentity,
		Intent: "inquiry:focus",
		TargetNodeIDs: []knowledge.NodeID{target.ID},
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := rt.RecordInquiryConsequenceEvent(InquiryConsequenceEvent{
		Action: InquiryFocus,
		Valence: 0.8,
		InformationGain: 0.7,
		PredictionError: 0.2,
		Reliability: 0.9,
		TargetNodeIDs: []knowledge.NodeID{target.ID},
		OutcomeNodeIDs: []knowledge.NodeID{outcome.ID},
		CausalLink: "inquiry:seed",
	}, now); err != nil {
		t.Fatal(err)
	}

	before := cloneNodeValues(rt.activation.CurrentState())
	value, modeled, err := rt.PredictInquiryInformationValue(InquiryFocus, 0.8, now)
	if err != nil {
		t.Fatal(err)
	}
	if !modeled {
		t.Fatal("expected learned action model")
	}

	prediction := rt.activation.PredictOutcomeFromNodes([]knowledge.NodeID{target.ID}, now, 1)
	if prediction.State[outcome.ID] <= 0 {
		t.Fatalf("learned causal outcome was not present in hypothetical prediction: %#v", prediction.State)
	}
	if value <= 0 {
		t.Fatalf("expected positive model-derived information value, got %v", value)
	}
	after := cloneNodeValues(rt.activation.CurrentState())
	if len(before) != len(after) {
		t.Fatalf("prediction changed activation state size: got %d want %d", len(after), len(before))
	}
	for id, want := range before {
		if got := after[id]; got != want {
			t.Fatalf("prediction mutated activation node %d: got %v want %v", id, got, want)
		}
	}
}
