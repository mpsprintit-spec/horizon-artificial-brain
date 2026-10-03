package runtime

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// TestInquiryInformationValueTracksOutcomeDifferentiation verifies that the
// structural part of information value comes from the learned action->outcome
// distribution: differentiated outcomes have greater entropy than a single
// predictable outcome, while the empirical action yield remains learned.
func TestInquiryInformationValueTracksOutcomeDifferentiation(t *testing.T) {
	now := time.Date(2026, 10, 3, 18, 0, 0, 0, time.UTC)

	build := func(twoOutcomes bool) float64 {
		brain := knowledge.NewBrain()
		rt := NewBrainRuntime(brain)

		target := brain.Store("target")
		outcomeA := brain.Store("outcome-a")
		outcomeB := brain.Store("outcome-b")

		if err := rt.RegisterActionBinding(ActionBinding{
			RequestID: "entropy-test",
			BrainIdentity: BrainIdentity,
			Intent: "inquiry:focus",
			TargetNodeIDs: []knowledge.NodeID{target.ID},
		}); err != nil {
			t.Fatal(err)
		}

		// Establish empirical information yield separately from the structural
		// outcome distribution being tested.
		if _, err := rt.RecordInquiryConsequenceEvent(InquiryConsequenceEvent{
			Action: InquiryFocus,
			InformationGain: 1,
			Reliability: 1,
			TargetNodeIDs: []knowledge.NodeID{target.ID},
			OutcomeNodeIDs: []knowledge.NodeID{outcomeA.ID},
			CausalLink: "entropy-test",
		}, now); err != nil {
			t.Fatal(err)
		}

		if twoOutcomes {
			if _, err := rt.RecordInquiryConsequenceEvent(InquiryConsequenceEvent{
				Action: InquiryFocus,
				InformationGain: 1,
				Reliability: 1,
				TargetNodeIDs: []knowledge.NodeID{target.ID},
				OutcomeNodeIDs: []knowledge.NodeID{outcomeB.ID},
				CausalLink: "entropy-test",
			}, now.Add(time.Nanosecond)); err != nil {
				t.Fatal(err)
			}
		}

		value, modeled, err := rt.PredictInquiryInformationValue(InquiryFocus, 0.8, now)
		if err != nil {
			t.Fatal(err)
		}
		if !modeled {
			t.Fatal("expected learned inquiry model")
		}
		return value
	}

	deterministic := build(false)
	differentiated := build(true)

	if deterministic != 0 {
		t.Fatalf("predictable single-outcome inquiry should have zero outcome entropy, got %v", deterministic)
	}
	if differentiated <= deterministic {
		t.Fatalf("differentiated outcomes did not increase information value: deterministic=%v differentiated=%v", deterministic, differentiated)
	}
}
