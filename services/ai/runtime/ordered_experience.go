package runtime

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

func validateExperienceStructure(experience learning.Experience) error {
	for _, pair := range []struct {
		words  []string
		frames [][]knowledge.NodeID
	}{{experience.Sequence, experience.NeuralSequence}, {experience.ContextSequence, experience.NeuralContext}, {experience.DataSequence, experience.NeuralData}} {
		if len(pair.words) > 0 && len(pair.frames) > 0 {
			return errors.New("experience stream cannot mix lexical and neural frames")
		}
		for _, word := range pair.words {
			if strings.TrimSpace(word) == "" {
				return errors.New("experience contains an empty surface")
			}
		}
		if err := (&knowledge.ObservationTrace{Stimulus: pair.frames}).Validate(nil); err != nil {
			return err
		}
	}
	return nil
}

func (r *BrainRuntime) validateExperienceLocked(experience learning.Experience) error {
	if err := validateExperienceStructure(experience); err != nil {
		return err
	}
	if r.dynamicsEncoding == knowledge.LegacyDynamics && experience.HasOrderedFields() {
		return errors.New("ordered experience requires ordered neural dynamics")
	}
	trace := &knowledge.ObservationTrace{Stimulus: experience.NeuralSequence, Context: experience.NeuralContext, Data: experience.NeuralData}
	return trace.Validate(func(id knowledge.NodeID) bool { return r.brain.Registry.GetByID(id) != nil })
}

func freezeExperience(experience learning.Experience) (learning.Experience, error) {
	data, err := json.Marshal(experience)
	if err != nil {
		return learning.Experience{}, err
	}
	var frozen learning.Experience
	if err := json.Unmarshal(data, &frozen); err != nil {
		return learning.Experience{}, err
	}
	return frozen, validateExperienceStructure(frozen)
}

func (r *BrainRuntime) validateNeuralEventLocked(event Event) error {
	if event.NeuralTrace != nil && !r.dynamicsEncoding.HasOrderedObservations() {
		return errors.New("neural trace requires ordered dynamics")
	}
	if err := validateEpisodeControls(event, r.dynamicsEncoding); err != nil {
		return err
	}
	return event.NeuralTrace.Validate(func(id knowledge.NodeID) bool { return r.brain.Registry.GetByID(id) != nil })
}
