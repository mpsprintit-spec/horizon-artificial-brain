package runtime

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestProcessInquiryResultRejectsBeforeBrainMutation(t *testing.T) {
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	runtime := NewBrainRuntime(knowledge.NewBrain())
	orch := NewCognitiveOrchestrator(runtime)

	execution := InquiryExecution{
		Proposal: InquiryProposal{
			BrainIdentity: BrainIdentity,
			Sequence: 4,
			CandidateID: "focus",
			Action: InquiryFocus,
		},
		Request: ExecutionRequest{
			RequestID: "inquiry-4-reobserve",
			BrainIdentity: BrainIdentity,
			Expiry: now.Add(time.Minute),
			authorized: true,
		},
	}
	_, _, err := orch.ProcessInquiryResult(InquiryResult{
		Execution: execution,
		Event: Event{ID: "rejected"},
		Observation: ObservationInput{Source: "camera", Modality: "vision", Tokens: []string{"cup"}},
	}, now)
	if err == nil {
		t.Fatal("expected mismatched execution to be rejected")
	}
	if runtime.brain.Registry.Get("cup") != nil {
		t.Fatal("rejected inquiry injected a neural representation")
	}
}

func TestProcessInquiryResultFeedsObservationIntoCanonicalPipeline(t *testing.T) {
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	runtime := NewBrainRuntime(knowledge.NewBrain())
	orch := NewCognitiveOrchestrator(runtime)

	execution := InquiryExecution{
		Proposal: InquiryProposal{
			BrainIdentity: BrainIdentity,
			Sequence: 5,
			CandidateID: "focus",
			Action: InquiryFocus,
		},
		Request: ExecutionRequest{
			RequestID: "inquiry-5-focus",
			BrainIdentity: BrainIdentity,
			Expiry: now.Add(time.Minute),
			authorized: true,
		},
	}
	interpretation, learned, err := orch.ProcessInquiryResult(InquiryResult{
		Execution: execution,
		Event: Event{ID: "focus-result", Timestamp: now},
		Observation: ObservationInput{
			Source: "camera",
			Modality: "vision",
			Tokens: []string{"cup", "table"},
		},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if learned {
		t.Fatal("inquiry result should not imply learning without an experience")
	}
	if len(interpretation.GroundedRepresentations) != 2 {
		t.Fatalf("expected two grounded representations, got %d", len(interpretation.GroundedRepresentations))
	}
	if runtime.brain.Registry.Get("cup") == nil || runtime.brain.Registry.Get("table") == nil {
		t.Fatal("inquiry observation did not enter the canonical Brain")
	}
	if interpretation.InquiryAgenda == nil {
		t.Fatal("inquiry result did not return the next inquiry agenda")
	}
}


func TestInquiryOutcomeClosesPredictionCausalPredictionLoop(t *testing.T) {
	now := time.Date(2026, 9, 23, 13, 0, 0, 0, time.UTC)
	brain := knowledge.NewBrain()
	target := brain.Store("target")
	runtime := NewBrainRuntime(brain)
	orch := NewCognitiveOrchestrator(runtime)

	requestID := "inquiry-10-focus"
	if err := runtime.RegisterActionBinding(ActionBinding{
		RequestID: requestID,
		BrainIdentity: BrainIdentity,
		Intent: "inquiry:focus",
		TargetNodeIDs: []knowledge.NodeID{target.ID},
	}); err != nil {
		t.Fatal(err)
	}

	output, err := runtime.CognitiveProcess(Event{
		ID: "pre-action",
		Stimulus: []string{"target"},
		Cycles: 1,
		Timestamp: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	prediction := output.Prediction
	if len(prediction.State) == 0 {
		t.Fatal("expected pre-action prediction")
	}

	execution := InquiryExecution{
		Proposal: InquiryProposal{
			BrainIdentity: BrainIdentity,
			Sequence: 10,
			CandidateID: "focus",
			Action: InquiryFocus,
			Reversibility: true,
		},
		Request: ExecutionRequest{
			RequestID: requestID,
			BrainIdentity: BrainIdentity,
			Expiry: now.Add(time.Minute),
			authorized: true,
		},
		Prediction: prediction,
		PredictionCapturedAt: now,
	}
	_, _, _, err = orch.ProcessInquiryOutcome(InquiryResult{
		Execution: execution,
		Event: Event{ID: requestID, Timestamp: now.Add(time.Second)},
		Observation: ObservationInput{
			Source: "camera",
			Modality: "vision",
			Tokens: []string{"outcome"},
		},
		Success: true,
		Reliability: 1,
		InformationGain: 1,
	}, now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}

	outcome := brain.Fetch("outcome")
	if outcome == nil {
		t.Fatal("outcome was not grounded")
	}
	predictionAfter := runtime.activation.PredictionSnapshot()
	if predictionAfter.State[outcome.ID] <= 0 {
		t.Fatal("causal outcome did not influence subsequent prediction")
	}
}
