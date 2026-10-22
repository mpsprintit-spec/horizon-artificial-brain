package runtime

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"
)

// InquiryOutcomeCommand is observed causal data, never an execution request.
// Authorization is checked at live ingress; replay cannot authorize or repeat
// a device action. The pre-action prediction lives in LoggedEvent.Event.
type InquiryOutcomeCommand struct {
	Observation     ObservationCommand `json:"observation"`
	Outcome         OutcomeEvent       `json:"outcome"`
	Action          InquiryAction      `json:"action"`
	Reversible      bool               `json:"reversible"`
	Cost            float64            `json:"cost"`
	Risk            float64            `json:"risk"`
	InformationGain float64            `json:"information_gain"`
	ValidatedAt     time.Time          `json:"validated_at"`
}

func validateInquiryOutcomeCommand(event *Event, command *InquiryOutcomeCommand) error {
	if command == nil {
		return errors.New("inquiry outcome command is required")
	}
	if err := validateObservationCommand(event, &command.Observation); err != nil {
		return err
	}
	if err := validateOutcome(command.Outcome); err != nil {
		return err
	}
	if command.Action == "" || command.ValidatedAt.IsZero() || event.ID == "" || event.PredictionOverride == nil || len(event.PredictionOverride.State) == 0 {
		return errors.New("inquiry outcome requires action, identity, validation time and pre-action prediction")
	}
	if !command.Observation.PlanInquiry || command.Observation.Inquiry != nil || event.Source != command.Outcome.Source || event.Modality != command.Outcome.Modality || !event.Timestamp.Equal(command.Outcome.ObservedAt) || !slices.Equal(command.Outcome.Observation, command.Observation.Input.Tokens) {
		return errors.New("inquiry outcome provenance disagrees with observation")
	}
	return nil
}

func (r *BrainRuntime) processInquiryOutcomeCommand(event Event, command InquiryOutcomeCommand) (CognitiveInterpretation, bool, uint64, error) {
	if r == nil || r.brain == nil {
		return CognitiveInterpretation{}, false, 0, errors.New("brain runtime is not initialized")
	}
	data, err := json.Marshal(struct {
		Event   Event
		Command InquiryOutcomeCommand
	}{event, command})
	if err != nil {
		return CognitiveInterpretation{}, false, 0, err
	}
	var frozen struct {
		Event   Event
		Command InquiryOutcomeCommand
	}
	if err = json.Unmarshal(data, &frozen); err != nil {
		return CognitiveInterpretation{}, false, 0, err
	}
	event, command = frozen.Event, frozen.Command
	if err = validateInquiryOutcomeCommand(&event, &command); err != nil {
		return CognitiveInterpretation{}, false, 0, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err = r.ensureWritableLocked(); err != nil {
		return CognitiveInterpretation{}, false, r.seq, err
	}
	if _, exists := r.processedEvents[event.ID]; exists {
		return CognitiveInterpretation{}, false, r.seq, fmt.Errorf("duplicate event ID %q", event.ID)
	}
	binding, ok := r.actions[command.Outcome.RequestID]
	if !ok || binding.BrainIdentity != BrainIdentity {
		return CognitiveInterpretation{}, false, r.seq, errors.New("outcome has no registered action binding")
	}
	if command.Observation.Experience != nil {
		if err = r.validateExperienceLocked(*command.Observation.Experience); err != nil {
			return CognitiveInterpretation{}, false, r.seq, err
		}
	}
	if err = r.validateNeuralEventLocked(event); err != nil {
		return CognitiveInterpretation{}, false, r.seq, err
	}
	var cognitive CognitiveInterpretation
	learned := false
	err = r.runJournalCommandLocked(LoggedEvent{Type: EventTypeInquiryOutcome, Timestamp: event.Timestamp, Event: cloneEvent(event), InquiryOutcome: &command}, func() error {
		if err := r.applyOutcomeLocked(command.Outcome, binding); err != nil {
			return err
		}
		var applyErr error
		cognitive, learned, applyErr = r.applyObservationCommandLocked(event, command.Observation)
		if applyErr != nil {
			return applyErr
		}
		predictionError := r.activation.PredictionError(*event.PredictionOverride, cognitive.State.Activations)
		if absFloat(predictionError-cognitive.State.PredictionError) > 1e-9 {
			return errors.New("inquiry prediction snapshot does not match cognitive prediction error")
		}
		consequence := AssessConsequence(ConsequenceInput{Success: command.Outcome.Success, Reliability: command.Outcome.Reliability, Reversible: command.Reversible, Cost: command.Cost, Risk: command.Risk, InformationGain: command.InformationGain, PredictionError: predictionError})
		cognitive.Consequence = &consequence
		r.activation.ApplyConsequencePlasticity(consequence.Valence, consequence.InformationGain, event.Timestamp)
		r.applyInquiryConsequenceLocked(InquiryConsequenceEvent{Action: command.Action, Valence: consequence.Valence, InformationGain: consequence.InformationGain, PredictionError: consequence.PredictionError, Reliability: consequence.Reliability, TargetNodeIDs: binding.TargetNodeIDs, OutcomeNodeIDs: groundedPopulationNodeIDs(cognitive.GroundedRepresentations), CausalLink: "inquiry:" + command.Outcome.RequestID}, event.Timestamp)
		return nil
	})
	if err != nil {
		return CognitiveInterpretation{}, false, r.seq, err
	}
	if r.telemetry != nil {
		e := r.telemetryEventLocked(EventTypeProcess, event.Timestamp, &event, event.Observation, &command.Outcome)
		e.StateDelta, e.PredictionError = &cognitive.ChangeAwareness.StateDelta, &cognitive.State.PredictionError
		r.telemetry.Publish(e)
	}
	return cognitive, learned, r.seq, nil
}
