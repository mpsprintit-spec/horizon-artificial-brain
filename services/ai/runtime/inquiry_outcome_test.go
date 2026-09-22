package runtime

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestProcessInquiryOutcomeUsesRegisteredActionBinding(t *testing.T) {
	now := time.Date(2026, 9, 22, 13, 0, 0, 0, time.UTC)
	brain := knowledge.NewBrain()
	runtime := NewBrainRuntime(brain)
	orch := NewCognitiveOrchestrator(runtime)

	node, _, err := brain.Registry.GetOrCreate("cup")
	if _, err := runtime.CognitiveProcess(Event{Stimulus: []string{"cup"}, Cycles: 1, Timestamp: now}); err != nil {
		t.Fatal(err)
	}
	if err := runtime.RegisterActionBinding(ActionBinding{
		RequestID: "inquiry-12-focus",
		BrainIdentity: BrainIdentity,
		Intent: "inquiry:focus",
		TargetNodeIDs: []knowledge.NodeID{node.ID},
	}); err != nil {
		t.Fatal(err)
	}

	execution, err := CaptureExecutionForTest(runtime, InquiryExecution{
		Proposal: InquiryProposal{
			BrainIdentity: BrainIdentity,
			Sequence: 12,
			CandidateID: "focus",
			Action: InquiryFocus,
		},
		Request: ExecutionRequest{
			RequestID: "inquiry-12-focus",
			BrainIdentity: BrainIdentity,
			Expiry: now.Add(time.Minute),
			authorized: true,
		},
	}, now)
	if err != nil {
		t.Fatal(err)
	}

	result := InquiryResult{
		Execution: execution,
		Event: Event{ID: "focus-outcome", Timestamp: now},
		Observation: ObservationInput{
			Source: "camera",
			Modality: "vision",
			Tokens: []string{"cup"},
		},
		Success: true,
		Reliability: 0.9,
	}

	interpretation, learned, sequence, err := orch.ProcessInquiryOutcome(result, now)
	if err != nil {
		t.Fatal(err)
	}
	if learned {
		t.Fatal("outcome should not imply a separate Experience")
	}
	if sequence == 0 {
		t.Fatal("outcome did not advance runtime sequence")
	}
	if len(interpretation.GroundedRepresentations) != 1 {
		t.Fatalf("expected one grounded representation, got %d", len(interpretation.GroundedRepresentations))
	}
}

func TestProcessInquiryOutcomeRejectsUnboundExecutionBeforeObservation(t *testing.T) {
	now := time.Date(2026, 9, 22, 13, 0, 0, 0, time.UTC)
	runtime := NewBrainRuntime(knowledge.NewBrain())
	orch := NewCognitiveOrchestrator(runtime)

	result := InquiryResult{
		Execution: InquiryExecution{
			Proposal: InquiryProposal{
				BrainIdentity: BrainIdentity,
				Sequence: 13,
				CandidateID: "focus",
				Action: InquiryFocus,
			},
			Request: ExecutionRequest{
				RequestID: "inquiry-13-focus",
				BrainIdentity: BrainIdentity,
				Expiry: now.Add(time.Minute),
				authorized: true,
			},
		},
		Observation: ObservationInput{
			Source: "camera",
			Modality: "vision",
			Tokens: []string{"unbound"},
		},
	}
	_, _, _, err := orch.ProcessInquiryOutcome(result, now)
	if err == nil {
		t.Fatal("expected unbound inquiry outcome to be rejected")
	}
	if runtime.brain.Registry.Get("unbound") != nil {
		t.Fatal("unbound outcome injected observation into Brain")
	}
}


func CaptureExecutionForTest(r *BrainRuntime, execution InquiryExecution, now time.Time) (InquiryExecution, error) {
	return r.CaptureInquiryPrediction(execution, now)
}
