package repository

import (
	"context"
	"github.com/project-horizon/horizon-core/services/behavior/internal/model"
)

type BehaviorPlanRepository interface {
	SaveBehaviorPlan(context.Context, model.BehaviorPlan) error
	GetBehaviorPlan(context.Context, string) (model.BehaviorPlan, error)
}
type BehaviorExecutionRepository interface {
	SaveBehaviorExecution(context.Context, model.BehaviorExecution) error
	GetBehaviorExecution(context.Context, string) (model.BehaviorExecution, error)
}
