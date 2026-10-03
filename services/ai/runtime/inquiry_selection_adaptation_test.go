package runtime

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// TestInquirySelectionAdaptsToLaterExperience verifies that inquiry policy is
// developmental: a later stream of observed outcomes can change which action
// is selected without changing the action definitions themselves.
func TestInquirySelectionAdaptsToLaterExperience(t *testing.T) {
	now := time.Date(2026, 10, 3, 17, 30, 0, 0, time.UTC)
	brain := knowledge.NewBrain()
	rt := NewBrainRuntime(brain)

	focusTarget := brain.Store("focus-target")
	focusA := brain.Store("focus-a")
	focusB := brain.Store("focus-b")
	viewTarget := brain.Store("view-target")
	viewA := brain.Store("view-a")
	viewB := brain.Store("view-b")

	register := func(requestID string, action InquiryAction, target knowledge.NodeID) {
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
	register("adaptive-focus", InquiryFocus, focusTarget.ID)
	register("adaptive-view", InquiryChangeView, viewTarget.ID)

	learn := func(action InquiryAction, target knowledge.NodeID, outcomes []knowledge.NodeID, gain float64) {
		for _, outcome := range outcomes {
			if _, err := rt.RecordInquiryConsequenceEvent(InquiryConsequenceEvent{
				Action: action,
				InformationGain: gain,
				Reliability: 1,
				TargetNodeIDs: []knowledge.NodeID{target},
				OutcomeNodeIDs: []knowledge.NodeID{outcome},
				CausalLink: "inquiry:adaptive-selection",
			}, now); err != nil {
				t.Fatal(err)
			}
		}
	}

	// Initial experience favors focus.
	learn(InquiryFocus, focusTarget.ID, []knowledge.NodeID{focusA.ID, focusB.ID}, 0.9)
	learn(InquiryChangeView, viewTarget.ID, []knowledge.NodeID{viewA.ID, viewB.ID}, 0.1)

	first, err := rt.PlanInquiry(0.8, now)
	if err != nil {
		t.Fatal(err)
	}
	if first.Selected == nil || first.Selected.Action != InquiryFocus {
		t.Fatalf("initial learned preference selected %v; want %v", selectedAction(first), InquiryFocus)
	}

	// Later experience reverses the empirical information yield.
	later := now.Add(time.Minute)
	for i := 0; i < 6; i++ {
		learn(InquiryFocus, focusTarget.ID, []knowledge.NodeID{focusA.ID, focusB.ID}, 0.05)
		learn(InquiryChangeView, viewTarget.ID, []knowledge.NodeID{viewA.ID, viewB.ID}, 0.95)
	}

	second, err := rt.PlanInquiry(0.8, later)
	if err != nil {
		t.Fatal(err)
	}
	if second.Selected == nil || second.Selected.Action != InquiryChangeView {
		t.Fatalf("later learned preference did not adapt: selected %v; want %v", selectedAction(second), InquiryChangeView)
	}
}

func selectedAction(agenda InquiryAgenda) InquiryAction {
	if agenda.Selected == nil {
		return ""
	}
	return agenda.Selected.Action
}
