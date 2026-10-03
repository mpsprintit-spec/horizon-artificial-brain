package runtime

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/activation"
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
	if value < 0 || value > 1 {
		t.Fatalf("expected bounded model-derived information value, got %v", value)
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

func TestPlanInquiryUsesModeledInformationValue(t *testing.T) {
	now := time.Date(2026, 9, 30, 12, 5, 0, 0, time.UTC)
	brain := knowledge.NewBrain()
	rt := NewBrainRuntime(brain)

	target := brain.Store("target")
	outcomeA := brain.Store("outcome-a")
	outcomeB := brain.Store("outcome-b")

	if err := rt.RegisterActionBinding(ActionBinding{
		RequestID: "inquiry-2-focus",
		BrainIdentity: BrainIdentity,
		Intent: "inquiry:focus",
		TargetNodeIDs: []knowledge.NodeID{target.ID},
	}); err != nil {
		t.Fatal(err)
	}

	for _, outcome := range []knowledge.NodeID{outcomeA.ID, outcomeB.ID} {
		if _, err := rt.RecordInquiryConsequenceEvent(InquiryConsequenceEvent{
			Action: InquiryFocus,
			Valence: 0.8,
			InformationGain: 0.6,
			PredictionError: 0.2,
			Reliability: 0.9,
			TargetNodeIDs: []knowledge.NodeID{target.ID},
			OutcomeNodeIDs: []knowledge.NodeID{outcome},
			CausalLink: "inquiry:branch",
		}, now); err != nil {
			t.Fatal(err)
		}
	}

	agenda, err := rt.PlanInquiry(0.8, now)
	if err != nil {
		t.Fatal(err)
	}

	var focusValue float64
	var focusModeled bool
	for _, evaluation := range agenda.Evaluations {
		if evaluation.Action == InquiryFocus {
			focusValue = evaluation.InformationValue
			focusModeled = true
			break
		}
	}
	if !focusModeled {
		t.Fatal("expected focus candidate in inquiry agenda")
	}
	if focusValue <= 0 {
		t.Fatalf("expected PlanInquiry to use learned non-deterministic outcome value, got %v", focusValue)
	}
	if focusValue > 1 {
		t.Fatalf("expected bounded modeled information value, got %v", focusValue)
	}
}



func TestInquiryInformationValueIgnoresUnrelatedRecurrentActivity(t *testing.T) {
	now := time.Date(2026, 9, 30, 12, 10, 0, 0, time.UTC)
	brain := knowledge.NewBrain()
	rt := NewBrainRuntime(brain)

	target := brain.Store("target")
	outcomeA := brain.Store("outcome-a")
	outcomeB := brain.Store("outcome-b")
	unrelated := brain.Store("unrelated")

	if err := rt.RegisterActionBinding(ActionBinding{
		RequestID: "inquiry-3-focus",
		BrainIdentity: BrainIdentity,
		Intent: "inquiry:focus",
		TargetNodeIDs: []knowledge.NodeID{target.ID},
	}); err != nil {
		t.Fatal(err)
	}

	for _, outcome := range []knowledge.NodeID{outcomeA.ID, outcomeB.ID} {
		if _, err := rt.RecordInquiryConsequenceEvent(InquiryConsequenceEvent{
			Action: InquiryFocus,
			Valence: 0.8,
			InformationGain: 0.6,
			PredictionError: 0.2,
			Reliability: 0.9,
			TargetNodeIDs: []knowledge.NodeID{target.ID},
			OutcomeNodeIDs: []knowledge.NodeID{outcome},
			CausalLink: "inquiry:unrelated-activity",
		}, now); err != nil {
			t.Fatal(err)
		}
	}

	before, modeled, err := rt.PredictInquiryInformationValue(InquiryFocus, 0.8, now)
	if err != nil || !modeled {
		t.Fatalf("expected modeled inquiry value before unrelated activity, value=%v modeled=%v err=%v", before, modeled, err)
	}

	rt.activation.ActivateWith(activation.Request{
		StimulusNodeIDs: []knowledge.NodeID{unrelated.ID},
		Cycles:          1,
		Now:             now,
	})

	after, modeled, err := rt.PredictInquiryInformationValue(InquiryFocus, 0.8, now)
	if err != nil || !modeled {
		t.Fatalf("expected modeled inquiry value after unrelated activity, value=%v modeled=%v err=%v", after, modeled, err)
	}
	if after != before {
		t.Fatalf("unrelated recurrent activity changed action information value: before=%v after=%v", before, after)
	}
}


func TestInquiryInformationValueLearnsObservedInformationYield(t *testing.T) {
	now := time.Date(2026, 9, 30, 12, 15, 0, 0, time.UTC)

	build := func(gain float64) float64 {
		brain := knowledge.NewBrain()
		rt := NewBrainRuntime(brain)
		target := brain.Store("target")
		outcomeA := brain.Store("outcome-a")
		outcomeB := brain.Store("outcome-b")

		if err := rt.RegisterActionBinding(ActionBinding{
			RequestID: "inquiry-yield-focus",
			BrainIdentity: BrainIdentity,
			Intent: "inquiry:focus",
			TargetNodeIDs: []knowledge.NodeID{target.ID},
		}); err != nil {
			t.Fatal(err)
		}
		for _, outcome := range []knowledge.NodeID{outcomeA.ID, outcomeB.ID} {
			if _, err := rt.RecordInquiryConsequenceEvent(InquiryConsequenceEvent{
				Action: InquiryFocus,
				InformationGain: gain,
				Reliability: 1,
				TargetNodeIDs: []knowledge.NodeID{target.ID},
				OutcomeNodeIDs: []knowledge.NodeID{outcome},
				CausalLink: "inquiry:yield",
			}, now); err != nil {
				t.Fatal(err)
			}
		}
		value, modeled, err := rt.PredictInquiryInformationValue(InquiryFocus, 0.8, now)
		if err != nil || !modeled {
			t.Fatalf("expected modeled value, value=%v modeled=%v err=%v", value, modeled, err)
		}
		return value
	}

	low := build(0.1)
	high := build(0.9)
	if high <= low {
		t.Fatalf("learned information yield did not affect prediction: low=%v high=%v", low, high)
	}
}

func TestPlanInquiryChangesWithLearnedInformationYield(t *testing.T) {
	now := time.Date(2026, 9, 30, 12, 20, 0, 0, time.UTC)
	build := func(gain float64) InquiryAction {
		brain := knowledge.NewBrain()
		rt := NewBrainRuntime(brain)
		target := brain.Store("target")
		a := brain.Store("outcome-a")
		b := brain.Store("outcome-b")
		if err := rt.RegisterActionBinding(ActionBinding{
			RequestID: "inquiry-plan-yield",
			BrainIdentity: BrainIdentity,
			Intent: "inquiry:focus",
			TargetNodeIDs: []knowledge.NodeID{target.ID},
		}); err != nil { t.Fatal(err) }
		for _, outcome := range []knowledge.NodeID{a.ID, b.ID} {
			if _, err := rt.RecordInquiryConsequenceEvent(InquiryConsequenceEvent{
				Action: InquiryFocus, InformationGain: gain, Reliability: 1,
				TargetNodeIDs: []knowledge.NodeID{target.ID},
				OutcomeNodeIDs: []knowledge.NodeID{outcome},
				CausalLink: "inquiry:plan-yield",
			}, now); err != nil { t.Fatal(err) }
		}
		agenda, err := rt.PlanInquiry(0.8, now)
		if err != nil { t.Fatal(err) }
		if agenda.Selected == nil { t.Fatal("expected selected inquiry action") }
		return agenda.Selected.Action
	}
	low := build(0.05)
	high := build(0.95)
	if low == high {
		t.Fatalf("learned information yield did not affect selected inquiry action: low=%q high=%q", low, high)
	}
}
