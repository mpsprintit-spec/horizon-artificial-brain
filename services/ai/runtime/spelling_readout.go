package runtime

import (
	"encoding/json"
	"errors"
	"slices"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

const EventTypeSpellingExposure = "spelling_exposure"

// LearnSpelling records paired numerical I/O supervision before changing the
// shared Brain's synapses. It grants no semantic truth or action permission.
func (r *BrainRuntime) LearnSpelling(exposure knowledge.SpellingExposure, at time.Time) (uint64, error) {
	if r == nil || r.brain == nil {
		return 0, errors.New("brain runtime is not initialized")
	}
	if err := exposure.Validate(); err != nil {
		return 0, err
	}
	exposure.Population, exposure.Scalars = slices.Clone(exposure.Population), slices.Clone(exposure.Scalars)
	// JSON freezing follows the same pre-append boundary as other typed commands.
	packet, err := json.Marshal(exposure)
	if err != nil {
		return 0, err
	}
	if err := json.Unmarshal(packet, &exposure); err != nil {
		return 0, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ensureWritableLocked(); err != nil {
		return r.seq, err
	}
	if at.IsZero() {
		at = r.nowLocked()
	}
	at = at.UTC()
	if err := r.brain.ValidateSpellingExposure(exposure); err != nil {
		return r.seq, err
	}
	err = r.runJournalCommandLocked(LoggedEvent{Type: EventTypeSpellingExposure, Timestamp: at, SpellingExposure: &exposure}, func() error {
		if err := r.brain.LearnSpellingAt(exposure, at); err != nil {
			return err
		}
		r.seq++
		return nil
	})
	return r.seq, err
}

// ReadLearnedSpelling is an owner-guarded, read-only numerical I/O boundary.
// It must be used inside the same supervisor lease as selection when a caller
// requires the spelling and native forecast from one committed Brain revision.
func (r *BrainRuntime) ReadLearnedSpelling(population []knowledge.NodeID) (knowledge.SpellingReadout, error) {
	if r == nil || r.brain == nil {
		return knowledge.SpellingReadout{}, errors.New("brain runtime is not initialized")
	}
	if len(population) > knowledge.MaxSpellingPopulation {
		return knowledge.SpellingReadout{}, errors.New("spelling population exceeds readout budget")
	}
	population = slices.Clone(population)
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ensureWritableLocked(); err != nil {
		return knowledge.SpellingReadout{}, err
	}
	return r.brain.ReadLearnedSpelling(population)
}
