package handler

import (
	"context"
	"github.com/project-horizon/horizon-core/services/glasses/internal/model"
)

type HUDUpdateHandler interface {
	RenderHUD(context.Context, model.HUDData) error
}
type AlertHandler interface {
	SendAlert(context.Context, model.Alert) (model.HUDData, error)
}
type UserAttentionHandler interface {
	UpdateAttention(context.Context, model.UserContext) (model.UserContext, error)
}
