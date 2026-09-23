package runtime

import (
	"errors"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/learning"
)

// InquiryResult is the post-execution handoff from an external/device
// executor back into Horizon's canonical cognitive pipeline. The executor
// owns the physical action; Horizon owns interpretation of the resulting
// observation.
type InquiryResult struct {
	Execution InquiryExecution
	Event Event
	Observation ObservationInput
	Experience *learning.Experience
	Success bool
	Reliability float64
	PredictionError float64
}

// ProcessInquiryResult validates the execution boundary before accepting any
// observation produced by the inquiry. The validation happens before
// grounding, learning, or cognitive processing, so an expired/mismatched
// execution cannot inject data into the Brain.
//
// The executor is expected to call this only after the external action has
// actually completed. This method does not execute the action itself.
func (o *CognitiveOrchestrator) ProcessInquiryResult(result InquiryResult, now time.Time) (CognitiveInterpretation, bool, error) {
	if o == nil || o.Runtime == nil {
		return CognitiveInterpretation{}, false, errors.New("cognitive orchestrator is not initialized")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if err := ValidateInquiryExecution(result.Execution, now); err != nil {
		return CognitiveInterpretation{}, false, err
	}
	if result.Event.ID == "" {
		result.Event.ID = result.Execution.Request.RequestID
	}
	if result.Event.Timestamp.IsZero() {
		result.Event.Timestamp = now.UTC()
	}
	result.Event.Source = result.Observation.Source
	result.Event.Modality = result.Observation.Modality
	return o.ProcessObservation(result.Event, result.Observation, result.Experience)
}

// ProcessInquiryOutcome records the verified execution outcome through the
// existing evidence/promotion subsystem and then feeds the same observation
// through the canonical cognitive pipeline. No second learning or memory
// subsystem is created here.
//
// The action binding must have been registered for the exact execution
// RequestID. Outcome learning therefore remains constrained by the explicit
// neural targets declared by the action binding.
func (o *CognitiveOrchestrator) ProcessInquiryOutcome(result InquiryResult, now time.Time) (CognitiveInterpretation, bool, uint64, error) {
	if o == nil || o.Runtime == nil {
		return CognitiveInterpretation{}, false, 0, errors.New("cognitive orchestrator is not initialized")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if err := ValidateInquiryExecution(result.Execution, now); err != nil {
		return CognitiveInterpretation{}, false, 0, err
	}
	if len(result.Execution.Prediction.State) == 0 {
		return CognitiveInterpretation{}, false, 0, errors.New("inquiry execution has no pre-action prediction snapshot")
	}
	observedAt := result.Event.Timestamp
	if observedAt.IsZero() {
		observedAt = now.UTC()
	}
	source := result.Observation.Source
	if source == "" {
		source = "inquiry-outcome"
	}
	modality := result.Observation.Modality
	if modality == "" {
		modality = "inquiry-result"
	}
	sequence, err := o.Runtime.ObserveOutcome(OutcomeEvent{
		RequestID: result.Execution.Request.RequestID,
		BrainIdentity: result.Execution.Request.BrainIdentity,
		Success: result.Success,
		Observation: append([]string(nil), result.Observation.Tokens...),
		Source: source,
		Modality: modality,
		ObservedAt: observedAt,
		Reliability: result.Reliability,
	})
	if err != nil {
		return CognitiveInterpretation{}, false, 0, err
	}
	if result.Event.ID == "" {
		result.Event.ID = result.Execution.Request.RequestID
	}
	result.Event.Timestamp = observedAt
	result.Event.Source = source
	result.Event.Modality = modality
	interpretation, learned, err := o.ProcessObservation(result.Event, result.Observation, result.Experience)
	if err != nil {
		return CognitiveInterpretation{}, learned, sequence, err
	}
	predictionError := o.Runtime.activation.PredictionError(result.Execution.Prediction, interpretation.State.Activations)
	if absFloat(predictionError-interpretation.State.PredictionError) > 1e-9 {
		return CognitiveInterpretation{}, learned, sequence, errors.New("inquiry prediction snapshot does not match cognitive prediction error")
	}
	result.PredictionError = predictionError
	consequence := AssessConsequence(ConsequenceInput{
		Success: result.Success,
		Reliability: result.Reliability,
		Reversible: result.Execution.Proposal.Reversibility,
		PredictionError: predictionError,
	})
	interpretation.Consequence = &consequence
	finalSequence, err := o.Runtime.RecordInquiryConsequenceEvent(InquiryConsequenceEvent{
		Action: result.Execution.Proposal.Action,
		Valence: consequence.Valence,
	}, observedAt)
	if err != nil {
		return CognitiveInterpretation{}, learned, sequence, err
	}
	return interpretation, learned, finalSequence, nil
}

func absFloat(v float64) float64 {
	if v < 0 { return -v }
	return v
}
