package activation

import (
	"errors"
	"strings"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// ContextState is bounded continuation state for the same neural substrate.
// Only actual observed stimulus populations enter it, never predictions.
type ContextState struct {
	EpisodeID string                    `json:"episode_id,omitempty"`
	Ordinal   uint64                    `json:"ordinal"`
	CueStart  uint64                    `json:"cue_start"`
	Frames    []knowledge.ContextSample `json:"frames,omitempty"`
}

func (s *ContextState) Clone() *ContextState {
	if s == nil {
		return nil
	}
	out := *s
	out.Frames = append([]knowledge.ContextSample(nil), s.Frames...)
	for i := range out.Frames {
		out.Frames[i].Population = append([]knowledge.NodeID(nil), s.Frames[i].Population...)
	}
	return &out
}

func (s *ContextState) before(now time.Time, excludeCue bool) []knowledge.ContextSample {
	if s == nil {
		return nil
	}
	var frames []knowledge.ContextSample
	for _, frame := range s.Frames {
		age := now.Sub(frame.At)
		if age < 0 || age > knowledge.ContextMaxAge || excludeCue && frame.Ordinal >= s.CueStart {
			continue
		}
		frames = append(frames, frame)
	}
	return frames
}

func (s *ContextState) prepare(episode string, reset bool, now time.Time) (*ContextState, bool) {
	if s == nil {
		return &ContextState{EpisodeID: episode}, reset
	}
	out := s.Clone()
	changed := reset || episode != s.EpisodeID || len(s.Frames) > 0 && s.Frames[len(s.Frames)-1].At.After(now)
	if changed {
		out = &ContextState{EpisodeID: episode}
	} else {
		out.Frames = out.before(now, false)
	}
	return out, changed
}

func (s *ContextState) observe(trace *knowledge.ObservationTrace, now time.Time) {
	s.CueStart = s.Ordinal + 1
	for _, frame := range trace.Stimulus {
		s.Ordinal++
		// Do not retain a partial population when it exceeds the context budget.
		if len(frame) > knowledge.ContextMaxMembers {
			continue
		}
		s.Frames = append(s.Frames, knowledge.ContextSample{Population: append([]knowledge.NodeID(nil), frame...), At: now.UTC(), Ordinal: s.Ordinal})
		members := 0
		for _, sample := range s.Frames {
			members += len(sample.Population)
		}
		for len(s.Frames) > knowledge.ContextMaxFrames || members > knowledge.ContextMaxMembers {
			members -= len(s.Frames[0].Population)
			s.Frames = s.Frames[1:]
		}
		s.Frames = append([]knowledge.ContextSample(nil), s.Frames...)
	}
}

func (s *ContextState) Validate(known func(knowledge.NodeID) bool) error {
	if s == nil {
		return nil
	}
	if len(s.Frames) > knowledge.ContextMaxFrames || len(s.Frames) > 0 && s.CueStart == 0 || s.Ordinal >= 1<<63 || s.CueStart > s.Ordinal+1 || s.EpisodeID != "" && strings.TrimSpace(s.EpisodeID) == "" {
		return errors.New("invalid recurrent context bounds or episode")
	}
	members := 0
	var ordinal uint64
	var at time.Time
	for _, frame := range s.Frames {
		members += len(frame.Population)
		if members > knowledge.ContextMaxMembers {
			return errors.New("recurrent context member budget exceeded")
		}
		if frame.At.IsZero() || frame.Ordinal <= ordinal || frame.Ordinal > s.Ordinal || frame.At.Before(at) {
			return errors.New("invalid recurrent context chronology")
		}
		if err := (&knowledge.ObservationTrace{Stimulus: [][]knowledge.NodeID{frame.Population}}).Validate(known); err != nil {
			return err
		}
		ordinal, at = frame.Ordinal, frame.At
	}
	return nil
}

func (e *Engine) applyRecurrentPatternPrediction(prediction, confidence map[knowledge.NodeID]float64, trace *knowledge.ObservationTrace, context *ContextState, now time.Time, excludeCue bool) (map[knowledge.NodeID]float64, map[knowledge.NodeID]float64) {
	if context == nil {
		return e.applyOrderedPatternPrediction(prediction, confidence, trace)
	}
	channels := e.Memory.Patterns.PredictRecurrent(trace, context.before(now, excludeCue), now)
	return e.applyPatternChannels(prediction, confidence, channels)
}

// RecurrentEpisode returns detached routing metadata for the runtime's causal
// predecessor reset. It does not expose writable engine state.
func (e *Engine) RecurrentEpisode() (string, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.contextState == nil {
		return "", false
	}
	return e.contextState.EpisodeID, true
}
