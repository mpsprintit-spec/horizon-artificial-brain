package runtime

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// RecoverTrailingSensorIntent removes only a verified, uncommitted final
// sensor-intent record. The full original journal is copied and synced to a
// backup first. A damaged, interrupted, or conflicting journal is rejected.
// This repairs durable log bytes, NOT the possibly mutated in-memory brain:
// callers must discard that runtime and replay the recovered log into a fresh
// canonical runtime before accepting any new mutation.
//
// Only call this method on the exclusive EventLog writer. Never call it while
// BrainRuntime is processing events.
func (l *EventLog) RecoverTrailingSensorIntent() (string, error) {
	if l == nil { return "", errors.New("event log is not initialized") }
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file == nil || l.lockFile == nil {
		return "", errors.New("event log writer is not open and locked")
	}
	path := l.file.Name()
	data, err := os.ReadFile(path)
	if err != nil { return "", fmt.Errorf("read journal for recovery: %w", err) }
	if len(data) == 0 || data[len(data)-1] != '\n' {
		return "", errors.New("journal does not end with a complete newline-terminated record")
	}
	lines := bytes.Split(data[:len(data)-1], []byte{'\n'})
	if len(lines) == 0 || len(lines[len(lines)-1]) == 0 {
		return "", errors.New("journal has no final record to recover")
	}
	var tail LoggedEvent
	if err := json.Unmarshal(lines[len(lines)-1], &tail); err != nil {
		return "", fmt.Errorf("decode journal tail: %w", err)
	}
	if tail.Type != EventTypeSensorIntent || tail.SchemaVersion != EventLogSchemaVersion ||
		tail.BrainIdentity != BrainIdentity || tail.Event == nil ||
		tail.Event.SensorObservation == nil || tail.Event.ID == "" ||
		tail.Event.ID != tail.Event.SensorObservation.SampleID ||
		!tail.Timestamp.Equal(tail.Event.Timestamp) ||
		!tail.Timestamp.Equal(tail.Event.SensorObservation.Timestamp) ||
		tail.IntentHash != "" || tail.Experience != nil ||
		tail.KnowledgeDocument != nil || tail.Vector != nil ||
		tail.Outcome != nil || tail.Consequence != nil {
		return "", errors.New("journal tail is not a valid uncommitted sensor intent")
	}

	prefix := data[:len(data)-len(lines[len(lines)-1])-1]
	// Validate the entire preceding log with the existing strict reader,
	// including hashes, sequence continuity, and all prior commit pairs.
	dir := filepath.Dir(path)
	staged, err := os.CreateTemp(dir, ".horizon-journal-prefix-*.tmp")
	if err != nil { return "", fmt.Errorf("stage journal prefix: %w", err) }
	stagedPath := staged.Name()
	defer os.Remove(stagedPath)
	if _, err := staged.Write(prefix); err != nil { staged.Close(); return "", err }
	if err := staged.Close(); err != nil { return "", err }
	events, err := ReadEventLog(stagedPath)
	if err != nil { return "", fmt.Errorf("journal prefix is not fully committed: %w", err) }
	if tail.Sequence != uint64(len(events))+1 {
		return "", fmt.Errorf("uncommitted intent sequence %d does not follow committed prefix %d", tail.Sequence, len(events))
	}

	backup, err := os.CreateTemp(dir, ".horizon-journal-recovery-*.backup")
	if err != nil { return "", fmt.Errorf("create journal backup: %w", err) }
	backupPath := backup.Name()
	backupValid := false
	defer func() { if !backupValid { _ = os.Remove(backupPath) } }()
	if _, err := backup.Write(data); err != nil { backup.Close(); return "", fmt.Errorf("write journal backup: %w", err) }
	if err := backup.Sync(); err != nil { backup.Close(); return "", fmt.Errorf("sync journal backup: %w", err) }
	if err := backup.Close(); err != nil { return "", fmt.Errorf("close journal backup: %w", err) }
	backupValid = true

	if err := l.file.Truncate(int64(len(prefix))); err != nil {
		return backupPath, fmt.Errorf("truncate uncommitted journal tail (backup at %s): %w", backupPath, err)
	}
	if err := l.file.Sync(); err != nil {
		return backupPath, fmt.Errorf("sync recovered journal (backup at %s): %w", backupPath, err)
	}
	if _, err := ReadEventLog(path); err != nil {
		return backupPath, fmt.Errorf("verify recovered journal (backup at %s): %w", backupPath, err)
	}
	return backupPath, nil
}
