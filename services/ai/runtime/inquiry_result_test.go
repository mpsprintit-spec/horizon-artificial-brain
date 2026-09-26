package runtime

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/activation"
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

	// The learned causal trace should affect a prediction when its action
	// target is presented again as a cue. The outcome itself is the result
	// of the previous inquiry, so the immediately stored prediction belongs
	// to that completed observation and is not retroactively recomputed.
	_, err = runtime.CognitiveProcess(Event{
		ID: "post-outcome-cue",
		Stimulus: []string{"target"},
		Cycles: 1,
		Timestamp: now.Add(2 * time.Second),
	})
	if err != nil {
		t.Fatal(err)
	}
	predictionAfter := runtime.activation.PredictionSnapshot()
	if predictionAfter.State[outcome.ID] <= 0 {
		t.Fatal("learned causal outcome did not influence prediction from the repeated action cue")
	}
}


func TestInquiryOutcomeAdaptsTemporalCausalPredictionAfterContradiction(t *testing.T) {
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	brain := knowledge.NewBrain()
	target := brain.Store("target")
	runtime := NewBrainRuntime(brain)
	orch := NewCognitiveOrchestrator(runtime)

	register := func(requestID string, sequence uint64) {
		t.Helper()
		if err := runtime.RegisterActionBinding(ActionBinding{
			RequestID: requestID,
			BrainIdentity: BrainIdentity,
			Intent: "inquiry:focus",
			TargetNodeIDs: []knowledge.NodeID{target.ID},
		}); err != nil {
			t.Fatal(err)
		}
		_ = sequence
	}
	register("inquiry-20-focus", 20)

	execution := func(requestID string, sequence uint64, prediction activation.Prediction, at time.Time) InquiryExecution {
		return InquiryExecution{
			Proposal: InquiryProposal{
				BrainIdentity: BrainIdentity,
				Sequence: sequence,
				CandidateID: "focus",
				Action: InquiryFocus,
				Reversibility: true,
			},
			Request: ExecutionRequest{
				RequestID: requestID,
				BrainIdentity: BrainIdentity,
				Expiry: at.Add(time.Minute),
				authorized: true,
			},
			Prediction: prediction,
			PredictionCapturedAt: at,
		}
	}

	output, err := runtime.CognitiveProcess(Event{
		ID: "cue-20",
		Stimulus: []string{"target"},
		Cycles: 1,
		Timestamp: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	firstExecution := execution("inquiry-20-focus", 20, output.Prediction, now)
	if _, _, _, err = orch.ProcessInquiryOutcome(InquiryResult{
		Execution: firstExecution,
		Event: Event{ID: "outcome-b", Timestamp: now.Add(time.Second)},
		Observation: ObservationInput{Source: "camera", Modality: "vision", Tokens: []string{"b"}},
		Success: true, Reliability: 1, InformationGain: 1,
	}, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	b := brain.Fetch("b")
	if b == nil {
		t.Fatal("first outcome B was not grounded")
	}

	_, err = runtime.CognitiveProcess(Event{
		ID: "cue-21",
		Stimulus: []string{"target"},
		Cycles: 1,
		Timestamp: now.Add(2 * time.Second),
	})
	if err != nil {
		t.Fatal(err)
	}
	predictionB := runtime.activation.PredictionSnapshot()
	if predictionB.State[b.ID] <= 0 {
		t.Fatal("expected first learned outcome B to become a prediction")
	}
	bBefore := brain.Patterns.ResultsFor(b.ID)
	if len(bBefore) != 1 {
		t.Fatalf("expected one A→B trace, got %d", len(bBefore))
	}
	weightBefore := bBefore[0].Weight

	register("inquiry-21-focus", 21)
	secondExecution := execution("inquiry-21-focus", 21, predictionB, now.Add(2*time.Second))
	if _, _, _, err = orch.ProcessInquiryOutcome(InquiryResult{
		Execution: secondExecution,
		Event: Event{ID: "outcome-c", Timestamp: now.Add(3 * time.Second)},
		Observation: ObservationInput{Source: "camera", Modality: "vision", Tokens: []string{"c"}},
		Success: true, Reliability: 1, InformationGain: 1,
	}, now.Add(3*time.Second)); err != nil {
		t.Fatal(err)
	}

	cNode := brain.Fetch("c")
	if cNode == nil {
		t.Fatal("contradictory outcome C was not grounded")
	}
	bAfter := brain.Patterns.ResultsFor(b.ID)
	if len(bAfter) != 1 {
		t.Fatalf("expected A→B history to remain after contradiction, got %d", len(bAfter))
	}
	if bAfter[0].Weight >= weightBefore {
		t.Fatalf("expected contradictory outcome to weaken A→B: before=%v after=%v", weightBefore, bAfter[0].Weight)
	}

	_, err = runtime.CognitiveProcess(Event{
		ID: "cue-22",
		Stimulus: []string{"target"},
		Cycles: 1,
		Timestamp: now.Add(4 * time.Second),
	})
	if err != nil {
		t.Fatal(err)
	}
	finalPrediction := runtime.activation.PredictionSnapshot()
	if finalPrediction.State[cNode.ID] <= 0 {
		t.Fatal("contradictory outcome C did not enter future prediction")
	}
	if finalPrediction.State[b.ID] <= 0 {
		t.Fatal("historical outcome B disappeared instead of remaining as a weaker alternative")
	}
}


func TestInquiryOutcomeUsesCapturedPredictionAfterInterveningCognitiveState(t *testing.T) {
	now := time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)
	brain := knowledge.NewBrain()
	target := brain.Store("target")
	intervening := brain.Store("intervening")
	runtime := NewBrainRuntime(brain)
	orch := NewCognitiveOrchestrator(runtime)
	requestID := "inquiry-explicit-prediction"
	if err := runtime.RegisterActionBinding(ActionBinding{RequestID: requestID, BrainIdentity: BrainIdentity, Intent: "inquiry:focus", TargetNodeIDs: []knowledge.NodeID{target.ID}}); err != nil { t.Fatal(err) }
	preAction, err := runtime.CognitiveProcess(Event{ID: "pre-action-explicit", Stimulus: []string{"target"}, Cycles: 1, Timestamp: now})
	if err != nil { t.Fatal(err) }
	captured := preAction.Prediction
	if len(captured.State) == 0 { t.Fatal("expected captured pre-action prediction") }
	if _, err := runtime.CognitiveProcess(Event{ID: "intervening-state", Stimulus: []string{"intervening"}, Cycles: 1, Timestamp: now.Add(500 * time.Millisecond)}); err != nil { t.Fatal(err) }
	execution := InquiryExecution{Proposal: InquiryProposal{BrainIdentity: BrainIdentity, Sequence: 30, CandidateID: "focus", Action: InquiryFocus, Reversibility: true}, Request: ExecutionRequest{RequestID: requestID, BrainIdentity: BrainIdentity, Expiry: now.Add(time.Minute), authorized: true}, Prediction: captured, PredictionCapturedAt: now}
	interpretation, _, _, err := orch.ProcessInquiryOutcome(InquiryResult{Execution: execution, Event: Event{ID: "explicit-prediction-outcome", Timestamp: now.Add(time.Second)}, Observation: ObservationInput{Source: "camera", Modality: "vision", Tokens: []string{"unexpected"}}, Success: true, Reliability: 1, InformationGain: 1}, now.Add(time.Second))
	if err != nil { t.Fatal(err) }
	expected := runtime.activation.PredictionError(captured, interpretation.State.Activations)
	if interpretation.State.PredictionError != expected { t.Fatalf("outcome did not use captured prediction: got=%v want=%v", interpretation.State.PredictionError, expected) }
	if interpretation.State.PredictionError <= 0 { t.Fatal("expected non-zero prediction error from unexpected outcome") }
}
