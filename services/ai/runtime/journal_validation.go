package runtime

import (
	"errors"
	"fmt"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

// Reject foreign payloads rather than replaying only the recognized fraction
// of a mixed record. Legacy process/think retain their lower-level meaning.
func validateJournalPayload(event LoggedEvent) error {
	if err := event.SurfaceEncoding.Validate(); err != nil {
		return err
	}
	if event.SchemaVersion == 1 && event.SurfaceEncoding != knowledge.LegacySurfaceEncoding {
		return errors.New("versioned surface encoding requires journal schema 2")
	}
	if err := event.DynamicsEncoding.Validate(); err != nil {
		return err
	}
	if event.SchemaVersion < 3 && event.DynamicsEncoding != knowledge.LegacyDynamics {
		return errors.New("ordered neural dynamics require journal schema 3")
	}
	if event.Event != nil {
		if sensor := event.Event.SensorObservation; sensor != nil && sensor.NoiseTolerance > 0 && !event.DynamicsEncoding.HasStableProjection() {
			return errors.New("measurement tolerance requires stable projection dynamics")
		}
		if event.Event.NeuralTrace != nil && !event.DynamicsEncoding.HasOrderedObservations() {
			return errors.New("neural trace requires ordered dynamics")
		}
		if err := validateEpisodeControls(*event.Event, event.DynamicsEncoding); err != nil {
			return err
		}
		if err := event.Event.NeuralTrace.Validate(nil); err != nil {
			return err
		}
	}
	experiences := []*learning.Experience{event.Experience}
	if event.ObservationCommand != nil {
		experiences = append(experiences, event.ObservationCommand.Experience)
	}
	if event.InquiryOutcome != nil {
		experiences = append(experiences, event.InquiryOutcome.Observation.Experience)
	}
	for _, experience := range experiences {
		if experience == nil {
			continue
		}
		if !event.DynamicsEncoding.HasEvidenceProvenance() && (experience.EpisodeID != "" || experience.CausalGroup != "") {
			return errors.New("typed experience provenance requires evidence-provenance dynamics")
		}
		if err := validateExperienceStructure(*experience); err != nil {
			return err
		}
		if experience.HasOrderedFields() && !event.DynamicsEncoding.HasOrderedObservations() {
			return errors.New("ordered experience requires ordered dynamics")
		}
	}
	outcomes := []*OutcomeEvent{event.Outcome}
	if event.InquiryOutcome != nil {
		if event.Event != nil {
			if err := validateInquiryOutcomeDynamics(*event.Event, *event.InquiryOutcome, event.DynamicsEncoding); err != nil {
				return err
			}
		}
		outcomes = append(outcomes, &event.InquiryOutcome.Outcome)
	}
	for _, outcome := range outcomes {
		if outcome != nil {
			if err := validateOutcomeDynamics(*outcome, event.DynamicsEncoding); err != nil {
				return err
			}
		}
	}
	if c := event.Consequence; c != nil && !event.DynamicsEncoding.HasEvidenceProvenance() && (c.Source != "" || c.EpisodeID != "" || c.IndependenceGroup != "" || c.CausalGroup != "") {
		return errors.New("typed consequence provenance requires evidence-provenance dynamics")
	}
	var required []string
	switch event.Type {
	case EventTypeWrittenGroundingMigration:
		required = []string{"written_grounding_migration"}
	case EventTypeSurfaceRebinding:
		required = []string{"surface_rebinding"}
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
	present := map[string]bool{"event": event.Event != nil, "experience": event.Experience != nil, "outcome": event.Outcome != nil, "consequence": event.Consequence != nil, "document": event.KnowledgeDocument != nil, "vector": event.Vector != nil, "intent_hash": event.IntentHash != "", "observation": event.ObservationCommand != nil, "action_binding": event.ActionBinding != nil, "inquiry_outcome": event.InquiryOutcome != nil, "surface_rebinding": event.SurfaceRebinding != nil}
	present["written_grounding_migration"] = event.WrittenGroundingMigration != nil
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
	case EventTypeWrittenGroundingMigration:
		if event.SurfaceEncoding != knowledge.WrittenTextEncoding || event.SchemaVersion < 3 || event.Timestamp.IsZero() { return errors.New("numeric written migration requires schema 3 and event time") }
		if err := event.WrittenGroundingMigration.validateShape(); err != nil { return err }
		if event.Sequence != event.WrittenGroundingMigration.Plan.ReferenceRevision+1 { return errors.New("written migration reference revision disagrees with its journal position") }
	case EventTypeSurfaceRebinding:
		if event.SurfaceEncoding != knowledge.WrittenTextEncoding || event.Timestamp.IsZero() || event.SurfaceRebinding.ReferenceHash == "" {
			return errors.New("surface rebinding requires a versioned reference and event time")
		}
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
		if e.ID != "" || len(e.Stimulus)+len(e.StimulusNodeIDs)+len(e.ContextTokens)+len(e.DataTokens) != 0 || e.Source != "" || e.Modality != "" || e.EpisodeID != "" || e.ResetContext || e.NeuralTrace != nil || e.PredictionOverride != nil || e.Observation != nil || e.SensorObservation != nil || event.Type == EventTypeCognitiveThink && len(e.Context) > 0 {
			return errors.New("think record contains ignored event fields")
		}
	}
	return nil
}
