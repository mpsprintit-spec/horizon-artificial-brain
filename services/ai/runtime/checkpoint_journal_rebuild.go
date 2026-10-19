package runtime

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// RebuildRuntimeFromCheckpointAndJournal restores a fresh runtime from an
// anchored v7 checkpoint, then replays only committed events after its
// sequence. The journal writer must be held exclusively by the caller. An
// incomplete journal is rejected; repair the trailing intent separately.
//
// The result is intentionally detached from the journal. A supervisor must
// retire the old runtime and coordinate its own atomic handover before
// attaching the writer. Never use this with a runtime still serving traffic.
func RebuildRuntimeFromCheckpointAndJournal(log *EventLog, checkpointPath string) (*BrainRuntime, error) {
	if log == nil { return nil, errors.New("event log is not initialized") }
	if checkpointPath == "" { return nil, errors.New("checkpoint path is empty") }

	log.mu.Lock()
	defer log.mu.Unlock()
	if log.file == nil || log.lockFile == nil {
		return nil, errors.New("event log writer is not open and locked")
	}
	// Always validate the complete committed journal, not merely the tail.
	events, err := ReadEventLog(log.file.Name())
	if err != nil { return nil, fmt.Errorf("verify committed journal: %w", err) }

	data, err := os.ReadFile(checkpointPath)
	if err != nil { return nil, fmt.Errorf("read recovery checkpoint: %w", err) }
	var snapshot BrainSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("decode recovery checkpoint: %w", err)
	}
	if snapshot.SchemaVersion != BrainSnapshotSchemaVersion {
		return nil, fmt.Errorf("checkpoint schema %d cannot support journal offset replay", snapshot.SchemaVersion)
	}
	// Earlier v5 checkpoints used a graph-only legacy Brain encoding that
	// omitted developmental episodes, experience traces, policy state and
	// surface metadata. A checksum can authenticate such an incomplete file,
	// but cannot reconstruct the omitted memory. Fail closed for offset replay.
	var brainFormat struct{ SchemaVersion int `json:"schema_version"` }
	if err := json.Unmarshal(snapshot.Brain, &brainFormat); err != nil {
		return nil, fmt.Errorf("decode checkpoint canonical brain header: %w", err)
	}
	if brainFormat.SchemaVersion != 2 {
		return nil, errors.New("checkpoint brain format is incomplete for journal offset recovery; create a new canonical checkpoint")
	}
	if snapshot.JournalPrefixHash == "" {
		return nil, errors.New("checkpoint was not anchored to a committed journal")
	}
	if snapshot.Sequence > uint64(len(events)) {
		return nil, fmt.Errorf("checkpoint sequence %d is ahead of committed journal sequence %d", snapshot.Sequence, len(events))
	}
	prefix := events[:int(snapshot.Sequence)]
	digest, err := journalPrefixHash(prefix)
	if err != nil { return nil, fmt.Errorf("hash journal prefix: %w", err) }
	if snapshot.JournalPrefixHash != digest {
		return nil, errors.New("checkpoint journal prefix hash mismatch")
	}

	// Each processed event at the checkpoint boundary must be represented
	// exactly in the authenticated journal prefix (no phantom dedup entries).
	index := make(map[string]uint64)
	for _, logged := range prefix {
		if input := processedJournalEvent(logged); input != nil && input.ID != "" {
			index[input.ID] = logged.Sequence
		}
	}
	if len(index) != len(snapshot.ProcessedEvents) {
		return nil, errors.New("checkpoint processed-event index disagrees with journal prefix")
	}
	for id, seq := range index {
		if snapshot.ProcessedEvents[id] != seq {
			return nil, fmt.Errorf("checkpoint processed-event index differs for %q", id)
		}
	}

	// Checkpoint owns the canonical neural substrate; no secondary brain is
	// created. Runtime construction supplies fresh transient engines that
	// RestoreCheckpoint reconnects to that canonical substrate.
	fresh := NewBrainRuntime(knowledge.NewBootstrapBrain())
	if fresh == nil { return nil, errors.New("could not initialize isolated runtime") }
	if err := fresh.RestoreCheckpoint(checkpointPath); err != nil {
		return nil, fmt.Errorf("restore verified checkpoint: %w", err)
	}
	if fresh.LastSequence() != snapshot.Sequence {
		return nil, fmt.Errorf("restored sequence mismatch: got %d want %d", fresh.LastSequence(), snapshot.Sequence)
	}
	tail := events[int(snapshot.Sequence):]
	if err := replayEventLogFromSequence(fresh, tail, snapshot.Sequence+1); err != nil {
		return nil, fmt.Errorf("replay committed checkpoint suffix: %w", err)
	}
	if fresh.LastSequence() != uint64(len(events)) {
		return nil, fmt.Errorf("recovered sequence mismatch: got %d want %d", fresh.LastSequence(), len(events))
	}
	if fresh.RecoveryRequired() { return nil, ErrRuntimeRecoveryRequired }
	return fresh, nil
}
