package http

import (
	"context"
	"github.com/project-horizon/horizon-core/services/ai/sensor-fusion/model"
)

type FuseSensorsHandler interface {
	FuseSensors(context.Context, []model.SensorFrame) (model.WorldState, error)
}
type EstimateStateHandler interface {
	EstimateState(context.Context, model.SensorState) (model.WorldState, error)
}
type GetNavigationStateHandler interface {
	GetNavigationState(context.Context, string) (model.WorldState, error)
}
