package core

import (
	"errors"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

// PulseWithContinuationPreview retains the native Answer, then uses only that
// already-selected population as the seed of a separate, read-only hypothesis.
// One owner lease covers the real pulse and its bounded learned I/O projection.
func (h *HorizonEngine) PulseWithContinuationPreview(input string, maxSteps int) (runtime.CognitiveInterpretation, runtime.ContinuationReadout, error) {
	out := runtime.ContinuationReadout{BrainIdentity: runtime.BrainIdentity, Encoding: runtime.ContinuationEncodingV1, Hypothetical: true, Status: runtime.ContinuationUnknown}
	if h == nil || h.Perception == nil || h.supervisor == nil {
		return runtime.CognitiveInterpretation{}, out, errors.New("horizon cognitive pipeline is unavailable")
	}
	if maxSteps < 1 || maxSteps > runtime.MaxContinuationSteps {
		return runtime.CognitiveInterpretation{}, out, errors.New("continuation step budget is invalid")
	}
	signals, err := h.Perception.Perceive(input)
	if err != nil {
		return runtime.CognitiveInterpretation{}, out, err
	}
	if len(signals) == 0 {
		return runtime.CognitiveInterpretation{}, out, errors.New("perception produced no signals")
	}
	var result runtime.CognitiveInterpretation
	err = h.supervisor.WithMutation(func(active *runtime.BrainRuntime) error {
		interpretations, err := h.pulseSignalsWith(runtime.NewCognitiveOrchestrator(active), signals)
		if err != nil {
			return err
		}
		result = interpretations[len(interpretations)-1]
		out.Sequence, out.At = result.State.Sequence, result.State.Timestamp
		if result.Answer.Status != runtime.AnswerNeuralForecast {
			return nil
		}
		if result.Answer.Confidence <= runtime.ContinuationConfidenceGate {
			out.Status = runtime.ContinuationUncertain
			return nil
		}
		seed := &knowledge.ObservationTrace{Stimulus: [][]knowledge.NodeID{result.Answer.NodeIDs}}
		out, err = active.ReadLearnedContinuation(seed, maxSteps, result.State.Timestamp)
		return err
	})
	return result, out, err
}
