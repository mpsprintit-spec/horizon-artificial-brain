package model

import "github.com/project-horizon/horizon-core/services/ai/domain"

type InformationPrioritizer = domain.InformationPrioritizer
type ARPlacementEngine = domain.ARPlacementEngine
type TargetHighlighter = domain.TargetHighlighter
type AlertNotifier = domain.AlertNotifier
type AttentionManager = domain.AttentionManager
type HUDRenderer = domain.HUDRenderer

type HUDAIService interface {
	InformationPrioritizer
	ARPlacementEngine
	TargetHighlighter
	AlertNotifier
	AttentionManager
	HUDRenderer
}
