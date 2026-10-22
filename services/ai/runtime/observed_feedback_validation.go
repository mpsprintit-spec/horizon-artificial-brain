package runtime

import (
	"encoding/json"
	"errors"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func validateObservationPrediction(snapshot BrainSnapshot) error {
	state := snapshot.Activation.ObservationPrediction
	if state == nil {
		return nil
	}
	if snapshot.SchemaVersion < observedFeedbackBrainSnapshotSchemaVersion {
		return errors.New("legacy checkpoint cannot carry observed prediction provenance")
	}
	var raw struct {
		Neurons []struct {
			ID knowledge.NodeID `json:"id"`
		} `json:"neural_units"`
		Patterns []struct {
			ID knowledge.PatternID `json:"id"`
		} `json:"temporal_patterns"`
	}
	if err := json.Unmarshal(snapshot.Brain, &raw); err != nil {
		return err
	}
	known := map[knowledge.NodeID]bool{}
	for _, n := range raw.Neurons {
		known[n.ID] = true
	}
	patterns := map[knowledge.PatternID]bool{}
	for _, p := range raw.Patterns {
		patterns[p.ID] = true
	}
	if err := state.Validate(func(id knowledge.NodeID) bool { return known[id] }, patterns); err != nil {
		return err
	}
	trace := snapshot.Activation.LastTrace
	if trace == nil {
		return errors.New("observed prediction is missing its source trace")
	}
	source := map[knowledge.NodeID]bool{}
	for _, frame := range trace.Stimulus {
		for _, id := range frame {
			source[id] = true
		}
	}
	if len(source) != len(state.Source) {
		return errors.New("observed prediction source/trace mismatch")
	}
	for id := range source {
		if state.Source[id] != 1 {
			return errors.New("observed prediction source is not direct evidence")
		}
	}
	if context := snapshot.Activation.ContextState; context != nil && len(context.Frames) > 0 && state.At.Before(context.Frames[len(context.Frames)-1].At) {
		return errors.New("observed prediction precedes its context")
	}
	context := snapshot.Activation.ContextState
	if context == nil || context.EpisodeID != state.EpisodeID {
		return errors.New("observed prediction episode mismatch")
	}
	if len(context.Frames) > 0 {
		last := context.Frames[len(context.Frames)-1]
		if last.Ordinal >= context.CueStart && !last.At.Equal(state.At) {
			return errors.New("observed prediction timestamp differs from its cue")
		}
	}
	return nil
}
