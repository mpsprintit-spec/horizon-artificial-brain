package runtime

import (
	"errors"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"strings"
	"time"
)

const EventTypeCompositionPerception = "composition_perception"

type CompositionPerception struct {
	ID     string                     `json:"id"`
	Source string                     `json:"source"`
	Input  knowledge.CompositionInput `json:"input"`
}

func (p CompositionPerception) validate(at time.Time) error {
	if strings.TrimSpace(p.ID) == "" || len(p.ID) > 512 || strings.TrimSpace(p.Source) == "" || len(p.Source) > 256 || len(p.Input.Sequence) == 0 || len(p.Input.Words) != 1 || p.Input.Scene.ObservedAt.After(at) {
		return errors.New("invalid or future numerical composition perception")
	}
	return p.Input.Validate()
}

// Actual numerical perception updates continuation, not knowledge topology.
// Unknown or different-looking inputs cannot allocate neurons on this path.
func (r *BrainRuntime) ObserveCompositionInput(p CompositionPerception, at time.Time) (uint64, error) {
	if r == nil || r.brain == nil {
		return 0, errors.New("runtime unavailable")
	}
	f, e := freezeComposition(p)
	if e != nil {
		return 0, e
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if e = r.ensureWritableLocked(); e != nil {
		return r.seq, e
	}
	if at.IsZero() {
		at = r.nowLocked()
	}
	at = at.UTC()
	if e = f.validate(at); e != nil {
		return r.seq, e
	}
	if r.composition == nil {
		return r.seq, errors.New("composition disabled")
	}
	if r.composition.Config.Version != knowledge.NeuralCompositionSequenceV4 && !knowledge.IsNumericalCompositionVersion(r.composition.Config.Version) {
		return r.seq, errors.New("sequence perception requires a sequence contract")
	}
	if r.composition.SequenceObservedAt != nil && f.Input.Scene.ObservedAt.Before(*r.composition.SequenceObservedAt) {
		return r.seq, errors.New("sequence perception timestamp regressed")
	}
	if _, ok := r.processedEvents[f.ID]; ok {
		return r.seq, errors.New("duplicate composition perception")
	}
	// Preview validates the version/input boundary without learning or grounding.
	if _, e = r.brain.ReadComposition(r.composition.Config, f.Input, at); e != nil {
		return r.seq, e
	}
	e = r.runJournalCommandLocked(LoggedEvent{Type: EventTypeCompositionPerception, Timestamp: at, CompositionPerception: &f}, func() error {
		r.composition.Words = f.Input.Words
		r.composition.Sequence = f.Input.Sequence
		r.composition.RequiredPrecision = f.Input.RequiredPrecision
		r.composition.MeasurementError = f.Input.MeasurementError
		scene := f.Input.Scene
		r.composition.Scene = &scene
		sequenceAt := scene.ObservedAt
		r.composition.SequenceObservedAt = &sequenceAt
		r.composition.SequenceSource = f.Source
		r.composition.Unsupported = false
		r.seq++
		r.processedEvents[f.ID] = r.seq
		return nil
	})
	return r.seq, e
}
