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

	if result.Event.ID == "" {
		result.Event.ID = result.Execution.Request.RequestID
	}
	result.Event.Timestamp = observedAt.UTC()
	result.Event.StimulusNodeIDs = nil
	result.Event.PredictionOverride = clonePrediction(&result.Execution.Prediction)
	result.Observation.Source, result.Observation.Modality = source, modality
	observation := ObservationCommand{Input: result.Observation, Experience: result.Experience, PlanInquiry: true}
	// Own the optional experience before filling its default provenance.
	if observation.Experience != nil {
		copy := *observation.Experience
		observation.Experience = &copy
	}
	result.Event, observation = normalizeObservationCommand(result.Event, observation, observedAt)
	return o.Runtime.processInquiryOutcomeCommand(result.Event, InquiryOutcomeCommand{
		Observation: observation,
		Outcome:     OutcomeEvent{RequestID: result.Execution.Request.RequestID, BrainIdentity: result.Execution.Request.BrainIdentity, Success: result.Success, Observation: append([]string(nil), result.Observation.Tokens...), Source: source, Modality: modality, ObservedAt: observedAt.UTC(), Reliability: result.Reliability},
		Action:      result.Execution.Proposal.Action,
		Reversible:  result.Execution.Proposal.Reversibility,
		Cost:        result.Cost, Risk: result.Risk, InformationGain: result.InformationGain, ValidatedAt: now.UTC(),
	})
}

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
