package runtime

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func validateCheckpointContext(snapshot BrainSnapshot) error {
	state := snapshot.Activation.ContextState
	if state == nil {
		return nil
	}
	if snapshot.SchemaVersion < recurrentBrainSnapshotSchemaVersion {
		return errors.New("legacy checkpoint cannot carry recurrent context")
	}
	var raw struct {
		Neurons []struct {
			ID knowledge.NodeID `json:"id"`
		} `json:"neural_units"`
	}
	if err := json.Unmarshal(snapshot.Brain, &raw); err != nil {
		return err
	}
	known := make(map[knowledge.NodeID]bool, len(raw.Neurons))
	for _, neuron := range raw.Neurons {
		known[neuron.ID] = true
	}
	return state.Validate(func(id knowledge.NodeID) bool { return known[id] })
}

func validateEpisodeControls(event Event, dynamics knowledge.DynamicsEncoding) error {
	if event.SensorObservation != nil && event.ResetContext {
		return errors.New("sensor episode reset requires a changed session identity")
	}
	if (event.ResetContext || event.EpisodeID != "" || event.SensorObservation != nil && event.SensorObservation.SessionID != "") && !dynamics.HasRecurrentContext() {
		return errors.New("episode controls require recurrent context dynamics")
	}
	if event.EpisodeID != "" && strings.TrimSpace(event.EpisodeID) == "" {
		return errors.New("episode identity is blank")
	}
	if dynamics.HasRecurrentContext() {
		if event.Observation != nil && event.Observation.SessionID != event.EpisodeID {
			return errors.New("observation and neural episode identities disagree")
		}
		if event.SensorObservation != nil && event.SensorObservation.SessionID != event.EpisodeID {
			return errors.New("sensor and neural episode identities disagree")
		}
	}
	return nil
}

// Reset the temporal predecessor at the same durable boundary as the neural
// context. Otherwise the observation learner would create an accidental edge
// between two explicitly separate episodes after activation had been reset.
func (r *BrainRuntime) resetTemporalContextLocked(event Event) {
	if !r.dynamicsEncoding.HasRecurrentContext() {
		return
	}
	episode, present := r.activation.RecurrentEpisode()
	if event.ResetContext || present && episode != event.EpisodeID || !present && event.EpisodeID != "" || !r.lastObservationAt.IsZero() && event.Timestamp.Before(r.lastObservationAt) {
		r.lastObservationPopulation = nil
		r.lastObservationAt = time.Time{}
	}
}
