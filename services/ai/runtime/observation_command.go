package runtime

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

// ObservationCommand preserves the complete adapter boundary, including
// optional learning and inquiry completion. Event is stored in LoggedEvent's
// normal event field so event inspection retains the original observation.
type ObservationCommand struct {
	Input       ObservationInput     `json:"input"`
	Experience  *learning.Experience `json:"experience,omitempty"`
	PlanInquiry bool                 `json:"plan_inquiry"`
	Inquiry     *InquiryRequest      `json:"inquiry,omitempty"`
}

func validateObservationCommand(event *Event, command *ObservationCommand) error {
	if event == nil || command == nil {
		return errors.New("observation command requires event and adapter input")
	}
	if event.SensorObservation != nil || len(event.StimulusNodeIDs) != 0 {
		return errors.New("observation command must precede neural grounding")
	}
	if event.Source != command.Input.Source || event.Modality != command.Input.Modality || !slices.Equal(event.ContextTokens, command.Input.ContextTokens) || !slices.Equal(event.DataTokens, command.Input.DataTokens) {
		return errors.New("observation event and adapter provenance disagree")
	}
	if envelope := event.Observation; envelope != nil {
		if err := envelope.Validate(); err != nil {
			return err
		}
		if envelope.EventID != event.ID || !envelope.Timestamp.Equal(event.Timestamp) || envelope.Source != event.Source || envelope.Modality != event.Modality || !slices.Equal(envelope.ContextTokens, command.Input.ContextTokens) || !slices.Equal(envelope.DataTokens, command.Input.DataTokens) || len(envelope.Tokens) > 0 && !slices.Equal(envelope.Tokens, command.Input.Tokens) {
			return errors.New("observation envelope and adapter input disagree")
		}
	}
	for _, group := range [][]string{command.Input.Tokens, command.Input.ContextTokens, command.Input.DataTokens} {
		for _, token := range group {
			if strings.TrimSpace(token) == "" {
				return errors.New("observation contains an empty surface")
			}
		}
	}
	if command.Experience != nil {
		for _, token := range command.Experience.Sequence {
			if strings.TrimSpace(token) == "" {
				return errors.New("experience contains an empty surface")
			}
		}
	}
	if command.Inquiry != nil && (command.Inquiry.BrainIdentity != BrainIdentity || command.Inquiry.RequestID == "" || command.Inquiry.Action == "") {
		return errors.New("invalid inquiry observation request")
	}
	return nil
}

func (r *BrainRuntime) processObservationCommand(event Event, command ObservationCommand) (CognitiveInterpretation, bool, error) {
	if r == nil || r.brain == nil {
		return CognitiveInterpretation{}, false, errors.New("brain runtime is not initialized")
	}
	// Freeze the event, envelope, token groups and optional experience together.
	packet, err := json.Marshal(struct {
		Event   Event
		Command ObservationCommand
	}{event, command})
	if err != nil {
		return CognitiveInterpretation{}, false, err
	}
	var frozen struct {
		Event   Event
		Command ObservationCommand
	}
	if err = json.Unmarshal(packet, &frozen); err != nil {
		return CognitiveInterpretation{}, false, err
	}
	event, command = frozen.Event, frozen.Command
	event.StimulusNodeIDs = nil
	r.mu.Lock()
	defer r.mu.Unlock()
	if err = r.ensureWritableLocked(); err != nil {
		return CognitiveInterpretation{}, false, err
	}
	if event.ID != "" {
		if _, exists := r.processedEvents[event.ID]; exists {
			return CognitiveInterpretation{}, false, fmt.Errorf("duplicate event ID %q", event.ID)
		}
	}
	event, command = normalizeObservationCommand(event, command, r.nowLocked())
	if err = validateObservationCommand(&event, &command); err != nil {
		return CognitiveInterpretation{}, false, err
	}
	var cognitive CognitiveInterpretation
	learned := false
	err = r.runJournalCommandLocked(LoggedEvent{Type: EventTypeObservation, Timestamp: event.Timestamp, Event: cloneEvent(event), ObservationCommand: &command}, func() error {
		var applyErr error
		cognitive, learned, applyErr = r.applyObservationCommandLocked(event, command)
		return applyErr
	})
	if err != nil {
		return CognitiveInterpretation{}, false, err
	}
	if r.telemetry != nil {
		e := r.telemetryEventLocked(EventTypeProcess, event.Timestamp, &event, event.Observation, nil)
		e.StateDelta = &cognitive.ChangeAwareness.StateDelta
		e.PredictionError = &cognitive.State.PredictionError
		r.telemetry.Publish(e)
	}
	return cognitive, learned, nil
}

func normalizeObservationCommand(event Event, command ObservationCommand, at time.Time) (Event, ObservationCommand) {
	if event.Timestamp.IsZero() {
		event.Timestamp = at
	}
	event.Timestamp = event.Timestamp.UTC()
	event.Source, event.Modality = command.Input.Source, command.Input.Modality
	event.ContextTokens = append([]string(nil), command.Input.ContextTokens...)
	event.DataTokens = append([]string(nil), command.Input.DataTokens...)
	if experience := command.Experience; experience != nil {
		if experience.ExperienceID == "" {
			experience.ExperienceID = event.ID
		}
		if experience.Source == "" {
			experience.Source = command.Input.Source
		}
		if experience.Modality == "" {
			experience.Modality = command.Input.Modality
		}
		if experience.Timestamp.IsZero() {
			experience.Timestamp = event.Timestamp
		}
	}
	return event, command
}

func (r *BrainRuntime) applyObservationCommandLocked(event Event, command ObservationCommand) (CognitiveInterpretation, bool, error) {
	input := command.Input
	grounded := make([]knowledge.GroundedRepresentation, 0)
	seen := make(map[string]bool)
	for _, group := range [][]string{input.Tokens, input.ContextTokens, input.DataTokens} {
		for _, token := range group {
			if seen[token] {
				continue
			}
			seen[token] = true
			representation, err := r.brain.GroundObservationWithEncodingAt(token, input.Source, input.Modality, GroundingThreshold, event.Timestamp, r.surfaceEncoding)
			if err != nil {
				return CognitiveInterpretation{}, false, err
			}
			grounded = append(grounded, representation)
		}
	}
	event.StimulusNodeIDs = groundedPopulationNodeIDs(grounded)
	output, err := r.cognitiveProcessWithPersistenceLocked(event, false)
	if err != nil {
		return CognitiveInterpretation{}, false, err
	}
	if err = r.learnObservedTransitionLocked(event.StimulusNodeIDs, event.Timestamp); err != nil {
		return CognitiveInterpretation{}, false, err
	}
	learned := command.Experience != nil && len(command.Experience.Sequence) > 0
	if learned {
		r.learning.LearnExperienceWithEncoding(*command.Experience, event.Timestamp, r.surfaceEncoding)
	}
	cognitive, err := r.InterpretCognitive(output, Observation{Source: input.Source, Modality: input.Modality, Tokens: input.Tokens, ContextTokens: input.ContextTokens, DataTokens: input.DataTokens})
	if err != nil {
		return CognitiveInterpretation{}, false, err
	}
	cognitive.GroundedRepresentations = grounded
	if command.Inquiry != nil {
		r.completeInquiryLocked(clamp01(command.Inquiry.Uncertainty-clamp01(cognitive.Answer.Uncertainty.Level)), output.PredictionError, event.Timestamp)
	}
	if command.PlanInquiry {
		agenda, err := r.planInquiryLocked(cognitive.Answer.Uncertainty.Level, event.Timestamp)
		if err != nil {
			return CognitiveInterpretation{}, false, err
		}
		cognitive.InquiryAgenda = &agenda
	}
	cognitive.ChangeAwareness = InternalChangeAwareness{BrainIdentity: BrainIdentity, Sequence: output.Sequence, Timestamp: output.Timestamp, StateDelta: output.StateDelta, KnowledgeChanges: []KnowledgeInjectionEvent{{BrainIdentity: BrainIdentity, Sequence: output.Sequence, Timestamp: event.Timestamp, Claims: knowledgeClaimsFromGrounding(grounded, event.Timestamp), ExperienceID: experienceID(command.Experience), ChangedNodeIDs: groundedNodeIDs(grounded)}}}
	return cognitive, learned, nil
}
