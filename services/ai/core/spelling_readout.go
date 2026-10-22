package core

import (
	"errors"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

// PulseWithSpelling decodes only the already-selected numeric forecast under
// the same owner lease. Spelling is paired I/O learned on this Brain, not proof
// of a correct answer. General activity and diagnostic labels never choose it.
func (h *HorizonEngine) PulseWithSpelling(input string) (runtime.CognitiveInterpretation, knowledge.SpellingReadout, error) {
	if h == nil || h.Perception == nil || h.supervisor == nil {
		return runtime.CognitiveInterpretation{}, knowledge.SpellingReadout{}, errors.New("horizon cognitive pipeline is unavailable")
	}
	signals, err := h.Perception.Perceive(input)
	if err != nil {
		return runtime.CognitiveInterpretation{}, knowledge.SpellingReadout{}, err
	}
	if len(signals) == 0 {
		return runtime.CognitiveInterpretation{}, knowledge.SpellingReadout{}, errors.New("perception produced no signals")
	}
	var result runtime.CognitiveInterpretation
	spelling := knowledge.SpellingReadout{Status: knowledge.SpellingUntrained}
	err = h.supervisor.WithMutation(func(active *runtime.BrainRuntime) error {
		interpretations, err := h.pulseSignalsWith(runtime.NewCognitiveOrchestrator(active), signals)
		if err != nil {
			return err
		}
		result = interpretations[len(interpretations)-1]
		if result.Answer.Status != runtime.AnswerNeuralForecast {
			return nil
		}
		spelling, err = active.ReadLearnedSpelling(result.Answer.NodeIDs)
		return err
	})
	return result, spelling, err
}
