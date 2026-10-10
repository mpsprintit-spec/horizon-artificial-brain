package core

import (
	"errors"
	"fmt"
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

// Pulse is the canonical external perception-to-cognition entry point.
// Perception only normalizes input; the CognitiveOrchestrator owns grounding,
// runtime processing, and interpretation. Pulse deliberately does not create
// an Experience, so an observation cannot become trusted learning merely by
// entering through the facade.
func (h *HorizonEngine) Pulse(input string) (runtime.CognitiveInterpretation, error) {
	if h == nil || h.Perception == nil || h.Orchestrator == nil || h.Runtime == nil {
		return runtime.CognitiveInterpretation{}, errors.New("horizon cognitive pipeline is unavailable")
	}
	signals, err := h.Perception.Perceive(input)
	if err != nil {
		return runtime.CognitiveInterpretation{}, err
	}
	interpretations, err := h.PulseSignals(signals)
	if err != nil {
		return runtime.CognitiveInterpretation{}, err
	}
	if len(interpretations) == 0 {
		return runtime.CognitiveInterpretation{}, errors.New("perception produced no signals")
	}
	return interpretations[len(interpretations)-1], nil
}

// PulseSignals feeds every perception signal through the canonical
// observation boundary. Each signal retains its own source, timestamp and
// modality, so multimodal input is not collapsed into text before grounding.
// The returned interpretations are ordered exactly as the input signals.
func (h *HorizonEngine) PulseSignals(signals []perception.PerceptionSignal) ([]runtime.CognitiveInterpretation, error) {
	if h == nil || h.Orchestrator == nil || h.Runtime == nil {
		return nil, errors.New("horizon cognitive pipeline is unavailable")
	}
	if len(signals) == 0 {
		return nil, errors.New("perception produced no signals")
	}

	interpretations := make([]runtime.CognitiveInterpretation, 0, len(signals))
	for index, signal := range signals {
		observedAt := signal.ObservedAt
		if observedAt.IsZero() {
			observedAt = time.Now().UTC()
		}
		modality := strings.TrimSpace(signal.Modality)
		if modality == "" {
			modality = "perception"
		}

		event := runtime.Event{
			ID:        fmt.Sprintf("pulse-%s-%d", observedAt.Format("20060102T150405.000000000Z0700"), index),
			Stimulus:  append([]string(nil), signal.Tokens...),
			Source:    signal.Source,
			Modality:  modality,
			Cycles:    1,
			Timestamp: observedAt,
		}
		observation := runtime.ObservationInput{
			Source:   signal.Source,
			Modality: modality,
			Tokens:   append([]string(nil), signal.Tokens...),
		}

		interpretation, _, err := h.Orchestrator.ProcessObservation(event, observation, nil)
		if err != nil {
			return nil, err
		}
		interpretations = append(interpretations, interpretation)
	}
	return interpretations, nil
}

func NewHorizonEngine() *HorizonEngine {
	brain := knowledge.NewBootstrapBrain()
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
	if h == nil || h.Gateway == nil {
		return errors.New("horizon execution gateway is unavailable")
	}
	if recommendation.BrainIdentity == "" {
		recommendation.BrainIdentity = runtime.BrainIdentity
	}
	assessment, err := h.Safety.Assess(recommendation)
	if err != nil {
		return err
	}
	authorization, err := h.Safety.Authorize(recommendation, assessment, strings.TrimSpace(approver))
	if err != nil {
		return err
	}
	request, err := h.Safety.BuildExecutionRequest(recommendation, assessment, authorization, expiry)
	if err != nil {
		return err
	}
	return h.Gateway.Execute(request)
}
