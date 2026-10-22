package runtime

import (
	"errors"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func validateOutcomeDynamics(o OutcomeEvent, d knowledge.DynamicsEncoding) error {
	if !d.HasEvidenceProvenance() && (o.EpisodeID != "" || o.IndependenceGroup != "" || o.CausalGroup != "") {
		return errors.New("typed outcome provenance requires evidence-provenance dynamics")
	}
	return nil
}

func validateInquiryOutcomeDynamics(event Event, command InquiryOutcomeCommand, d knowledge.DynamicsEncoding) error {
	if err := validateOutcomeDynamics(command.Outcome, d); err != nil {
		return err
	}
	if d.HasEvidenceProvenance() && command.Outcome.EpisodeID != event.EpisodeID {
		return errors.New("inquiry outcome episode disagrees with observation")
	}
	return nil
}
