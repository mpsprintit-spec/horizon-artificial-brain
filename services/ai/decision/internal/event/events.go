package event

import "github.com/project-horizon/horizon-core/services/ai/domain"

type Name = domain.EventName
type Event = domain.DomainEvent

const (
	NavigationStateEstimated = domain.EventNavigationStateEstimated
	ObjectDetected           = domain.EventObjectDetected
	ObstacleDetected         = domain.EventObstacleDetected
	MissionContextUpdated    = domain.EventMissionContextUpdated
	UserContextUpdated       = domain.EventUserContextUpdated
	SystemStatusUpdated      = domain.EventSystemStatusUpdated
	SituationAssessed        = domain.EventSituationAssessed
	RiskAnalyzed             = domain.EventRiskAnalyzed
	GoalPlanCreated          = domain.EventGoalPlanCreated
	BehaviorSelected         = domain.EventBehaviorSelected
	DecisionCreated          = domain.EventDecisionCreated
)
