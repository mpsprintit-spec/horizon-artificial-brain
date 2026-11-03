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
	if event.SchemaVersion == coordinatedJournalSchemaVersion && (event.Type != EventTypeCognitiveModel || event.CognitiveModel == nil || event.CognitiveModel.Coordinated == nil) {
		return errors.New("schema13 reserved for coordinated V5")
	}
	if event.SchemaVersion == terminalExposureJournalSchemaVersion && event.Type != EventTypeCognitiveModel {
		return errors.New("schema12 reserved for controlled terminal exposure")
	}
	if event.SchemaVersion == phase4JournalSchemaVersion && event.Type != EventTypeCognitiveModel {
		return errors.New("schema11 reserved for V5 Phase IV")
	}
	if event.SchemaVersion == phase3JournalSchemaVersion && event.Type != EventTypeCognitiveModel {
		return errors.New("journal schema10 is reserved for explicit phase3 commands")
	}
	if event.SchemaVersion == phase2JournalSchemaVersion && event.Type != EventTypeCognitiveModel {
		return errors.New("journal schema9 is reserved for explicit phase2 commands")
	}
	if event.CompositionEncoding != "" && (!knowledge.SupportedCompositionVersion(event.CompositionEncoding) || event.SchemaVersion < compositionJournalSchemaVersion) {
		return errors.New("unsupported composition journal contract")
	}
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
		if sensor := event.Event.SensorObservation; sensor != nil && sensor.CompositionScene != nil && (event.SchemaVersion < compositionJournalSchemaVersion || !knowledge.SupportedCompositionVersion(event.CompositionEncoding)) {
			return errors.New("composition sensor scene requires its active journal contract")
		}
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
	case EventTypeCognitiveModel:
		required = []string{"cognitive_model"}
	case EventTypeCompositionMigration:
		required = []string{"composition_migration"}
	case EventTypeCompositionPerception:
		required = []string{"composition_perception"}
	case EventTypeCompositionExposure:
		required = []string{"composition_exposure"}
	case EventTypeSpellingExposure:
		required = []string{"spelling_exposure"}
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
	present := map[string]bool{"cognitive_model": event.CognitiveModel != nil, "event": event.Event != nil, "experience": event.Experience != nil, "outcome": event.Outcome != nil, "consequence": event.Consequence != nil, "document": event.KnowledgeDocument != nil, "vector": event.Vector != nil, "intent_hash": event.IntentHash != "", "observation": event.ObservationCommand != nil, "action_binding": event.ActionBinding != nil, "inquiry_outcome": event.InquiryOutcome != nil, "surface_rebinding": event.SurfaceRebinding != nil}
	present["written_grounding_migration"] = event.WrittenGroundingMigration != nil
	present["spelling_exposure"] = event.SpellingExposure != nil
	present["composition_migration"] = event.CompositionMigration != nil
	present["composition_exposure"] = event.CompositionExposure != nil
	present["composition_perception"] = event.CompositionPerception != nil
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
	case EventTypeCognitiveModel:
		if (event.SchemaVersion != coordinatedJournalSchemaVersion && event.SchemaVersion != terminalExposureJournalSchemaVersion && event.SchemaVersion != phase4JournalSchemaVersion && event.SchemaVersion != phase3JournalSchemaVersion && event.SchemaVersion != phase2JournalSchemaVersion && event.SchemaVersion != cognitiveModelJournalSchemaVersion && event.SchemaVersion != neuralMetaJournalSchemaVersion && event.SchemaVersion != terminalReadoutJournalSchemaVersion && event.SchemaVersion != componentLearningJournalSchemaVersion) || event.Timestamp.IsZero() {
			return errors.New("invalid cognitive journal contract")
		}
		if err := event.CognitiveModel.Validate(); err != nil {
			return err
		}
		c := event.CognitiveModel
		if (c.Coordinated != nil) != (event.SchemaVersion == coordinatedJournalSchemaVersion) {
			return errors.New("coordinated update requires schema13")
		}
		terminalExposure := c.Phase4Training != nil && c.Phase4Training.Protocol == knowledge.Phase4TerminalProtocolV5
		if terminalExposure != (event.SchemaVersion == terminalExposureJournalSchemaVersion) {
			return errors.New("terminal exposure requires explicit schema12")
		}
		if (c.Phase4Init != nil || c.Phase4Training != nil) != (event.SchemaVersion == phase4JournalSchemaVersion || event.SchemaVersion == terminalExposureJournalSchemaVersion) {
			return errors.New("Phase IV requires explicit schema11 or terminal schema12")
		}
		if (event.CognitiveModel.Phase3Training != nil) != (event.SchemaVersion == phase3JournalSchemaVersion) {
			return errors.New("phase3 learning requires explicit schema10")
		}
		phase2 := event.CognitiveModel.Phase2Init != nil || event.CognitiveModel.Phase2Training != nil
		if phase2 != (event.SchemaVersion == phase2JournalSchemaVersion) {
			return errors.New("phase2 commands require their explicit journal schema9")
		}
		if c := event.CognitiveModel; c.Calibration != nil && c.Calibration.TerminalReadoutOnly && event.SchemaVersion != terminalReadoutJournalSchemaVersion {
			return errors.New("terminal readout calibration requires journal schema 7")
		}
		if event.CognitiveModel.Training != nil && event.CognitiveModel.Training.TerminalReadoutOnly && event.SchemaVersion != terminalReadoutJournalSchemaVersion {
			return errors.New("terminal readout fitting requires journal schema 7")
		}
		if c := event.CognitiveModel; (c.Training != nil && (c.Training.ComponentV4 != "" || c.Training.EntropyWeightV4 != nil) || c.Calibration != nil && (c.Calibration.ComponentV4 != "" || c.Calibration.EntropyWeightV4 != nil)) && event.SchemaVersion != componentLearningJournalSchemaVersion {
			return errors.New("component learning requires journal schema 8")
		}
		if event.SchemaVersion != coordinatedJournalSchemaVersion && event.SchemaVersion != terminalExposureJournalSchemaVersion && event.SchemaVersion != phase4JournalSchemaVersion && event.SchemaVersion != phase3JournalSchemaVersion && event.SchemaVersion != phase2JournalSchemaVersion && event.SchemaVersion != neuralMetaJournalSchemaVersion && event.SchemaVersion != terminalReadoutJournalSchemaVersion && event.SchemaVersion != componentLearningJournalSchemaVersion {
			c := event.CognitiveModel
			if c.Calibration != nil || (c.Initialize != nil && c.Initialize.Version == knowledge.CognitiveModelV2) || (c.Input != nil && c.Input.NeuralMeta != nil) {
				return errors.New("native neural metareasoning requires journal schema 6")
			}
			if c.Training != nil {
				for _, e := range c.Training.Episodes {
					if e.VerifiedTarget != nil || len(e.VerifiedOperations) > 0 {
						return errors.New("native verified learning requires journal schema 6")
					}
				}
			}
		}
	case EventTypeCompositionMigration:
		if event.SchemaVersion < compositionJournalSchemaVersion || event.Timestamp.IsZero() || event.CompositionEncoding != "" {
			return errors.New("composition migration requires an initial versioned command")
		}
		if err := event.CompositionMigration.validateShape(); err != nil {
			return err
		}
		if event.Sequence != event.CompositionMigration.Plan.ReferenceRevision+1 {
			return errors.New("composition migration reference revision disagrees with journal")
		}
	case EventTypeCompositionPerception:
		if event.SchemaVersion < compositionJournalSchemaVersion || !knowledge.SupportedCompositionVersion(event.CompositionEncoding) || event.Timestamp.IsZero() {
			return errors.New("invalid sequence perception journal contract")
		}
		if err := event.CompositionPerception.validate(event.Timestamp); err != nil {
			return err
		}
	case EventTypeCompositionExposure:
		if event.SchemaVersion < compositionJournalSchemaVersion || event.Timestamp.IsZero() || !knowledge.SupportedCompositionVersion(event.CompositionEncoding) {
			return errors.New("composition exposure requires its active versioned contract")
		}
		if err := event.CompositionExposure.Validate(); err != nil {
			return err
		}
	case EventTypeSpellingExposure:
		if event.SchemaVersion < 3 || event.Timestamp.IsZero() {
			return errors.New("spelling supervision requires schema 3 and event time")
		}
		if err := event.SpellingExposure.Validate(); err != nil {
			return err
		}
	case EventTypeWrittenGroundingMigration:
		if event.SurfaceEncoding != knowledge.WrittenTextEncoding || event.SchemaVersion < 3 || event.Timestamp.IsZero() {
			return errors.New("numeric written migration requires schema 3 and event time")
		}
		if err := event.WrittenGroundingMigration.validateShape(); err != nil {
			return err
		}
		if event.Sequence != event.WrittenGroundingMigration.Plan.ReferenceRevision+1 {
			return errors.New("written migration reference revision disagrees with its journal position")
		}
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
		if p := event.ActionBinding.NumericPrediction; p != nil {
			if err := p.validate(); err != nil {
				return err
			}
			if !knowledge.SupportedCompositionVersion(event.CompositionEncoding) || event.SchemaVersion < compositionJournalSchemaVersion || p.ReferenceRevision+1 != event.Sequence {
				return errors.New("numeric action prediction has no current versioned reference")
			}
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
