package core

import (
	"errors"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/activation"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

// Supervisor returns the facade's stable owner handle. The handle resolves
// the active runtime for each lease; callers cannot replace the facade owner.
func (h *HorizonEngine) Supervisor() *runtime.RuntimeSupervisor {
	if h == nil {
		return nil
	}
	return h.supervisor
}

// Deprecated: a live mutable Brain callback is unsupported on the managed
// facade. Kept to return an explicit migration error to old callers; the
// callback is never invoked. Use InspectActiveBrain and typed runtime commands.
func (h *HorizonEngine) WithActiveBrain(fn func(*knowledge.Brain) error) error {
	if h == nil || h.supervisor == nil {
		return errors.New("horizon engine is unavailable")
	}
	if fn == nil {
		return errors.New("active brain callback is nil")
	}
	return h.supervisor.WithMutation(func(rt *runtime.BrainRuntime) error { return rt.WithBrainView(fn) })
}

// PulseWithLabels holds one owner lease across cognition and label lookup.
func (h *HorizonEngine) PulseWithLabels(input string) (runtime.CognitiveInterpretation, []string, error) {
	if h == nil || h.Perception == nil || h.supervisor == nil {
		return runtime.CognitiveInterpretation{}, nil, errors.New("horizon cognitive pipeline is unavailable")
	}
	signals, err := h.Perception.Perceive(input)
	if err != nil {
		return runtime.CognitiveInterpretation{}, nil, err
	}
	if len(signals) == 0 {
		return runtime.CognitiveInterpretation{}, nil, errors.New("perception produced no signals")
	}
	var result runtime.CognitiveInterpretation
	var labels []string
	err = h.supervisor.WithMutation(func(active *runtime.BrainRuntime) error {
		interpretations, err := h.pulseSignalsWith(runtime.NewCognitiveOrchestrator(active), signals)
		if err != nil {
			return err
		}
		result = interpretations[len(interpretations)-1]
		labels, err = active.ResolveNodeLabels(result.Interpretation.RankedNodeIDs, "language")
		return err
	})
	return result, labels, err
}

func (h *HorizonEngine) ThinkActive(cycles int) (runtime.CognitiveOutput, error) {
	if h == nil || h.supervisor == nil {
		return runtime.CognitiveOutput{}, errors.New("horizon engine is unavailable")
	}
	var output runtime.CognitiveOutput
	err := h.supervisor.WithMutation(func(rt *runtime.BrainRuntime) error {
		var err error
		output, err = rt.CognitiveThink(cycles)
		return err
	})
	return output, err
}

func (h *HorizonEngine) SaveActiveBrain(path string) error {
	if h == nil || h.supervisor == nil {
		return errors.New("horizon engine is unavailable")
	}
	return h.supervisor.WithMutation(func(rt *runtime.BrainRuntime) error { return rt.SaveBrain(path) })
}
func (h *HorizonEngine) LoadActiveBrain(path string) error {
	if h == nil || h.supervisor == nil {
		return errors.New("horizon engine is unavailable")
	}
	return h.supervisor.WithMutation(func(rt *runtime.BrainRuntime) error { return rt.LoadBrain(path) })
}
func (h *HorizonEngine) ActivateActiveBootstrap() (activation.Result, error) {
	if h == nil || h.supervisor == nil {
		return activation.Result{}, errors.New("horizon engine is unavailable")
	}
	var result activation.Result
	err := h.supervisor.WithMutation(func(rt *runtime.BrainRuntime) error {
		var err error
		result, err = rt.ActivateBootstrap(time.Now().UTC())
		return err
	})
	return result, err
}
