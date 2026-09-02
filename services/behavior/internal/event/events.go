package event

import "github.com/project-horizon/horizon-core/services/ai/domain"

type Event = domain.DomainEvent
type Name = domain.EventName

const (
	BehaviorSelected = domain.EventBehaviorSelected
	DecisionCreated  = domain.EventDecisionCreated
	BehaviorPlanned  = domain.EventBehaviorPlanned
	BehaviorExecuted = domain.EventBehaviorExecuted
	BehaviorFailed   = domain.EventBehaviorFailed
)
