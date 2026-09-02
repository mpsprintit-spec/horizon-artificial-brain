package repository

import (
	"context"
	"github.com/project-horizon/horizon-core/services/ai/decision/internal/model"
)

type DecisionRepository interface {
	SaveDecision(context.Context, model.Decision) error
	GetDecision(context.Context, string) (model.Decision, error)
}
type RiskAssessmentRepository interface {
	SaveRiskAssessment(context.Context, model.RiskAssessment) error
	GetRiskAssessment(context.Context, string) (model.RiskAssessment, error)
}
type GoalPlanRepository interface {
	SaveGoalPlan(context.Context, model.GoalPlan) error
	GetGoalPlan(context.Context, string) (model.GoalPlan, error)
}
