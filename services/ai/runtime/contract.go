package runtime

import (
	"errors"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

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
	return CognitiveState{BrainIdentity: o.BrainIdentity, Sequence: o.Sequence, Timestamp: o.Timestamp, ActiveNodeIDs: append([]knowledge.NodeID(nil), o.RankedNodeIDs...), Activations: cloneNodeValues(o.Activations), Confidence: cloneNodeValues(o.Confidence), Resonance: o.Resonance, PredictionError: o.PredictionError}
}

// OutcomeEvent records an externally observed consequence of a previously
// issued action/recommendation. It is an event, not a second memory store.
type OutcomeEvent struct {
	RequestID string
	BrainIdentity string
	Success bool
	Observation []string
	Source string
	Modality string
	ObservedAt time.Time
	Reliability float64
}

// ObserveOutcome records the outcome first, then evaluates it against the
// explicit action binding and learning policy. An outcome never guesses a
// neural target and never mutates the brain before promotion is accepted.
func (r *BrainRuntime) ObserveOutcome(outcome OutcomeEvent) (uint64, error) {
	if r == nil || r.brain == nil || r.learning == nil || r.promotion == nil || r.evidence == nil { return 0, errors.New("brain runtime is not initialized") }
	if outcome.BrainIdentity == "" { outcome.BrainIdentity = BrainIdentity }
	if outcome.BrainIdentity != BrainIdentity { return 0, errors.New("outcome belongs to a different brain") }
	if outcome.RequestID == "" { return 0, errors.New("outcome request ID is required") }
	if len(outcome.Observation) == 0 { return 0, errors.New("outcome observation is required") }
	if outcome.ObservedAt.IsZero() { outcome.ObservedAt = r.now() }
	if outcome.Source == "" { outcome.Source = "outcome" }
	if outcome.Modality == "" { outcome.Modality = "execution-outcome" }

	r.mu.Lock(); defer r.mu.Unlock()
	binding, ok := r.actions[outcome.RequestID]
	if !ok { return 0, errors.New("outcome has no registered action binding") }
	if binding.BrainIdentity != BrainIdentity { return 0, errors.New("action binding belongs to a different brain") }

	nextSeq := r.seq + 1
	if r.eventLog != nil {
		if err := r.eventLog.Append(LoggedEvent{SchemaVersion: EventLogSchemaVersion, BrainIdentity: BrainIdentity, Sequence: nextSeq, Type: EventTypeOutcome, Timestamp: outcome.ObservedAt, Outcome: cloneOutcome(outcome)}); err != nil { return r.seq, err }
	}

	weight, confidence, contradictions := 0.45, 0.30, 1
	if outcome.Success { weight, confidence, contradictions = 0.60, 0.50, 0 }
	evidence := learning.Evidence{Weight: weight, Confidence: confidence, Reliability: clamp01(outcome.Reliability), IndependentSources: 1, Contradictions: contradictions}
	for _, nodeID := range binding.TargetNodeIDs {
		combined, err := r.evidence.Record(learning.OutcomeEvidence{RequestID: outcome.RequestID, NodeID: nodeID, Source: outcome.Source, Evidence: evidence})
		if err != nil { return r.seq, err }
		if _, err := r.promotion.Apply(nodeID, combined, outcome.ObservedAt); err != nil { return r.seq, err }
	}
	r.seq = nextSeq
	return r.seq, nil
}

func cloneOutcome(in OutcomeEvent) *OutcomeEvent { out := in; out.Observation = append([]string(nil), in.Observation...); return &out }
func clamp01(v float64) float64 { if v < 0 { return 0 }; if v > 1 { return 1 }; return v }
