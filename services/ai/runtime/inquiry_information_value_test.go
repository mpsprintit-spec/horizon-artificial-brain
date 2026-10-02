package runtime

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestInquiryInformationValueReflectsLearnedOutcomeDistribution(t *testing.T) {
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	brain := knowledge.NewBrain()
	rt := NewBrainRuntime(brain)

	focusTarget := brain.Store("focus-target")
	viewTarget := brain.Store("view-target")
	outcomeB := brain.Store("outcome-b")
	outcomeC := brain.Store("outcome-c")

	for _, binding := range []ActionBinding{
		{
			RequestID:     "inquiry-focus",
			BrainIdentity: BrainIdentity,
			Intent:        "inquiry:focus",
			TargetNodeIDs: []knowledge.NodeID{focusTarget.ID},
		},
		{
			RequestID:     "inquiry-change-view",
			BrainIdentity: BrainIdentity,
			Intent:        "inquiry:change_view",
			TargetNodeIDs: []knowledge.NodeID{viewTarget.ID},
		},
	} {
		if err := rt.RegisterActionBinding(binding); err != nil {
			t.Fatal(err)
		}
	}

	record := func(action InquiryAction, target, outcome knowledge.NodeID, link string) {
		t.Helper()
		if _, err := rt.RecordInquiryConsequenceEvent(InquiryConsequenceEvent{
			Action:         action,
			Valence:        0.8,
			InformationGain: 0.7,
			PredictionError: 0.2,
			Reliability:    0.9,
			TargetNodeIDs:  []knowledge.NodeID{target},
			OutcomeNodeIDs: []knowledge.NodeID{outcome},
			CausalLink:     link,
		}, now); err != nil {
			t.Fatal(err)
		}
	}

	// Focus has a skewed learned outcome distribution: B is repeated,
	// while C remains as contradictory evidence.
	record(InquiryFocus, focusTarget.ID, outcomeB.ID, "focus-b-1")
	record(InquiryFocus, focusTarget.ID, outcomeB.ID, "focus-b-2")
	record(InquiryFocus, focusTarget.ID, outcomeC.ID, "focus-c-1")

	// Change-view has a balanced learned outcome distribution.
	record(InquiryChangeView, viewTarget.ID, outcomeB.ID, "view-b-1")
	record(InquiryChangeView, viewTarget.ID, outcomeC.ID, "view-c-1")

	focusValue, focusModeled, err := rt.PredictInquiryInformationValue(InquiryFocus, 0.8, now)
	if err != nil {
		t.Fatal(err)
	}
	viewValue, viewModeled, err := rt.PredictInquiryInformationValue(InquiryChangeView, 0.8, now)
	if err != nil {
		t.Fatal(err)
	}
	if !focusModeled || !viewModeled {
		t.Fatalf("expected both learned action models: focus=%v view=%v", focusModeled, viewModeled)
	}
	if focusValue <= viewValue {
		t.Fatalf("expected the more concentrated learned outcome distribution to have greater modeled information value: focus=%v view=%v", focusValue, viewValue)
	}
}
