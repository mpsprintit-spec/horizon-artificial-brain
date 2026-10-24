package core

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

// InitializeNeuralComposition is explicit opt-in. Each migration has its own
// durable pre-change backup; existing learned units are retained. The facade
// never opens an external execution path as a consequence of initialization.
func (h *HorizonEngine) InitializeNeuralComposition(c knowledge.CompositionConfig, backupPath string, at time.Time) error {
	if h == nil || h.supervisor == nil {
		return errors.New("horizon engine is unavailable")
	}
	if err := c.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(backupPath) == "" {
		return errors.New("composition requires a new rollback backup path")
	}
	return h.supervisor.WithMutation(func(rt *runtime.BrainRuntime) error {
		plan, err := rt.PlanNeuralComposition(c)
		if errors.Is(err, runtime.ErrCompositionRequiresWrittenMigration) {
			written, err := rt.PlanNeuralWrittenGrounding()
			if err != nil {
				return err
			}
			if err := rt.ApplyNeuralWrittenGrounding(written, backupPath+".written", at); err != nil {
				return err
			}
			plan, err = rt.PlanNeuralComposition(c)
			if err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		return rt.ApplyNeuralComposition(plan, backupPath, at)
	})
}

func (h *HorizonEngine) LearnComposition(e knowledge.CompositionExposure, at time.Time) (uint64, error) {
	if h == nil || h.supervisor == nil {
		return 0, errors.New("horizon engine is unavailable")
	}
	var sequence uint64
	err := h.supervisor.WithMutation(func(rt *runtime.BrainRuntime) error {
		var err error
		sequence, err = rt.LearnComposition(e, at)
		return err
	})
	return sequence, err
}

func (h *HorizonEngine) ObserveCompositionScene(id, source string, scene knowledge.CompositionScene) (runtime.CognitiveInterpretation, error) {
	if h == nil || h.supervisor == nil {
		return runtime.CognitiveInterpretation{}, errors.New("horizon engine is unavailable")
	}
	var output runtime.CognitiveInterpretation
	err := h.supervisor.WithMutation(func(rt *runtime.BrainRuntime) error {
		var err error
		output, err = rt.ObserveCompositionScene(id, source, scene)
		return err
	})
	return output, err
}

func (h *HorizonEngine) CompositionStatus() (*runtime.CompositionContinuation, error) {
	if h == nil || h.supervisor == nil {
		return nil, errors.New("horizon engine is unavailable")
	}
	var output *runtime.CompositionContinuation
	err := h.supervisor.With(func(rt *runtime.BrainRuntime) error { var err error; output, err = rt.CompositionStatus(); return err })
	return output, err
}

// RequestCompositionAction is explicit external I/O, never called by Pulse,
// learning, thought or replay. Its owner lease prevents a scene/Brain handover
// between validation and bounded adapter dispatch. The adapter must honor expiry.
func (h *HorizonEngine) RequestCompositionAction(proposal knowledge.CompositionReadout, actuatorID, requestID, approver string, expiry time.Time) error {
	if h == nil || h.supervisor == nil || h.Gateway == nil {
		return errors.New("horizon execution gateway is unavailable")
	}
	if err := (&runtime.NumericMotorCommand{ActuatorID: actuatorID, Values: proposal.Motor}).Validate(); err != nil {
		return err
	}
	if len(proposal.Goal) != 5 || len(proposal.Coordinates) != 6 {
		return errors.New("invalid composition proposal coordinates")
	}
	proposal, err := freezeCompositionProposal(proposal)
	if err != nil {
		return err
	}
	return h.supervisor.WithMutation(func(rt *runtime.BrainRuntime) error {
		now := time.Now().UTC()
		if err := rt.ValidateCompositionAction(proposal, now); err != nil {
			return err
		}
		recommendation := runtime.Recommendation{RequestID: requestID, BrainIdentity: runtime.BrainIdentity, Intent: "numeric-motor", Motor: &runtime.NumericMotorCommand{ActuatorID: actuatorID, Values: proposal.Motor}, Confidence: proposal.Confidence, RiskLevel: runtime.RiskMedium, CreatedAt: now}
		assessment, err := h.Safety.Assess(recommendation)
		if err != nil {
			return err
		}
		authorization, err := h.Safety.Authorize(recommendation, assessment, approver)
		if err != nil {
			return err
		}
		request, err := h.Safety.BuildExecutionRequest(recommendation, assessment, authorization, expiry)
		if err != nil {
			return err
		}
		if err := rt.BindCompositionAction(proposal, requestID, now); err != nil {
			return err
		}
		return h.Gateway.Execute(request)
	})
}

func freezeCompositionProposal(p knowledge.CompositionReadout) (knowledge.CompositionReadout, error) {
	var out knowledge.CompositionReadout
	data, err := json.Marshal(p)
	if err == nil {
		err = json.Unmarshal(data, &out)
	}
	return out, err
}
