package runtime

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
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
}

const (
	EventTypeProcess = "process"
	EventTypeLearn = "learn"
	EventTypeThink = "think"
	EventTypeOutcome = "outcome"
	EventTypeConsequence = "consequence"
)

type EventLog struct {
	mu sync.Mutex
	file *os.File
	lockFile *os.File
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

func ReadEventLog(path string) ([]LoggedEvent, error) {
	file, err := os.Open(path)
	if err != nil { return nil, err }
	defer file.Close()

	var events []LoggedEvent
	var expected uint64 = 1
	seen := make(map[uint64][]byte)
	scanner := bufio.NewScanner(file)
	// Event payloads can exceed Scanner's small default token limit as the
	// runtime evolves; allow records up to 16 MiB while still bounding memory.
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

		if event.Sequence < expected {
			// Repeated records may be separated by newer records after a retry
			// or interrupted append. Skip only if the sequence and payload are
			// identical to the first occurrence. Conflicting duplicates remain
			// fatal so we never silently choose between different brain events.
			prior, exists := seen[event.Sequence]
			if exists && bytes.Equal(canonical, prior) {
				continue
			}
			if exists {
				return nil, fmt.Errorf("conflicting duplicate event sequence %d at line %d", event.Sequence, lineNumber)
			}
			return nil, fmt.Errorf("event sequence %d at line %d is older than expected %d but has no prior record", event.Sequence, lineNumber, expected)
		}
		if event.Sequence > expected {
			return nil, fmt.Errorf("event sequence discontinuity at line %d: got %d want %d", lineNumber, event.Sequence, expected)
		}

		seen[event.Sequence] = canonical
		events = append(events, event)
		expected++
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read event log: %w", err)
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
			if _, _, err := brainRuntime.Process(*logged.Event); err != nil { return err }
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
