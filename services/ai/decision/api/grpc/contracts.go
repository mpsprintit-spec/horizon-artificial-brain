package grpc

import (
	"context"
	"github.com/project-horizon/horizon-core/services/ai/decision/internal/model"
)

type CreateDecisionHandler interface {
	CreateDecision(context.Context, model.MissionContext, model.UserContext, model.BehaviorSelection) (model.Decision, error)
}
type AssessSituationHandler interface {
	AssessSituation(context.Context, model.MissionContext, model.NavigationState, []model.Detection) (model.SituationAssessment, error)
}
type SelectBehaviorHandler interface {
	SelectBehavior(context.Context, model.SituationAssessment, model.GoalPlan) (model.BehaviorSelection, error)
}
type GetDecisionHandler interface {
	GetDecision(context.Context, string) (model.Decision, error)
}
