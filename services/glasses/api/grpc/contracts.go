package grpc

import (
	"context"
	"github.com/project-horizon/horizon-core/services/glasses/internal/model"
)

type RenderHUDHandler interface {
	RenderHUD(context.Context, model.HUDData) error
}
type PrioritizeInformationHandler interface {
	PrioritizeInformation(context.Context, []model.HUDData) (model.HUDData, error)
}
type SendAlertHandler interface {
	SendAlert(context.Context, model.Alert) (model.HUDData, error)
}
