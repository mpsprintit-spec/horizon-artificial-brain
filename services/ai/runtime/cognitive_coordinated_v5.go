package runtime

import (
	"errors"
	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// Explicit opt-in API. It never changes the default runtime decision route.
func (r *BrainRuntime) PreviewCoordinatedV5(trace []k.CognitiveInput, drop k.Phase4DropV5) (k.CoordinatedPredictionV5, error) {
	if r == nil || r.brain == nil {
		return k.CoordinatedPredictionV5{}, errors.New("runtime unavailable")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if e := r.ensureWritableLocked(); e != nil {
		return k.CoordinatedPredictionV5{}, e
	}
	return r.brain.PreviewCoordinatedV5(trace, drop)
}
