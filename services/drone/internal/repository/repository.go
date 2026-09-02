package repository

import (
	"context"
	"github.com/project-horizon/horizon-core/services/drone/internal/model"
)

type FlightPlanRepository interface {
	SaveFlightPlan(context.Context, model.FlightPlan) error
	GetFlightPlan(context.Context, string) (model.FlightPlan, error)
}
type DroneCommandRepository interface {
	SaveDroneCommand(context.Context, model.DroneCommand) error
	GetDroneCommand(context.Context, string) (model.DroneCommand, error)
}
type DroneTelemetryRepository interface {
	SaveTelemetry(context.Context, model.Telemetry) error
	GetTelemetry(context.Context, string) (model.Telemetry, error)
}
