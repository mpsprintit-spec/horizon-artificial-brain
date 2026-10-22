package runtime

import (
	"errors"
	"fmt"
)

// Reject foreign payloads rather than replaying only the recognized fraction
// of a mixed record. Legacy process/think retain their lower-level meaning.
func validateJournalPayload(event LoggedEvent) error {
	var required []string
	switch event.Type {
	case EventTypeProcess, EventTypeThink, EventTypeCognitiveProcess, EventTypeCognitiveThink, EventTypeSensorIntent:
		required = []string{"event"}
	case EventTypeObservation:
		required = []string{"event", "observation"}
	case EventTypeInquiryOutcome:
		required = []string{"event", "inquiry_outcome"}
	case EventTypeActionBinding:
		required = []string{"action_binding"}
	case EventTypeLearn:
		required = []string{"experience"}
	case EventTypeOutcome:
		required = []string{"outcome"}
	case EventTypeConsequence:
		required = []string{"consequence"}
	case EventTypeKnowledgeImport:
		required = []string{"document"}
	case EventTypeVectorGround:
		required = []string{"vector"}
	case EventTypeSensorCommit:
		required = []string{"intent_hash"}
	default:
		return fmt.Errorf("unknown event type %q", event.Type)
	}
	present := map[string]bool{"event": event.Event != nil, "experience": event.Experience != nil, "outcome": event.Outcome != nil, "consequence": event.Consequence != nil, "document": event.KnowledgeDocument != nil, "vector": event.Vector != nil, "intent_hash": event.IntentHash != "", "observation": event.ObservationCommand != nil, "action_binding": event.ActionBinding != nil, "inquiry_outcome": event.InquiryOutcome != nil}
	for _, field := range required {
		if !present[field] {
			return fmt.Errorf("%s record requires %s", event.Type, field)
		}
		delete(present, field)
	}
	for field, exists := range present {
		if exists {
			return fmt.Errorf("%s record contains foreign %s payload", event.Type, field)
		}
	}
	switch event.Type {
	case EventTypeObservation, EventTypeInquiryOutcome, EventTypeCognitiveProcess, EventTypeCognitiveThink:
		if event.Timestamp.IsZero() || !event.Timestamp.Equal(event.Event.Timestamp) || event.Event.SensorObservation != nil {
			return errors.New("command event time or sensor boundary is invalid")
		}
		if event.Type == EventTypeObservation {
			if err := validateObservationCommand(event.Event, event.ObservationCommand); err != nil {
				return err
			}
		}
		if event.Type == EventTypeInquiryOutcome {
			if err := validateInquiryOutcomeCommand(event.Event, event.InquiryOutcome); err != nil {
				return err
			}
		}
	case EventTypeActionBinding:
		if event.Timestamp.IsZero() || event.ActionBinding.RequestID == "" || event.ActionBinding.BrainIdentity != BrainIdentity {
			return errors.New("incomplete action binding command")
		}
	}
	if event.Type == EventTypeCognitiveThink || event.Type == EventTypeThink {
		e := event.Event
		if e.ID != "" || len(e.Stimulus)+len(e.StimulusNodeIDs)+len(e.ContextTokens)+len(e.DataTokens) != 0 || e.Source != "" || e.Modality != "" || e.PredictionOverride != nil || e.Observation != nil || e.SensorObservation != nil || event.Type == EventTypeCognitiveThink && len(e.Context) > 0 {
			return errors.New("think record contains ignored event fields")
		}
	}
	return nil
}
