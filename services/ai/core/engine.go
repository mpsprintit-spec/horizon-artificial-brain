package core

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/decision"
	"github.com/project-horizon/horizon-core/services/ai/execution"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/perception"
	"github.com/project-horizon/horizon-core/services/ai/runtime"
	"github.com/project-horizon/horizon-core/services/ai/websearch"
)

type HorizonEngine struct {
	// The managed facade publishes only the owner handle and detached values.
	// It never caches a raw Brain, runtime, or legacy cognitive engine.
	supervisor *runtime.RuntimeSupervisor
	Decision *decision.Engine

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
	if h == nil || h.Perception == nil || h.supervisor == nil {
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
	if h == nil || h.supervisor == nil {
		return nil, errors.New("horizon cognitive pipeline is unavailable")
	}
	if len(signals) == 0 {
		return nil, errors.New("perception produced no signals")
	}
	var outputs []runtime.CognitiveInterpretation
	err := h.supervisor.WithMutation(func(active *runtime.BrainRuntime) error {
		var err error
		outputs, err = h.pulseSignalsWith(runtime.NewCognitiveOrchestrator(active), signals)
		return err
	})
	return outputs, err
}

// pulseSignalsWith assumes the caller owns a supervisor mutation lease when
// managed. The same active runtime serves the entire multimodal batch.
func (h *HorizonEngine) pulseSignalsWith(orchestrator *runtime.CognitiveOrchestrator, signals []perception.PerceptionSignal) ([]runtime.CognitiveInterpretation, error) {
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
	EpisodeID: signal.SessionID, ResetContext: signal.ResetContext,
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

		interpretation, _, err := orchestrator.ProcessObservation(event, observation, nil)
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
	supervisor, err := runtime.NewRuntimeSupervisor(rt)
	if err != nil { return nil }
	core := execution.NewExecutionCore()
	return &HorizonEngine{
		supervisor: supervisor, Decision: decision.NewEngine(),
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
