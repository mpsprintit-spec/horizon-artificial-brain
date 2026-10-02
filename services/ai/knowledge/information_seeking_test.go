package knowledge

import (
	"testing"
	"time"
)

func TestInformationSeekingPrefersUncertainNovelExperience(t *testing.T) {
	states := []map[NodeID]float64{
		{1: 0.1, 2: 0.9},
		{1: 0.9, 2: 0.1},
	}
	candidate := InquiryCandidate{
		ActionID: "explore",
		PredictedStates: states,
		OutcomeProbabilities: []float64{0.5, 0.5},
		Novelty: 0.9,
		Repeatability: 0,
	}
	score := InformationSeekingScore(candidate, 0.9, 0.4, 0.2)
	if score <= 0.2 {
		t.Fatalf("expected meaningful information value, got %f", score)
	}
}

func TestInformationSeekingPenalizesRepetition(t *testing.T) {
	candidate := InquiryCandidate{
		ActionID: "repeat",
		PredictedStates: []map[NodeID]float64{{1: 0.5}, {1: 0.5}},
		OutcomeProbabilities: []float64{0.5, 0.5},
		Novelty: 0.1,
		Repeatability: 1,
	}
	score := InformationSeekingScore(candidate, 0.8, 0.2, 0.8)
	if score != 0 {
		t.Fatalf("expected repeated predictable experience to have no selection pressure, got %f", score)
	}
}

func TestInformationSeekingSelectionIsDeterministic(t *testing.T) {
	candidates := []InquiryCandidate{
		{ActionID: "b", Sequence: 2, PredictedStates: []map[NodeID]float64{{1: 0.2}, {1: 0.8}}, OutcomeProbabilities: []float64{0.5, 0.5}, Novelty: 0.8},
		{ActionID: "a", Sequence: 1, PredictedStates: []map[NodeID]float64{{1: 0.2}, {1: 0.8}}, OutcomeProbabilities: []float64{0.5, 0.5}, Novelty: 0.8},
	}
	selected, _, ok := SelectInformationSeekingCandidate(candidates, 0.8, 0.3, 0)
	if !ok {
		t.Fatal("expected a selected candidate")
	}
	if selected.ActionID != "a" {
		t.Fatalf("expected deterministic tie-break by action id, got %q", selected.ActionID)
	}
}

func TestRecordInquirySelectionUpdatesNumericBoundary(t *testing.T) {
	kb := NewKnowledgeBase()
	candidate := InquiryCandidate{ActionID: "candidate-1", Sequence: 4}
	kb.RecordInquirySelection(candidate, 0.73, fixedTestTime())
	state := kb.BrainState.InquiryState
	if !state.Pending || state.SelectedAction != "candidate-1" {
		t.Fatalf("selection boundary not persisted: %+v", state)
	}
	if state.SelectedScore != 0.73 || state.ExpectedInformationGain != 0.73 {
		t.Fatalf("unexpected score: %+v", state)
	}
}

func fixedTestTime() time.Time {
	return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
}
