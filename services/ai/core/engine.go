package core

import (
	"github.com/project-horizon/horizon-core/services/ai/decision"
	"github.com/project-horizon/horizon-core/services/ai/execution"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/language"
	"github.com/project-horizon/horizon-core/services/ai/learning"
	"github.com/project-horizon/horizon-core/services/ai/perception"
	"github.com/project-horizon/horizon-core/services/ai/runtime"
	"github.com/project-horizon/horizon-core/services/ai/thinking"
	"github.com/project-horizon/horizon-core/services/ai/websearch"
)

type HorizonEngine struct {
	// Runtime is the official neural runtime boundary. The fields below remain
	// available during migration, but they all operate on the same Brain.
	Runtime    *runtime.BrainRuntime
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
	brain := knowledge.NewBrain()
	rt := runtime.NewBrainRuntime(brain)
	return &HorizonEngine{
		Runtime:    rt,
		Knowledge:  brain,
		Learning:   learning.NewLearningUnit(brain),
		Thinking:   thinking.NewThinkingEngine(brain),
		Decision:   decision.NewEngine(),
		Language:   language.NewEngine(brain),
		Execution:  execution.NewExecutionCore(),
		WebSearch:  websearch.NewEngine(nil),
		Perception: perception.UserInputPerception{},
	}
}
