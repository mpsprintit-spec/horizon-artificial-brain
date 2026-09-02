package handler

import (
	"context"
	"github.com/project-horizon/horizon-core/services/drone/internal/event"
	"github.com/project-horizon/horizon-core/services/drone/internal/model"
)

type CommandHandler interface {
	DispatchCommand(context.Context, model.DroneCommand) error
}
type TelemetryHandler interface {
	HandleTelemetry(context.Context, model.Telemetry) error
}
type ObstacleUpdateHandler interface {
	HandleObstacleUpdate(context.Context, event.Event) error
}
