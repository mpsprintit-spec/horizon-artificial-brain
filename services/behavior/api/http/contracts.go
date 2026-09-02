package http

import (
	"context"
	"github.com/project-horizon/horizon-core/services/behavior/internal/model"
)

type ExecuteBehaviorHandler interface {
	ExecuteBehavior(context.Context, model.BehaviorPlan) (model.BehaviorExecution, error)
}
type GetBehaviorPlanHandler interface {
	GetBehaviorPlan(context.Context, string) (model.BehaviorPlan, error)
}
type GetBehaviorExecutionHandler interface {
	GetBehaviorExecution(context.Context, string) (model.BehaviorExecution, error)
}
