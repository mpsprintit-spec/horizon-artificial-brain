package repository

import (
	"context"
	"github.com/project-horizon/horizon-core/services/ai/sensor-fusion/model"
)

type NavigationStateRepository interface {
	SaveWorldState(context.Context, model.WorldState) error
	GetWorldState(context.Context, string) (model.WorldState, error)
}
type SensorSnapshotRepository interface {
	SaveSensorFrames(context.Context, []model.SensorFrame) error
	GetSensorFrames(context.Context, string) ([]model.SensorFrame, error)
}
