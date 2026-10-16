package runtime

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/learning"
)

const EventLogSchemaVersion = 1

type LoggedEvent struct {
	SchemaVersion int                  `json:"schema_version"`
	BrainIdentity string               `json:"brain_identity"`
	Sequence      uint64               `json:"sequence"`
	Type          string               `json:"type"`
	Timestamp     time.Time            `json:"timestamp"`
	Event         *Event               `json:"event,omitempty"`
	Experience    *learning.Experience `json:"experience,omitempty"`
	Outcome       *OutcomeEvent        `json:"outcome,omitempty"`
	Consequence  *InquiryConsequenceEvent `json:"consequence,omitempty"`
	IntentHash   string `json:"intent_hash,omitempty"`
}

const (
	EventTypeProcess = "process"
	EventTypeLearn = "learn"
	EventTypeThink = "think"
	EventTypeOutcome = "outcome"
	EventTypeConsequence = "consequence"
	EventTypeSensorIntent = "sensor_intent"
	EventTypeSensorCommit = "sensor_commit"
)

type EventLog struct {
	mu sync.Mutex
	file *os.File
	lockFile *os.File
	// Test-only fault injection; guarded by mu like file writes.
	beforeAppend func(LoggedEvent) error
}

func OpenEventLog(path string) (*EventLog, error) {
	if path == "" {
		return nil, errors.New("event log path is empty")
	}
	lockFile, err := os.OpenFile(path+".writer.lock", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("open event log writer lock: %w", err)
	}
	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		lockFile.Close()
		return nil, fmt.Errorf("event log already has another writer: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)
		lockFile.Close()
		return nil, fmt.Errorf("open event log: %w", err)
	}
	return &EventLog{file: file, lockFile: lockFile}, nil
}

func (l *EventLog) Append(event LoggedEvent) error {
	if l == nil || l.file == nil { return errors.New("event log is not initialized") }
	if event.SchemaVersion == 0 { event.SchemaVersion = EventLogSchemaVersion }
	if event.BrainIdentity == "" { event.BrainIdentity = BrainIdentity }
	data, err := json.Marshal(event)
	if err != nil { return fmt.Errorf("encode logged event: %w", err) }
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.beforeAppend != nil {
		if err := l.beforeAppend(event); err != nil { return fmt.Errorf("inject event log append failure: %w", err) }
	}
	if _, err := l.file.Write(append(data, '\n')); err != nil { return fmt.Errorf("append logged event: %w", err) }
	if err := l.file.Sync(); err != nil { return fmt.Errorf("sync event log: %w", err) }
	return nil
}

func (l *EventLog) Close() error {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	var closeErr error
	if l.file != nil {
		closeErr = l.file.Close()
		l.file = nil
	}
	if l.lockFile != nil {
		if err := syscall.Flock(int(l.lockFile.Fd()), syscall.LOCK_UN); err != nil && closeErr == nil {
			closeErr = err
		}
		if err := l.lockFile.Close(); err != nil && closeErr == nil {
			closeErr = err
		}
		l.lockFile = nil
	}
	return closeErr
}


// RepairConflictingSequences repairs a legacy log written by multiple runtimes
// that each maintained their own sequence counter. It only proceeds when every
// record is valid, physical append order is non-decreasing by the old sequence,
// and the distinct old sequences form a complete range starting at 1. It keeps
// every record, assigns a single global sequence in physical append order, and
// writes an exact backup before atomically replacing the active log.
//
// The caller must hold the EventLog writer lock for the duration of recovery.
func (l *EventLog) RepairConflictingSequences() (string, error) {
	if l == nil {
		return "", errors.New("event log is not initialized")
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file == nil || l.lockFile == nil {
		return "", errors.New("event log writer lock is not held")
	}

	path := l.file.Name()
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read event log for recovery: %w", err)
	}
	if len(data) == 0 {
		return "", errors.New("cannot repair an empty event log")
	}

	rawLines := bytes.Split(data, []byte{'\n'})
	if len(rawLines) > 0 && len(rawLines[len(rawLines)-1]) == 0 {
		rawLines = rawLines[:len(rawLines)-1]
	}
	if len(rawLines) == 0 {
		return "", errors.New("event log contains no records")
	}

	events := make([]LoggedEvent, 0, len(rawLines))
	seen := make(map[uint64][]byte)
	var previous, maxSequence uint64
	conflicts := 0
	for i, raw := range rawLines {
		lineNumber := i + 1
		if len(raw) == 0 {
			return "", fmt.Errorf("cannot repair blank event log record at line %d", lineNumber)
		}
		var event LoggedEvent
		if err := json.Unmarshal(raw, &event); err != nil {
			return "", fmt.Errorf("cannot repair malformed event at line %d: %w", lineNumber, err)
		}
		if event.SchemaVersion != EventLogSchemaVersion {
			return "", fmt.Errorf("cannot repair unsupported schema version %d at line %d", event.SchemaVersion, lineNumber)
		}
		if event.BrainIdentity != BrainIdentity {
			return "", fmt.Errorf("cannot repair event for brain %q at line %d", event.BrainIdentity, lineNumber)
		}
		if event.Type == EventTypeSensorIntent || event.Type == EventTypeSensorCommit {
			return "", fmt.Errorf("cannot repair transactional sensor records at line %d using legacy sequence renumbering", lineNumber)
		}
		if event.Sequence == 0 {
			return "", fmt.Errorf("cannot repair zero event sequence at line %d", lineNumber)
		}
		if lineNumber == 1 && event.Sequence != 1 {
			return "", fmt.Errorf("cannot repair log whose first sequence is %d", event.Sequence)
		}
		if event.Sequence < previous {
			return "", fmt.Errorf("cannot safely repair non-monotonic sequence %d at line %d after %d", event.Sequence, lineNumber, previous)
		}
		previous = event.Sequence
		if event.Sequence > maxSequence {
			maxSequence = event.Sequence
		}
		canonical, err := json.Marshal(event)
		if err != nil {
			return "", fmt.Errorf("normalize event at line %d: %w", lineNumber, err)
		}
		if prior, exists := seen[event.Sequence]; exists {
			if !bytes.Equal(prior, canonical) {
				conflicts++
			}
		} else {
			seen[event.Sequence] = canonical
		}
		events = append(events, event)
	}
	if conflicts == 0 {
		return "", errors.New("event log has no conflicting duplicate sequences to repair")
	}
	if uint64(len(seen)) != maxSequence {
		return "", fmt.Errorf("cannot safely repair sequence gaps: found %d distinct sequences with maximum %d", len(seen), maxSequence)
	}

	// Keep all events and their payloads. The line order is the durable append
	// order, so assign one monotonically increasing sequence across that order.
	var repaired bytes.Buffer
	for i := range events {
		events[i].Sequence = uint64(i + 1)
		record, err := json.Marshal(events[i])
		if err != nil {
			return "", fmt.Errorf("encode recovered event %d: %w", i+1, err)
		}
		repaired.Write(record)
		repaired.WriteByte('\n')
	}

	backupPath := fmt.Sprintf("%s.conflict-backup-%s", path, time.Now().UTC().Format("20060102T150405.000000000Z"))
	backup, err := os.OpenFile(backupPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", fmt.Errorf("create event log recovery backup: %w", err)
	}
	if _, err := backup.Write(data); err != nil {
		backup.Close()
		os.Remove(backupPath)
		return "", fmt.Errorf("write event log recovery backup: %w", err)
	}
	if err := backup.Sync(); err != nil {
		backup.Close()
		os.Remove(backupPath)
		return "", fmt.Errorf("sync event log recovery backup: %w", err)
	}
	if err := backup.Close(); err != nil {
		os.Remove(backupPath)
		return "", fmt.Errorf("close event log recovery backup: %w", err)
	}

	temp, err := os.CreateTemp(filepath.Dir(path), ".horizon-event-log-recovery-*.tmp")
	if err != nil {
		return "", fmt.Errorf("create recovered event log: %w", err)
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if _, err := temp.Write(repaired.Bytes()); err != nil {
		temp.Close()
		return "", fmt.Errorf("write recovered event log: %w", err)
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return "", fmt.Errorf("sync recovered event log: %w", err)
	}
	if err := temp.Close(); err != nil {
		return "", fmt.Errorf("close recovered event log: %w", err)
	}

	// Keep the inter-process lock file open while replacing only the data file.
	if err := l.file.Close(); err != nil {
		l.file = nil
		return "", fmt.Errorf("close old event log before recovery: %w", err)
	}
	l.file = nil
	if err := os.Rename(tempPath, path); err != nil {
		reopened, reopenErr := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
		if reopenErr == nil {
			l.file = reopened
		}
		return "", fmt.Errorf("atomically replace event log: %w", err)
	}
	reopened, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return "", fmt.Errorf("reopen recovered event log (backup at %s): %w", backupPath, err)
	}
	l.file = reopened
	return backupPath, nil
}

// ErrUncommittedSensorTransaction indicates that a durable intent does not
// have a matching commit marker. It must not be silently replayed.
var ErrUncommittedSensorTransaction = errors.New("uncommitted sensor transaction")

func sensorIntentHash(intent LoggedEvent) (string, error) {
	data, err := json.Marshal(intent)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum[:]), nil
}

// ReadEventLog preserves the v1 event stream while interpreting a committed
// sensor intent/commit pair as one logical process event with one sequence.
// An orphan, interrupted, or mismatched pair is an error, not a successful
// observation. A caller must recover from a known-good checkpoint/log.
func ReadEventLog(path string) ([]LoggedEvent, error) {
	file, err := os.Open(path)
	if err != nil { return nil, err }
	defer file.Close()

	var events []LoggedEvent
	var expected uint64 = 1
	type eventKey struct {
		sequence uint64
		eventType string
	}
	seen := make(map[eventKey][]byte)
	var pending *LoggedEvent
	var pendingHash string
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := append([]byte(nil), scanner.Bytes()...)
		var event LoggedEvent
		if err := json.Unmarshal(line, &event); err != nil {
			return nil, fmt.Errorf("decode event log at line %d: %w", lineNumber, err)
		}
		if event.SchemaVersion != EventLogSchemaVersion {
			return nil, fmt.Errorf("unsupported event log schema version %d at line %d", event.SchemaVersion, lineNumber)
		}
		if event.BrainIdentity != BrainIdentity {
			return nil, fmt.Errorf("event belongs to brain %q at line %d", event.BrainIdentity, lineNumber)
		}
		canonical, err := json.Marshal(event)
		if err != nil {
			return nil, fmt.Errorf("normalize event at line %d: %w", lineNumber, err)
		}
		key := eventKey{sequence: event.Sequence, eventType: event.Type}
		if event.Sequence < expected {
			// Preserve legacy nonadjacent idempotent duplicate recovery, but
			// require identical bytes and the same physical record type.
			if prior, ok := seen[key]; ok && bytes.Equal(prior, canonical) {
				continue
			}
			return nil, fmt.Errorf("conflicting or unknown duplicate event sequence %d type %q at line %d", event.Sequence, event.Type, lineNumber)
		}
		if event.Sequence > expected || event.Sequence == 0 {
			return nil, fmt.Errorf("event sequence discontinuity at line %d: got %d want %d", lineNumber, event.Sequence, expected)
		}

		if pending != nil {
			if event.Type != EventTypeSensorCommit {
				return nil, fmt.Errorf("%w: sensor intent at sequence %d interrupted by %q at line %d", ErrUncommittedSensorTransaction, expected, event.Type, lineNumber)
			}
			if event.IntentHash == "" || event.IntentHash != pendingHash || event.Event != nil ||
				event.Experience != nil || event.Outcome != nil || event.Consequence != nil ||
				!event.Timestamp.Equal(pending.Timestamp) {
				return nil, fmt.Errorf("%w: sensor commit mismatch at line %d", ErrUncommittedSensorTransaction, lineNumber)
			}
			seen[key] = canonical
			committed := *pending
			committed.Type = EventTypeProcess
			committed.IntentHash = ""
			events = append(events, committed)
			pending = nil
			pendingHash = ""
			expected++
			continue
		}

		switch event.Type {
		case EventTypeSensorIntent:
			if event.Event == nil || event.Event.SensorObservation == nil ||
				event.Event.ID == "" || event.Event.ID != event.Event.SensorObservation.SampleID ||
				!event.Timestamp.Equal(event.Event.Timestamp) ||
				!event.Timestamp.Equal(event.Event.SensorObservation.Timestamp) ||
				event.IntentHash != "" {
				return nil, fmt.Errorf("invalid sensor intent payload at line %d", lineNumber)
			}
			digest, err := sensorIntentHash(event)
			if err != nil { return nil, fmt.Errorf("hash sensor intent at line %d: %w", lineNumber, err) }
			pending = &event
			pendingHash = digest
			seen[key] = canonical
		case EventTypeSensorCommit:
			return nil, fmt.Errorf("%w: sensor commit without intent at line %d", ErrUncommittedSensorTransaction, lineNumber)
		default:
			seen[key] = canonical
			events = append(events, event)
			expected++
		}
	}
	if err := scanner.Err(); err != nil { return nil, fmt.Errorf("read event log: %w", err) }
	if pending != nil {
		return nil, fmt.Errorf("%w: sensor intent sequence %d has no commit marker", ErrUncommittedSensorTransaction, pending.Sequence)
	}
	return events, nil
}

// ReplayEventLog re-applies durable events to the same neural substrate. The
// source log is temporarily detached so replay cannot append duplicate events.
// Replay is intended as a recovery operation and should not run concurrently
// with normal runtime mutation.
func ReplayEventLog(brainRuntime *BrainRuntime, events []LoggedEvent) error {
	if brainRuntime == nil { return errors.New("brain runtime is nil") }
	brainRuntime.mu.Lock()
	originalLog := brainRuntime.eventLog
	brainRuntime.eventLog = nil
	brainRuntime.mu.Unlock()
	defer func() {
		brainRuntime.mu.Lock()
		brainRuntime.eventLog = originalLog
		brainRuntime.mu.Unlock()
	}()

	var expected uint64 = 1
	for _, logged := range events {
		if logged.BrainIdentity != BrainIdentity { return fmt.Errorf("cannot replay event for brain %q", logged.BrainIdentity) }
		if logged.SchemaVersion != EventLogSchemaVersion { return fmt.Errorf("unsupported event log schema version %d", logged.SchemaVersion) }
		if logged.Sequence != expected { return fmt.Errorf("replay sequence discontinuity: got %d want %d", logged.Sequence, expected) }
		expected++
		switch logged.Type {
		case EventTypeLearn:
			if logged.Experience == nil { return errors.New("learn event has no experience") }
			if _, err := brainRuntime.LearnExperience(*logged.Experience, logged.Timestamp); err != nil { return err }
		case EventTypeProcess:
			if logged.Event == nil { return errors.New("process event has no event payload") }
			if logged.Event.SensorObservation != nil {
				// Sensor processing includes deterministic projection into the canonical
				// substrate before cognitive processing. Re-run the complete boundary,
				// not only the downstream event with previously assigned node IDs.
				orchestrator := NewCognitiveOrchestrator(brainRuntime)
				if _, err := orchestrator.ProcessSensorObservation(*logged.Event.SensorObservation); err != nil {
					return fmt.Errorf("replay sensor observation %q: %w", logged.Event.SensorObservation.SampleID, err)
				}
			} else if _, _, err := brainRuntime.Process(*logged.Event); err != nil {
				return err
			}
		case EventTypeThink:
			if logged.Event == nil { return errors.New("think event has no event payload") }
			if _, _, err := brainRuntime.ThinkAt(logged.Event.Cycles, logged.Timestamp); err != nil { return err }
		case EventTypeOutcome:
			if logged.Outcome == nil { return errors.New("outcome event has no outcome payload") }
			if _, err := brainRuntime.ObserveOutcome(*logged.Outcome); err != nil { return err }
		case EventTypeConsequence:
			if logged.Consequence == nil { return errors.New("consequence event has no consequence payload") }
			if _, err := brainRuntime.RecordInquiryConsequenceEvent(*logged.Consequence, logged.Timestamp); err != nil { return err }
		default:
			return fmt.Errorf("unknown event type %q", logged.Type)
		}
	}
	return nil
}
