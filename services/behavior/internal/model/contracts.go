package model

import "github.com/project-horizon/horizon-core/services/ai/domain"

type BehaviorExecutor = domain.BehaviorExecutor
type BehaviorPlanner = domain.BehaviorPlanner
type ActionDispatcher = domain.ActionDispatcher
type BehaviorPolicy = domain.BehaviorPolicy

type BehaviorService interface {
	BehaviorExecutor
	BehaviorPlanner
	ActionDispatcher
	BehaviorPolicy
}
