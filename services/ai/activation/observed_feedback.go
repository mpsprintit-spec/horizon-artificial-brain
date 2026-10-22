package activation

import (
	"errors"
	"math"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// ObservationPrediction preserves one forecast made at a real observation.
// Thought may evolve lastPrediction, but cannot rewrite this evidence boundary.
// These are detached numeric continuation references to the existing substrate.
type ObservationPrediction struct {
	EpisodeID  string                          `json:"episode_id,omitempty"`
	At         time.Time                       `json:"observed_at"`
	Source     map[knowledge.NodeID]float64    `json:"source"`
	Prediction map[knowledge.NodeID]float64    `json:"prediction"`
	Patterns   map[knowledge.PatternID]float64 `json:"pattern_credit,omitempty"`
}

func (s *ObservationPrediction) Clone() *ObservationPrediction {
	if s == nil {
		return nil
	}
	out := &ObservationPrediction{EpisodeID: s.EpisodeID, At: s.At, Source: cloneState(s.Source), Prediction: cloneState(s.Prediction)}
	if s.Patterns != nil {
		out.Patterns = make(map[knowledge.PatternID]float64, len(s.Patterns))
		for id, q := range s.Patterns {
			out.Patterns[id] = q
		}
	}
	return out
}

func (s *ObservationPrediction) Validate(known func(knowledge.NodeID) bool, patterns map[knowledge.PatternID]bool) error {
	if s == nil {
		return nil
	}
	if s.At.IsZero() || len(s.Source) == 0 {
		return errors.New("invalid observation prediction time/source")
	}
	for _, state := range []map[knowledge.NodeID]float64{s.Source, s.Prediction} {
		for id, value := range state {
			if id <= 0 || !known(id) || value < 0 || math.IsNaN(value) || math.IsInf(value, 0) {
				return errors.New("invalid observation prediction neuron/value")
			}
		}
	}
	for id, q := range s.Patterns {
		if !patterns[id] || q <= 0 || q > 1 || math.IsNaN(q) {
			return errors.New("invalid observation pattern credit")
		}
	}
	return nil
}

func observedPopulation(trace *knowledge.ObservationTrace) map[knowledge.NodeID]float64 {
	out := map[knowledge.NodeID]float64{}
	if trace != nil {
		for _, frame := range trace.Stimulus {
			for _, id := range frame {
				out[id] = 1
			}
		}
	}
	return out
}

// LastObservedPopulation is detached direct input, excluding propagation and
// internal thought. It supports causal outcome validation at the runtime edge.
func (e *Engine) LastObservedPopulation() map[knowledge.NodeID]float64 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.observationPrediction == nil {
		return map[knowledge.NodeID]float64{}
	}
	return cloneState(e.observationPrediction.Source)
}

// ApplyObservedActionFeedback is called only after the runtime has validated
// an executed inquiry's binding. Its sources come from that binding, rather
// than from inferred prediction members or the last unrelated observation.
func (e *Engine) ApplyObservedActionFeedback(sources []knowledge.NodeID, prediction Prediction, actual map[knowledge.NodeID]float64, now time.Time) {
	source := map[knowledge.NodeID]float64{}
	for _, id := range sources {
		source[id] = 1
	}
	credit := e.Memory.Patterns.ObservedActionCredit(source, prediction.State)
	e.applyObservedFeedback(&ObservationPrediction{At: now, Source: source, Patterns: credit}, prediction.State, actual, now)
}

// applyObservedFeedback credits the previous actual source, not propagated
// target activity. Missing source or episode/clock discontinuity supplies no
// causal evidence. Each real observation replaces this single-use boundary.
func (e *Engine) applyObservedFeedback(previous *ObservationPrediction, predicted, actual map[knowledge.NodeID]float64, now time.Time) float64 {
	if previous == nil || len(actual) == 0 || now.Before(previous.At) || now.Sub(previous.At) > knowledge.ContextMaxAge {
		return 0
	}
	errorSignal := stateDifference(predicted, actual)
	eligibility := math.Exp2(-now.Sub(previous.At).Seconds())
	e.Memory.Lock()
	defer e.Memory.Unlock()
	for _, sourceID := range sortedNodeIDs(previous.Source) {
		source := e.Memory.Registry.GetByID(sourceID)
		if source == nil {
			continue
		}
		for _, s := range source.OutboundAll() {
			if s == nil || s.Inhibitory {
				continue
			}
			direction := clamp01(actual[s.TargetID]) - clamp01(predicted[s.TargetID])
			if abs(direction) < .001 {
				continue
			}
			delta := predictionLearningRate * errorSignal * abs(direction) * eligibility * clamp01(previous.Source[sourceID])
			if delta == 0 {
				continue
			}
			if direction < 0 {
				delta = -delta
			}
			s.Dynamic.Weight = clamp(s.Dynamic.Weight+delta, predictionMinWeight, predictionMaxWeight)
			s.Dynamic.Confidence = clamp01(s.Dynamic.Confidence + delta*.5)
			s.Dynamic.LastModification = now
			s.Dynamic.Eligibility = max(decayEligibility(s.Dynamic.Eligibility, s.Dynamic.LastEligibilityUpdate, now), eligibility*clamp01(previous.Source[sourceID]))
			if s.Dynamic.LastEligibilityUpdate.IsZero() || now.After(s.Dynamic.LastEligibilityUpdate) {
				s.Dynamic.LastEligibilityUpdate = now
			}
			s.Weight, s.Confidence = s.Dynamic.Weight, s.Dynamic.Confidence
		}
	}
	e.Memory.Patterns.ApplyObservedPatternFeedback(previous.Patterns, actual, errorSignal)
	return errorSignal
}
