package runtime

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

const RevisionTelemetrySchemaVersion = 2

// publishBuilt selects recipients before doing any work. Subscription and
// cancellation never wait for serialization. A subscriber joining during a
// build starts with its handshake snapshot, not an event built for an older
// recipient set. The runtime lock binds the build to one completed revision.
func (h *TelemetryHub) publishBuilt(build func(canonicalHash bool) (TelemetryEvent, error)) {
	if h == nil {
		return
	}
	h.mu.RLock()
	if h.closed || len(h.subscribers) == 0 {
		h.mu.RUnlock()
		return
	}
	recipients := make([]*TelemetrySubscriber, 0, len(h.subscribers))
	canonicalHash := false
	for s := range h.subscribers {
		recipients = append(recipients, s)
		canonicalHash = canonicalHash || !s.revisionsOnly
	}
	h.mu.RUnlock()
	event, err := build(canonicalHash)
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, s := range recipients {
		if _, exists := h.subscribers[s]; !exists {
			continue
		}
		if err == nil {
			item := event
			if s.revisionsOnly {
				item.SchemaVersion = RevisionTelemetrySchemaVersion
				item.CanonicalStateHash = ""
			}
			select {
			case s.ch <- item:
				continue
			default:
			}
		}
		// Overflow or encoding failure requires resynchronization. Never send
		// a hash of an incomplete/failed encoding as a valid integrity digest.
		delete(h.subscribers, s)
		s.close()
	}
}

// Caller holds r.mu and has completed the entire transition/durable commit.
// Revision telemetry does not serialize or clone the neural graph. With no
// subscribers the event envelope and plasticity map are not built either.
func (r *BrainRuntime) publishTelemetryLocked(eventType string, at time.Time, event *Event, observation *ObservationEnvelope, outcome *OutcomeEvent, delta *CognitiveStateDelta, predictionError *float64) {
	r.telemetry.publishBuilt(func(canonicalHash bool) (TelemetryEvent, error) {
		payload := struct {
			Type        string               `json:"type"`
			Sequence    uint64               `json:"sequence"`
			Event       *Event               `json:"event,omitempty"`
			Observation *ObservationEnvelope `json:"observation,omitempty"`
			Outcome     *OutcomeEvent        `json:"outcome,omitempty"`
		}{eventType, r.seq, event, observation, outcome}
		data, err := json.Marshal(payload)
		if err != nil {
			return TelemetryEvent{}, err
		}
		eh := sha256.Sum256(data)
		e := TelemetryEvent{Type: eventType, BrainIdentity: BrainIdentity, StateRevision: r.seq, Timestamp: at.UTC(), EventHash: "sha256:" + hex.EncodeToString(eh[:]), Event: event, Observation: observation, Outcome: outcome, StateDelta: delta, PredictionError: predictionError}
		if canonicalHash {
			hash, err := r.brain.CanonicalHash()
			if err != nil {
				return TelemetryEvent{}, err
			}
			e.CanonicalStateHash = hash
		}
		r.brain.RLock()
		e.Plasticity = cloneMonitorFloatMap(r.brain.BrainState.PlasticityState)
		r.brain.RUnlock()
		return e, nil
	})
}
