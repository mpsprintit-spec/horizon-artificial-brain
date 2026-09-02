package core

import (
	"github.com/project-horizon/horizon-core/services/ai/decision"
	"github.com/project-horizon/horizon-core/services/ai/execution"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/language"
	"github.com/project-horizon/horizon-core/services/ai/learning"
	"github.com/project-horizon/horizon-core/services/ai/perception"
	"github.com/project-horizon/horizon-core/services/ai/thinking"
	"github.com/project-horizon/horizon-core/services/ai/websearch"
)

type HorizonEngine struct {
	Knowledge  *knowledge.KnowledgeBase
	Learning   *learning.LearningUnit
	Thinking   *thinking.ThinkingEngine
	Decision   *decision.Engine
	Language   *language.Engine
	Execution  *execution.ExecutionCore
	WebSearch  *websearch.Engine
	Perception perception.PerceptionLayer
}

func NewHorizonEngine() *HorizonEngine {
	kb := knowledge.NewKnowledgeBase()
	return &HorizonEngine{
		Knowledge:  kb,
		Learning:   learning.NewLearningUnit(kb),
		Thinking:   thinking.NewThinkingEngine(kb),
		Decision:   decision.NewEngine(),
		Language:   language.NewEngine(kb),
		Execution:  execution.NewExecutionCore(),
		WebSearch:  websearch.NewEngine(nil),
		Perception: perception.UserInputPerception{},
	}
}
