package runtime

import (
	"errors"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/learning"
)

type InquiryResult struct {
	Execution InquiryExecution
	Event Event
	Observation ObservationInput
	Experience *learning.Experience
	Success bool
	Reliability float64
	Cost float64
	Risk float64
	InformationGain float64
	PredictionError float64
}

func (o *CognitiveOrchestrator) ProcessInquiryResult(result InquiryResult, now time.Time) (CognitiveInterpretation, bool, error) {
	if o == nil || o.Runtime == nil { return CognitiveInterpretation{}, false, errors.New("cognitive orchestrator is not initialized") }
	if now.IsZero() { now = time.Now().UTC() }
	if err := ValidateInquiryExecution(result.Execution, now); err != nil { return CognitiveInterpretation{}, false, err }
	if result.Event.ID == "" { result.Event.ID = result.Execution.Request.RequestID }
	if result.Event.Timestamp.IsZero() { result.Event.Timestamp = now.UTC() }
	result.Event.Source = result.Observation.Source
	result.Event.Modality = result.Observation.Modality
	return o.ProcessObservation(result.Event, result.Observation, result.Experience)
}

func (o *CognitiveOrchestrator) ProcessInquiryOutcome(result InquiryResult, now time.Time) (CognitiveInterpretation, bool, uint64, error) {
	if o == nil || o.Runtime == nil { return CognitiveInterpretation{}, false, 0, errors.New("cognitive orchestrator is not initialized") }
	if now.IsZero() { now = time.Now().UTC() }
	if err := ValidateInquiryExecution(result.Execution, now); err != nil { return CognitiveInterpretation{}, false, 0, err }
	if len(result.Execution.Prediction.State) == 0 { return CognitiveInterpretation{}, false, 0, errors.New("inquiry execution has no pre-action prediction snapshot") }

	observedAt := result.Event.Timestamp
	if observedAt.IsZero() { observedAt = now.UTC() }
	source := result.Observation.Source
	if source == "" { source = "inquiry-outcome" }
	modality := result.Observation.Modality
	if modality == "" { modality = "inquiry-result" }

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
	if err != nil { return CognitiveInterpretation{}, false, 0, err }

	if result.Event.ID == "" { result.Event.ID = result.Execution.Request.RequestID }
	result.Event.Timestamp = observedAt
	result.Event.Source = source
	result.Event.Modality = modality
	// The prediction captured before the external action is the causal
	// reference for this outcome. Carry it explicitly through the runtime
	// instead of relying on whichever recurrent prediction happens to be
	// current when the device reports back.
	result.Event.PredictionOverride = clonePrediction(&result.Execution.Prediction)

	interpretation, learned, err := o.ProcessObservation(result.Event, result.Observation, result.Experience)
	if err != nil { return CognitiveInterpretation{}, learned, sequence, err }
	predictionError := o.Runtime.activation.PredictionError(result.Execution.Prediction, interpretation.State.Activations)
	if absFloat(predictionError-interpretation.State.PredictionError) > 1e-9 { return CognitiveInterpretation{}, learned, sequence, errors.New("inquiry prediction snapshot does not match cognitive prediction error") }
	result.PredictionError = predictionError

	consequence := AssessConsequence(ConsequenceInput{
		Success: result.Success,
		Reliability: result.Reliability,
		Reversible: result.Execution.Proposal.Reversibility,
		Cost: result.Cost,
		Risk: result.Risk,
		InformationGain: result.InformationGain,
		PredictionError: predictionError,
	})
	interpretation.Consequence = &consequence
	o.Runtime.activation.ApplyConsequencePlasticity(consequence.Valence, consequence.InformationGain, observedAt)
	binding, bindingOK := o.Runtime.actionBinding(result.Execution.Request.RequestID)
	outcomeNodeIDs := groundedPopulationNodeIDs(interpretation.GroundedRepresentations)
	causalLink := "inquiry:" + result.Execution.Request.RequestID
	consequenceEvent := InquiryConsequenceEvent{
		Action: result.Execution.Proposal.Action,
		Valence: consequence.Valence,
		InformationGain: consequence.InformationGain,
		PredictionError: consequence.PredictionError,
		Reliability: consequence.Reliability,
		CausalLink: causalLink,
		OutcomeNodeIDs: outcomeNodeIDs,
	}
	if bindingOK { consequenceEvent.TargetNodeIDs = binding.TargetNodeIDs }
	finalSequence, err := o.Runtime.RecordInquiryConsequenceEvent(consequenceEvent, observedAt)
	if err != nil { return CognitiveInterpretation{}, learned, sequence, err }
	return interpretation, learned, finalSequence, nil
}

func absFloat(v float64) float64 { if v < 0 { return -v }; return v }
