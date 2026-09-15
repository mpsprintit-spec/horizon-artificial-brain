package runtime

import (
	"errors"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
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

// ObserveOutcome appends the outcome before changing the neural substrate.
// The outcome is then learned into the same Brain as an experience with a
// causal link to the originating request.
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

	r.mu.Lock()
	defer r.mu.Unlock()
	nextSeq := r.seq + 1
	if r.eventLog != nil {
		if err := r.eventLog.Append(LoggedEvent{
			SchemaVersion: EventLogSchemaVersion,
			BrainIdentity: BrainIdentity,
			Sequence: nextSeq,
			Type: EventTypeOutcome,
			Timestamp: outcome.ObservedAt,
			Outcome: cloneOutcome(outcome),
		}); err != nil {
			return r.seq, err
		}
	}

	weight := 0.45
	confidence := 0.30
	if outcome.Success {
		weight = 0.60
		confidence = 0.50
	}
	experience := learningExperienceFromOutcome(outcome, weight, confidence)
	r.learning.LearnExperience(experience, outcome.ObservedAt)
	r.seq = nextSeq
	return r.seq, nil
}

func learningExperienceFromOutcome(outcome OutcomeEvent, weight, confidence float64) learning.Experience {
	return learning.Experience{
		ExperienceID: outcome.RequestID + ":outcome",
		Sequence: append([]string(nil), outcome.Observation...),
		Weight: weight,
		Confidence: confidence,
		Source: outcome.Source,
		Modality: outcome.Modality,
		Timestamp: outcome.ObservedAt,
		Reliability: outcome.Reliability,
		IndependenceGroup: outcome.RequestID,
		CausalLink: outcome.RequestID,
	}
}

func cloneOutcome(in OutcomeEvent) *OutcomeEvent {
	out := in
	out.Observation = append([]string(nil), in.Observation...)
	return &out
}
