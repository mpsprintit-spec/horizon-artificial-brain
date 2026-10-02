package plugin

import "github.com/project-horizon/horizon-core/services/ai/knowledge"

// ExperientialPlugin is an optional extension of Plugin for actions whose
// consequences can be predicted and observed. Existing Plugin implementations
// remain source-compatible and can continue to use Trigger.
type ExperientialPlugin interface {
	Plugin
	Predict(contextData string) []knowledge.InquiryCandidate
	Execute(contextData string) knowledge.InquiryOutcome
}
