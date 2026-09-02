package handler

import (
	"context"
	"github.com/project-horizon/horizon-core/services/ai/decision/internal/event"
	"github.com/project-horizon/horizon-core/services/ai/decision/internal/model"
)

type CommandHandler interface {
	CreateDecision(context.Context, model.MissionContext, model.UserContext, model.BehaviorSelection) (model.Decision, error)
}
type ContextUpdateHandler interface {
	HandleContextUpdate(context.Context, event.Event) error
}
type QueryHandler interface {
	GetDecision(context.Context, string) (model.Decision, error)
}
