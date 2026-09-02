package learning

import (
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/memory"
)

type LearningUnit struct {
	Kb          *knowledge.KnowledgeBase
	Memory      *memory.Engine
	LastTouched []TouchedRelation
}

func NewLearningUnit(kb *knowledge.KnowledgeBase) *LearningUnit {
	return &LearningUnit{Kb: kb, Memory: memory.NewEngine(kb)}
}
