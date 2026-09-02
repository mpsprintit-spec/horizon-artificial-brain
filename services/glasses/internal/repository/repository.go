package repository

import (
	"context"
	"github.com/project-horizon/horizon-core/services/glasses/internal/model"
)

type HUDSessionRepository interface {
	SaveHUDSession(context.Context, model.HUDData) error
	GetHUDSession(context.Context, string) (model.HUDData, error)
}
type HUDDataRepository interface {
	SaveHUDData(context.Context, model.HUDData) error
	GetHUDData(context.Context, string) (model.HUDData, error)
}
type AlertRepository interface {
	SaveAlert(context.Context, model.Alert) error
	GetAlert(context.Context, string) (model.Alert, error)
}
