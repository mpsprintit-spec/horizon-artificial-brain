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
	Event     Event
	Observation ObservationInput
	Experience *learning.Experience
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
