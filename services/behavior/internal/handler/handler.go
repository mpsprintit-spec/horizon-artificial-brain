package handler

import (
	"context"
	"github.com/project-horizon/horizon-core/services/behavior/internal/event"
	"github.com/project-horizon/horizon-core/services/behavior/internal/model"
)

type SelectionEventHandler interface {
	HandleBehaviorSelected(context.Context, event.Event) error
}
type ExecutionCommandHandler interface {
	ExecuteBehavior(context.Context, model.BehaviorPlan) (model.BehaviorExecution, error)
}
