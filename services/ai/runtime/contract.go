package runtime

import (
	"errors"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// CognitiveState is the explicit runtime state passed from neural processing
// toward interpretation. It contains only substrate/runtime observations; it
// does not encode semantic rules or a hard-coded reasoning sequence.
type CognitiveState struct {
	BrainIdentity   string
	Sequence        uint64
	Timestamp       time.Time
	ActiveNodeIDs   []knowledge.NodeID
	Activations     map[knowledge.NodeID]float64
	Confidence      map[knowledge.NodeID]float64
	Resonance       float64
	PredictionError float64
}

func (o CognitiveOutput) State() CognitiveState {
	return CognitiveState{
		BrainIdentity: o.BrainIdentity,
		Sequence: o.Sequence,
		Timestamp: o.Timestamp,
		ActiveNodeIDs: append([]knowledge.NodeID(nil), o.RankedNodeIDs...),
		Activations: cloneNodeValues(o.Activations),
		Confidence: cloneNodeValues(o.Confidence),
		Resonance: o.Resonance,
		PredictionError: o.PredictionError,
	}
}

// OutcomeEvent records an externally observed consequence of a previously
// issued action/recommendation. It is an event, not a second memory store.
type OutcomeEvent struct {
	RequestID     string
	BrainIdentity string
	Success       bool
	Observation   []string
	Source        string
	Modality      string
	ObservedAt    time.Time
	Reliability   float64
}

func (r *BrainRuntime) ObserveOutcome(outcome OutcomeEvent) (uint64, error) {
	if r == nil || r.brain == nil || r.learning == nil {
		return 0, errors.New("brain runtime is not initialized")
	}
	if outcome.BrainIdentity == "" {
		outcome.BrainIdentity = BrainIdentity
	}
	if outcome.BrainIdentity != BrainIdentity {
		return 0, errors.New("outcome belongs to a different brain")
	}
	if outcome.RequestID == "" {
		return 0, errors.New("outcome request ID is required")
	}
	if len(outcome.Observation) == 0 {
		return 0, errors.New("outcome observation is required")
	}
	if outcome.ObservedAt.IsZero() {
		outcome.ObservedAt = r.now()
	}
	if outcome.Source == "" {
		outcome.Source = "outcome"
	}
	if outcome.Modality == "" {
		outcome.Modality = "execution-outcome"
	}

	// Outcomes are learned into the same neural substrate. Success/failure is
	// provenance on the experience, not a semantic relation encoded in nodes.
	weight := 0.45
	confidence := 0.30
	if outcome.Success {
		weight = 0.60
		confidence = 0.50
	}
	experience := learningExperienceFromOutcome(outcome, weight, confidence)
	return r.LearnExperience(experience, outcome.ObservedAt)
}
