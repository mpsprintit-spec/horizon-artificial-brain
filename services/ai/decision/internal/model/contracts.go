package model

import (
	"context"
	"github.com/project-horizon/horizon-core/services/ai/domain"
)

type SituationAssessor = domain.SituationAssessor
type RiskAnalyzer = domain.RiskAnalyzer
type GoalPlanner = domain.GoalPlanner
type BehaviorSelector = domain.BehaviorSelector
type DecisionOptimizer = domain.DecisionOptimizer
type ContextAwareDecisionEngine = domain.ContextAwareDecisionEngine

type DecisionService interface {
	SituationAssessor
	RiskAnalyzer
	GoalPlanner
	BehaviorSelector
	DecisionOptimizer
	ContextAwareDecisionEngine
}

type DecisionCreator interface {
	CreateDecision(context.Context, MissionContext, UserContext, BehaviorSelection) (Decision, error)
}
