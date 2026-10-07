package runtime

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

const ObservationEnvelopeSchemaVersion = 1

type ObservationProvenance struct {
	Source string `json:"source"`
	Modality string `json:"modality"`
	CaptureDevice string `json:"capture_device,omitempty"`
	Adapter string `json:"adapter,omitempty"`
	Synthetic bool `json:"synthetic"`
}

type ObservationEnvelope struct {
	SchemaVersion int `json:"schema_version"`
	BrainIdentity string `json:"brain_identity"`
	EventID string `json:"event_id"`
	SessionID string `json:"session_id,omitempty"`
	Sequence uint64 `json:"sequence"`
	Timestamp time.Time `json:"timestamp"`
	Modality string `json:"modality"`
	Source string `json:"source"`
	Provenance ObservationProvenance `json:"provenance"`
	Payload json.RawMessage `json:"payload,omitempty"`
	PayloadHash string `json:"payload_hash"`
	PayloadBytes int `json:"payload_bytes"`
	Tokens []string `json:"tokens,omitempty"`
	ContextTokens []string `json:"context_tokens,omitempty"`
	DataTokens []string `json:"data_tokens,omitempty"`
}

func (e ObservationEnvelope) Validate() error {
	if e.SchemaVersion != 0 && e.SchemaVersion != ObservationEnvelopeSchemaVersion { return fmt.Errorf("unsupported observation schema version %d", e.SchemaVersion) }
	if e.BrainIdentity != "" && e.BrainIdentity != BrainIdentity { return fmt.Errorf("observation belongs to brain %q", e.BrainIdentity) }
	if strings.TrimSpace(e.EventID) == "" { return errors.New("observation event_id is required") }
	if strings.TrimSpace(e.Modality) == "" { return errors.New("observation modality is required") }
	if strings.TrimSpace(e.Source) == "" { return errors.New("observation source is required") }
	if e.Timestamp.IsZero() { return errors.New("observation timestamp is required") }
	if e.PayloadBytes < 0 { return errors.New("observation payload_bytes is invalid") }
	if len(e.Payload) > 0 {
		sum := sha256.Sum256(e.Payload)
		expected := "sha256:" + hex.EncodeToString(sum[:])
		if e.PayloadHash != "" && e.PayloadHash != expected { return errors.New("observation payload hash mismatch") }
	}
	return nil
}

func NormalizeObservationEnvelope(e ObservationEnvelope) (ObservationEnvelope, error) {
	if e.SchemaVersion == 0 { e.SchemaVersion = ObservationEnvelopeSchemaVersion }
	if e.BrainIdentity == "" { e.BrainIdentity = BrainIdentity }
	if e.Timestamp.IsZero() { e.Timestamp = time.Now().UTC() }
	e.Timestamp = e.Timestamp.UTC()
	if len(e.Payload) > 0 {
		sum := sha256.Sum256(e.Payload)
		e.PayloadHash = "sha256:" + hex.EncodeToString(sum[:])
		e.PayloadBytes = len(e.Payload)
	}
	e.Modality = strings.TrimSpace(e.Modality)
	e.Source = strings.TrimSpace(e.Source)
	if err := e.Validate(); err != nil { return ObservationEnvelope{}, err }
	return e, nil
}

type TelemetryEvent struct {
	Type string `json:"type"`
	BrainIdentity string `json:"brain_identity"`
	StateRevision uint64 `json:"state_revision"`
	Timestamp time.Time `json:"timestamp"`
	EventHash string `json:"event_hash"`
	CanonicalStateHash string `json:"canonical_state_hash"`
	Event *Event `json:"event,omitempty"`
	Observation *ObservationEnvelope `json:"observation,omitempty"`
	Outcome *OutcomeEvent `json:"outcome,omitempty"`
	StateDelta *CognitiveStateDelta `json:"state_delta,omitempty"`
	PredictionError *float64 `json:"prediction_error,omitempty"`
	Plasticity map[string]float64 `json:"plasticity,omitempty"`
}

type TelemetrySubscriber struct { ch chan TelemetryEvent; once sync.Once }
func (s *TelemetrySubscriber) C() <-chan TelemetryEvent { return s.ch }
func (s *TelemetrySubscriber) close() { s.once.Do(func(){ close(s.ch) }) }

type TelemetryHub struct {
	mu sync.RWMutex
	subscribers map[*TelemetrySubscriber]struct{}
	capacity int
}
func NewTelemetryHub(capacity int) *TelemetryHub {
	if capacity < 1 { capacity = 64 }
	return &TelemetryHub{subscribers: make(map[*TelemetrySubscriber]struct{}), capacity: capacity}
}
func (h *TelemetryHub) Subscribe() *TelemetrySubscriber {
	if h == nil { return nil }
	s := &TelemetrySubscriber{ch: make(chan TelemetryEvent, h.capacity)}
	h.mu.Lock(); h.subscribers[s] = struct{}{}; h.mu.Unlock()
	return s
}
func (h *TelemetryHub) Unsubscribe(s *TelemetrySubscriber) {
	if h == nil || s == nil { return }
	h.mu.Lock()
	if _, ok := h.subscribers[s]; ok { delete(h.subscribers,s); s.close() }
	h.mu.Unlock()
}
func (h *TelemetryHub) Publish(event TelemetryEvent) {
	if h == nil { return }
	h.mu.RLock(); subs := make([]*TelemetrySubscriber,0,len(h.subscribers))
	for s := range h.subscribers { subs=append(subs,s) }; h.mu.RUnlock()
	for _, s := range subs {
		select { case s.ch <- event: default: h.Unsubscribe(s) }
	}
}
