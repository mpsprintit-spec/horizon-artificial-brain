package runtime

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// TestInquirySelectionEmergesFromLearnedInformationValue verifies the complete
// policy boundary: two actions with different learned information yields can
// produce different selections without assigning a semantic preference to
// either action. All non-information experience is held equal by using neutral
// consequence valence and identical causal structure.
func TestInquirySelectionEmergesFromLearnedInformationValue(t *testing.T) {
	now := time.Date(2026, 10, 3, 17, 0, 0, 0, time.UTC)

	build := func(focusGain, changeViewGain float64) InquiryAction {
		brain := knowledge.NewBrain()
		rt := NewBrainRuntime(brain)

		focusTarget := brain.Store("focus-target")
		focusA := brain.Store("focus-a")
		focusB := brain.Store("focus-b")
		viewTarget := brain.Store("view-target")
		viewA := brain.Store("view-a")
		viewB := brain.Store("view-b")

		bind := func(requestID string, action InquiryAction, target knowledge.NodeID) {
			t.Helper()
			if err := rt.RegisterActionBinding(ActionBinding{
				RequestID: requestID,
				BrainIdentity: BrainIdentity,
				Intent: "inquiry:" + string(action),
				TargetNodeIDs: []knowledge.NodeID{target},
			}); err != nil {
				t.Fatal(err)
			}
		}
		bind("selection-focus", InquiryFocus, focusTarget.ID)
		bind("selection-view", InquiryChangeView, viewTarget.ID)

		learn := func(action InquiryAction, target knowledge.NodeID, outcomes []knowledge.NodeID, gain float64) {
			for _, outcome := range outcomes {
				if _, err := rt.RecordInquiryConsequenceEvent(InquiryConsequenceEvent{
					Action: action,
					InformationGain: gain,
					Reliability: 1,
					TargetNodeIDs: []knowledge.NodeID{target},
					OutcomeNodeIDs: []knowledge.NodeID{outcome},
					CausalLink: "inquiry:selection",
				}, now); err != nil {
					t.Fatal(err)
				}
			}
		}
		learn(InquiryFocus, focusTarget.ID, []knowledge.NodeID{focusA.ID, focusB.ID}, focusGain)
		learn(InquiryChangeView, viewTarget.ID, []knowledge.NodeID{viewA.ID, viewB.ID}, changeViewGain)

		agenda, err := rt.PlanInquiry(0.8, now)
		if err != nil {
			t.Fatal(err)
		}
		if agenda.Selected == nil {
			t.Fatal("expected an inquiry selection")
		}
		return agenda.Selected.Action
	}

	focusSelected := build(0.9, 0.1)
	if focusSelected != InquiryFocus {
		t.Fatalf("high-information focus action was not selected: got %q", focusSelected)
	}

	viewSelected := build(0.1, 0.9)
	if viewSelected != InquiryChangeView {
		t.Fatalf("high-information change-view action was not selected: got %q", viewSelected)
	}
}
