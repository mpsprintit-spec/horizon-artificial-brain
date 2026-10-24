package runtime

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/project-horizon/horizon-core/services/ai/internal/jsonstream"
)

// checkpointChecksumV5 binds both the neural state and recovery metadata.
// In contrast to the legacy v4 checksum, changing the event sequence,
// timestamp predecessor, dedup index, or journal anchor is detectable.
func checkpointChecksumV5(snapshot BrainSnapshot) (string, error) {
	return checkpointChecksumV5WithBrain(snapshot, nil)
}

func checkpointChecksumV5WithBrain(snapshot BrainSnapshot, brainWriter jsonstream.ValueWriter) (string, error) {
	// These fields are omitted from the snapshot JSON when empty. Normalize
	// them here so encoding and subsequent decoding hash the same state.
	processed := snapshot.ProcessedEvents
	if len(processed) == 0 {
		processed = nil
	}
	population := snapshot.LastObservationPopulation
	if len(population) == 0 {
		population = nil
	}
	payload := struct {
		SchemaVersion             int                       `json:"schema_version"`
		BrainIdentity             string                    `json:"brain_identity"`
		Sequence                  uint64                    `json:"sequence"`
		Brain                     json.RawMessage           `json:"brain"`
		Activation                any                       `json:"activation"`
		CognitiveState            CognitiveState            `json:"cognitive_state"`
		InquiryValence            map[InquiryAction]float64 `json:"inquiry_valence"`
		LastObservationPopulation any                       `json:"last_observation_population"`
		LastObservationAt         any                       `json:"last_observation_at"`
		ProcessedEvents           map[string]uint64         `json:"processed_events"`
		JournalPrefixHash         string                    `json:"journal_prefix_hash"`
	}{
		SchemaVersion:             snapshot.SchemaVersion,
		BrainIdentity:             snapshot.BrainIdentity,
		Sequence:                  snapshot.Sequence,
		Brain:                     snapshot.Brain,
		Activation:                snapshot.Activation,
		CognitiveState:            snapshot.CognitiveState,
		InquiryValence:            snapshot.InquiryValence,
		LastObservationPopulation: population,
		LastObservationAt:         snapshot.LastObservationAt,
		ProcessedEvents:           processed,
		JournalPrefixHash:         snapshot.JournalPrefixHash,
	}
	if brainWriter != nil {
		hash := sha256.New()
		if err := jsonstream.Object(hash, payload, false, "", map[string]jsonstream.ValueWriter{"brain": brainWriter}); err != nil {
			return "", err
		}
		return hex.EncodeToString(hash.Sum(nil)), nil
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

// checkpointChecksumV6 binds all v5 neural and runtime metadata PLUS the
// separate adapter surface annotations. Old v5 checksums remain readable.
func checkpointChecksumV6(snapshot BrainSnapshot) (string, error) {
	return checkpointChecksumV6WithBrain(snapshot, nil)
}
func checkpointChecksumV6WithBrain(snapshot BrainSnapshot, writer jsonstream.ValueWriter) (string, error) {
	base, err := checkpointChecksumV5WithBrain(snapshot, writer)
	if err != nil {
		return "", err
	}
	annotations := snapshot.SurfaceAnnotations
	if len(annotations) == 0 {
		annotations = nil
	}
	data, err := json.Marshal(struct {
		BaseDigest         string `json:"base_digest"`
		SurfaceAnnotations any    `json:"surface_annotations"`
	}{BaseDigest: base, SurfaceAnnotations: annotations})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// v7 authenticates the runtime action/evidence state required by later outcomes.
func checkpointChecksumV7(snapshot BrainSnapshot) (string, error) {
	return checkpointChecksumV7WithBrain(snapshot, nil)
}
func checkpointChecksumV7WithBrain(snapshot BrainSnapshot, writer jsonstream.ValueWriter) (string, error) {
	base, err := checkpointChecksumV6WithBrain(snapshot, writer)
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
	for id, seq := range input {
		out[id] = seq
	}
	return out
}

// v8 binds the serving contract separately from neural state. All historical
// checksum laws remain unchanged and continue to validate old checkpoints.
func checkpointChecksumV8(snapshot BrainSnapshot) (string, error) {
	return checkpointChecksumV8WithBrain(snapshot, nil)
}

func checkpointChecksumV8WithBrain(snapshot BrainSnapshot, writer jsonstream.ValueWriter) (string, error) {
	base, err := checkpointChecksumV7WithBrain(snapshot, writer)
	if err != nil { return "", err }
	data, err := json.Marshal(struct {
		Base string `json:"base_digest"`
		NeuralWrittenGrounding bool `json:"neural_written_grounding"`
	}{base, snapshot.NeuralWrittenGrounding})
	if err != nil { return "", err }
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func checkpointChecksumV9(snapshot BrainSnapshot) (string,error) { return checkpointChecksumV9WithBrain(snapshot,nil) }

// v9 authenticates current physical/linguistic continuation and the enabled
// numerical composition law. Historical checksum laws are left untouched.
func checkpointChecksumV9WithBrain(snapshot BrainSnapshot,writer jsonstream.ValueWriter) (string,error) {
	base,err:=checkpointChecksumV8WithBrain(snapshot,writer);if err!=nil{return "",err}
	data,err:=json.Marshal(struct{Base string `json:"base_digest"`;Composition *CompositionContinuation `json:"composition"`}{base,snapshot.Composition});if err!=nil{return "",err}
	sum:=sha256.Sum256(data);return hex.EncodeToString(sum[:]),nil
}

// journalPrefixHash hashes the *logical committed* prefix, never the raw
// intent/commit physical framing. This anchors a checkpoint to a specific
// replayable event history and detects tampering with its earlier records.
func journalPrefixHash(events []LoggedEvent) (string, error) {
	// ReadEventLog returns nil for an empty journal. Preserve its historical
	// SHA256(null), including when recovery slices a zero-length prefix from
	// a journal that now has a suffix. Do not change old checkpoint checksums.
	if len(events) == 0 {
		events = nil
	}
	data, err := json.Marshal(events)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

// checkpointJournalPrefixLocked requires r.mu. Lock order is runtime -> log,
// the same order as the event-writing path.
func (r *BrainRuntime) checkpointJournalPrefixLocked() (string, error) {
	if r.eventLog == nil {
		return "", nil
	}
	log := r.eventLog
	log.mu.Lock()
	defer log.mu.Unlock()
	if log.file == nil || log.lockFile == nil {
		return "", errors.New("event log writer is not open and locked")
	}
	events, err := ReadEventLog(log.file.Name())
	if err != nil {
		return "", err
	}
	if uint64(len(events)) != r.seq {
		return "", fmt.Errorf("checkpoint sequence %d disagrees with committed journal count %d", r.seq, len(events))
	}
	return journalPrefixHash(events)
}
