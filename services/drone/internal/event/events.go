package event

import "github.com/project-horizon/horizon-core/services/ai/domain"

type Event = domain.DomainEvent
type Name = domain.EventName

const (
	BehaviorExecuted         = domain.EventBehaviorExecuted
	NavigationStateEstimated = domain.EventNavigationStateEstimated
	ObstacleDetected         = domain.EventObstacleDetected
	ObstacleTracked          = domain.EventObstacleTracked
	PathPlanned              = domain.EventPathPlanned
	ObstacleAvoidancePlanned = domain.EventObstacleAvoidancePlanned
	DroneCommandCreated      = domain.EventDroneCommandCreated
	DroneCommandDispatched   = domain.EventDroneCommandDispatched
	DroneCommandExecuted     = domain.EventDroneCommandExecuted
	FlightStabilized         = domain.EventFlightStabilized
	EnergyPlanUpdated        = domain.EventEnergyPlanUpdated
)
