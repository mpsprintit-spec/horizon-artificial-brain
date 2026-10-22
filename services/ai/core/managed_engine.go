package core

import (
	"errors"
	"fmt"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/activation"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

// WithActiveBrain executes a synchronous, lock-scoped visit to the currently
// published canonical brain. The callback must not retain a brain/node
// reference or call the supervisor again. Legacy public Knowledge/Runtime
// aliases are not retargeted during recovery; only this managed path is safe.
func (h *HorizonEngine) WithActiveBrain(fn func(*knowledge.Brain) error) error {
	if h == nil { return errors.New("horizon engine is unavailable") }
	if fn == nil { return errors.New("active brain callback is nil") }
	if h.Supervisor == nil {
		if h.Knowledge == nil { return errors.New("canonical brain is unavailable") }
		return fn(h.Knowledge)
	}
	return h.Supervisor.WithMutation(func(rt *runtime.BrainRuntime) error {
		return rt.WithBrainView(fn)
	})
}

// PulseWithLabels is the managed CLI boundary. Cognition and the associated
// label resolution happen within a SINGLE mutation lease; the labels therefore
// cannot accidentally come from a different runtime after a handover.
func (h *HorizonEngine) PulseWithLabels(input string) (runtime.CognitiveInterpretation, []string, error) {
	if h == nil || h.Perception == nil {
		return runtime.CognitiveInterpretation{}, nil, errors.New("horizon cognitive pipeline is unavailable")
	}
	signals, err := h.Perception.Perceive(input)
	if err != nil { return runtime.CognitiveInterpretation{}, nil, err }
	if len(signals) == 0 { return runtime.CognitiveInterpretation{}, nil, errors.New("perception produced no signals") }

	var result runtime.CognitiveInterpretation
	var labels []string
	resolve := func(brain *knowledge.Brain) {
		labels = make([]string, 0, len(result.Interpretation.RankedNodeIDs))
		seen := make(map[string]struct{})
		for _, id := range result.Interpretation.RankedNodeIDs {
			node := brain.Registry.GetByID(id)
			if node == nil { continue }
			label := brain.SurfaceForNode(id, "language")
			if label == "" { label = fmt.Sprintf("node:%d", node.ID) }
			if _, ok := seen[label]; ok { continue }
			seen[label] = struct{}{}
			labels = append(labels, label)
		}
	}
	if h.Supervisor != nil {
		err = h.Supervisor.WithMutation(func(rt *runtime.BrainRuntime) error {
			interpretations, err := h.pulseSignalsWith(runtime.NewCognitiveOrchestrator(rt), signals)
			if err != nil { return err }
			result = interpretations[len(interpretations)-1]
			return rt.WithBrainView(func(brain *knowledge.Brain) error {
				resolve(brain)
				return nil
			})
		})
	} else {
		interpretations, processErr := h.PulseSignals(signals)
		if processErr != nil { return runtime.CognitiveInterpretation{}, nil, processErr }
		result = interpretations[len(interpretations)-1]
		err = h.WithActiveBrain(func(brain *knowledge.Brain) error { resolve(brain); return nil })
	}
	if err != nil { return runtime.CognitiveInterpretation{}, nil, err }
	return result, labels, nil
}

// ThinkActive advances the active runtime through its supervisor. The facade
// never caches a replacement runtime between autonomous cognitive cycles.
func (h *HorizonEngine) ThinkActive(cycles int) (runtime.CognitiveOutput, error) {
	if h == nil { return runtime.CognitiveOutput{}, errors.New("horizon engine is unavailable") }
	if h.Supervisor == nil {
		if h.Runtime == nil { return runtime.CognitiveOutput{}, errors.New("brain runtime is unavailable") }
		return h.Runtime.CognitiveThink(cycles)
	}
	var output runtime.CognitiveOutput
	err := h.Supervisor.WithMutation(func(rt *runtime.BrainRuntime) error {
		var err error
		output, err = rt.CognitiveThink(cycles)
		return err
	})
	return output, err
}

// SaveActiveBrain and LoadActiveBrain preserve the legacy CLI brain-memory
// format while routing I/O to whichever runtime the supervisor currently owns.
// These are manual brain-file operations, not transactional checkpoint replay.
func (h *HorizonEngine) SaveActiveBrain(path string) error {
	if h == nil { return errors.New("horizon engine is unavailable") }
	if h.Supervisor == nil {
		if h.Runtime == nil { return errors.New("brain runtime is unavailable") }
		return h.Runtime.SaveBrain(path)
	}
	return h.Supervisor.WithMutation(func(rt *runtime.BrainRuntime) error { return rt.SaveBrain(path) })
}

func (h *HorizonEngine) LoadActiveBrain(path string) error {
	if h == nil { return errors.New("horizon engine is unavailable") }
	if h.Supervisor == nil {
		if h.Runtime == nil { return errors.New("brain runtime is unavailable") }
		return h.Runtime.LoadBrain(path)
	}
	return h.Supervisor.WithMutation(func(rt *runtime.BrainRuntime) error { return rt.LoadBrain(path) })
}

func (h *HorizonEngine) ActivateActiveBootstrap() (activation.Result, error) {
	if h == nil { return activation.Result{}, errors.New("horizon engine is unavailable") }
	if h.Supervisor == nil {
		if h.Runtime == nil { return activation.Result{}, errors.New("brain runtime is unavailable") }
		return h.Runtime.ActivateBootstrap(horizonNow())
	}
	var result activation.Result
	err := h.Supervisor.WithMutation(func(rt *runtime.BrainRuntime) error {
		var err error
		result, err = rt.ActivateBootstrap(horizonNow())
		return err
	})
	return result, err
}

// This clock is for legacy CLI bootstrap activation, not event-time recovery.
func horizonNow() time.Time { return time.Now().UTC() }

