package core

import (
	"errors"
	"strings"
	"time"

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
	Runtime   *runtime.BrainRuntime
	Knowledge *knowledge.KnowledgeBase
	Learning  *learning.LearningUnit
	Thinking  *thinking.ThinkingEngine
	Decision  *decision.Engine
	Language  *language.Engine

	// Orchestrator is the canonical integration boundary for the neural
	// perception -> cognition -> interpretation cycle.
	Orchestrator *runtime.CognitiveOrchestrator

	// Gateway is the only execution surface exposed by HorizonEngine. The
	// underlying ExecutionCore is intentionally private to this facade.
	Gateway *execution.ExecutionGateway

	WebSearch        *websearch.Engine
	WebSearchEnabled bool
	Perception       perception.PerceptionLayer

	Safety runtime.SafetyBoundary
}

func NewHorizonEngine() *HorizonEngine {
	brain := knowledge.NewBrain()
	rt := runtime.NewBrainRuntime(brain)
	core := execution.NewExecutionCore()
	return &HorizonEngine{
		Runtime: rt, Knowledge: brain, Learning: learning.NewLearningUnit(brain),
		Thinking: thinking.NewThinkingEngine(brain), Decision: decision.NewEngine(),
		Language: language.NewEngine(brain), Orchestrator: runtime.NewCognitiveOrchestrator(rt),
		Gateway: execution.NewExecutionGateway(core),
		WebSearch: websearch.NewEngine(nil), WebSearchEnabled: false,
		Perception: perception.UserInputPerception{}, Safety: runtime.SafetyBoundary{},
	}
}

// RequestAction is the facade-level action boundary. Callers cannot bypass the
// safety assessment/authorization stages or reach ExecutionCore directly.
func (h *HorizonEngine) RequestAction(recommendation runtime.Recommendation, approver string, expiry time.Time) error {
	if h == nil || h.Gateway == nil { return errors.New("horizon execution gateway is unavailable") }
	if recommendation.BrainIdentity == "" { recommendation.BrainIdentity = runtime.BrainIdentity }
	assessment, err := h.Safety.Assess(recommendation)
	if err != nil { return err }
	authorization, err := h.Safety.Authorize(recommendation, assessment, strings.TrimSpace(approver))
	if err != nil { return err }
	request, err := h.Safety.BuildExecutionRequest(recommendation, assessment, authorization, expiry)
	if err != nil { return err }
	return h.Gateway.Execute(request)
}
