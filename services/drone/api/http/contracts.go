package http

import (
	"context"
	"github.com/project-horizon/horizon-core/services/drone/internal/model"
)

type PlanPathHandler interface {
	PlanPath(context.Context, model.BehaviorPlan, model.NavigationState) (model.FlightPlan, error)
}
type DispatchCommandHandler interface {
	DispatchCommand(context.Context, model.DroneCommand) error
}
type GetFlightStateHandler interface {
	GetFlightState(context.Context, string) (model.NavigationState, error)
}
