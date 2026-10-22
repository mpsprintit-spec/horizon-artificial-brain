package runtime

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
)

// checkpointChecksumV5 binds both the neural state and recovery metadata.
// In contrast to the legacy v4 checksum, changing the event sequence,
// timestamp predecessor, dedup index, or journal anchor is detectable.
func checkpointChecksumV5(snapshot BrainSnapshot) (string, error) {
	// These fields are omitted from the snapshot JSON when empty. Normalize
	// them here so encoding and subsequent decoding hash the same state.
	processed := snapshot.ProcessedEvents
	if len(processed) == 0 { processed = nil }
	population := snapshot.LastObservationPopulation
	if len(population) == 0 { population = nil }
	payload := struct {
		SchemaVersion int `json:"schema_version"`
		BrainIdentity string `json:"brain_identity"`
		Sequence uint64 `json:"sequence"`
		Brain json.RawMessage `json:"brain"`
		Activation any `json:"activation"`
		CognitiveState CognitiveState `json:"cognitive_state"`
		InquiryValence map[InquiryAction]float64 `json:"inquiry_valence"`
		LastObservationPopulation any `json:"last_observation_population"`
		LastObservationAt any `json:"last_observation_at"`
		ProcessedEvents map[string]uint64 `json:"processed_events"`
		JournalPrefixHash string `json:"journal_prefix_hash"`
	}{
		SchemaVersion: snapshot.SchemaVersion,
		BrainIdentity: snapshot.BrainIdentity,
		Sequence: snapshot.Sequence,
		Brain: snapshot.Brain,
		Activation: snapshot.Activation,
		CognitiveState: snapshot.CognitiveState,
		InquiryValence: snapshot.InquiryValence,
		LastObservationPopulation: population,
		LastObservationAt: snapshot.LastObservationAt,
		ProcessedEvents: processed,
		JournalPrefixHash: snapshot.JournalPrefixHash,
	}
	data, err := json.Marshal(payload)
	if err != nil { return "", err }
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

// checkpointChecksumV6 binds all v5 neural and runtime metadata PLUS the
// separate adapter surface annotations. Old v5 checksums remain readable.
func checkpointChecksumV6(snapshot BrainSnapshot) (string, error) {
	base, err := checkpointChecksumV5(snapshot)
	if err != nil { return "", err }
	annotations := snapshot.SurfaceAnnotations
	if len(annotations) == 0 { annotations = nil }
	data, err := json.Marshal(struct{
		BaseDigest string `json:"base_digest"`
		SurfaceAnnotations any `json:"surface_annotations"`
	}{BaseDigest: base, SurfaceAnnotations: annotations})
	if err != nil { return "", err }
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// v7 authenticates the runtime action/evidence state required by later outcomes.
func checkpointChecksumV7(snapshot BrainSnapshot) (string, error) {
	base, err := checkpointChecksumV6(snapshot)
	if err != nil {
		return "", err
	}
	actions := snapshot.ActionBindings
	if len(actions) == 0 {
		actions = nil
	}
	data, err := json.Marshal(struct {
		Base     string
		Actions  map[string]ActionBinding
		Evidence any
	}{base, actions, snapshot.EvidenceLedger})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func cloneProcessedEventIndex(input map[string]uint64) map[string]uint64 {
	out := make(map[string]uint64, len(input))
	for id, seq := range input { out[id] = seq }
	return out
}

// journalPrefixHash hashes the *logical committed* prefix, never the raw
// intent/commit physical framing. This anchors a checkpoint to a specific
// replayable event history and detects tampering with its earlier records.
func journalPrefixHash(events []LoggedEvent) (string, error) {
	data, err := json.Marshal(events)
	if err != nil { return "", err }
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

// checkpointJournalPrefixLocked requires r.mu. Lock order is runtime -> log,
// the same order as the event-writing path.
func (r *BrainRuntime) checkpointJournalPrefixLocked() (string, error) {
	if r.eventLog == nil { return "", nil }
	log := r.eventLog
	log.mu.Lock()
	defer log.mu.Unlock()
	if log.file == nil || log.lockFile == nil {
		return "", errors.New("event log writer is not open and locked")
	}
	events, err := ReadEventLog(log.file.Name())
	if err != nil { return "", err }
	if uint64(len(events)) != r.seq {
		return "", fmt.Errorf("checkpoint sequence %d disagrees with committed journal count %d", r.seq, len(events))
	}
	return journalPrefixHash(events)
}
