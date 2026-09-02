package handler

import (
	"context"
	"github.com/project-horizon/horizon-core/services/ai/sensor-fusion/model"
)

type SensorPacketHandler interface {
	FuseSensors(context.Context, []model.SensorFrame) (model.WorldState, error)
}
type TelemetryHandler interface {
	HandleTelemetry(context.Context, model.SensorFrame) error
}
type StateQueryHandler interface {
	GetNavigationState(context.Context, string) (model.WorldState, error)
}
