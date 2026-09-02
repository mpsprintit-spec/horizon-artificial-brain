package event

import "github.com/project-horizon/horizon-core/services/ai/domain"

type Event = domain.DomainEvent
type Name = domain.EventName

const (
	DecisionCreated      = domain.EventDecisionCreated
	BehaviorSelected     = domain.EventBehaviorSelected
	ObjectDetected       = domain.EventObjectDetected
	HUDDataPrepared      = domain.EventHUDDataPrepared
	HUDRendered          = domain.EventHUDRendered
	AlertPrioritized     = domain.EventAlertPrioritized
	AlertDisplayed       = domain.EventAlertDisplayed
	UserAttentionUpdated = domain.EventUserAttentionUpdated
)
