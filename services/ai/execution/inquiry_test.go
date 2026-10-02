package execution

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

type testExperientialPlugin struct{}

func (testExperientialPlugin) Name() string { return "test-experience" }
func (testExperientialPlugin) Trigger(string) {}

func (testExperientialPlugin) Predict(string) []knowledge.InquiryCandidate {
	return []knowledge.InquiryCandidate{{
		ActionID: "observe",
		PredictedStates: []map[knowledge.NodeID]float64{
			{1: 0.9, 2: 0.1},
			{1: 0.8, 2: 0.2},
		},
		OutcomeProbabilities: []float64{0.5, 0.5},
		Novelty: 0.8,
		Repeatability: 0.0,
		Sequence: 1,
	}}
}

func (testExperientialPlugin) Execute(string) knowledge.InquiryOutcome {
	return knowledge.InquiryOutcome{
		State: map[knowledge.NodeID]float64{1: 0.1, 2: 0.9},
		At: time.Unix(100, 0).UTC(),
	}
}

func TestExecuteInquiryClosesInquiryTrajectory(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	kb.BrainState.InquiryState.Uncertainty = 0.9

	result, ok := ExecuteInquiry(kb, testExperientialPlugin{}, "context", time.Unix(50, 0).UTC())
	if !ok {
		t.Fatal("expected inquiry execution")
	}
	if result.Candidate.ActionID != "observe" {
		t.Fatalf("unexpected selected action: %q", result.Candidate.ActionID)
	}
	if kb.BrainState.InquiryState.Pending {
		t.Fatal("inquiry should be closed after outcome")
	}
	if kb.BrainState.InquiryState.OutcomePredictionError <= 0 {
		t.Fatal("expected prediction error after mismatched outcome")
	}
}
