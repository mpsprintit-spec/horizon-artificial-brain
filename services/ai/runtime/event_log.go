package runtime

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
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
}

func OpenEventLog(path string) (*EventLog, error) {
	if path == "" { return nil, errors.New("event log path is empty") }
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil { return nil, fmt.Errorf("open event log: %w", err) }
	return &EventLog{file: file}, nil
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
	if l == nil || l.file == nil { return nil }
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.file.Close()
}

func ReadEventLog(path string) ([]LoggedEvent, error) {
	file, err := os.Open(path)
	if err != nil { return nil, err }
	defer file.Close()
	var events []LoggedEvent
	var expected uint64 = 1
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var event LoggedEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil { return nil, fmt.Errorf("decode event log: %w", err) }
		if event.SchemaVersion != EventLogSchemaVersion { return nil, fmt.Errorf("unsupported event log schema version %d", event.SchemaVersion) }
		if event.BrainIdentity != BrainIdentity { return nil, fmt.Errorf("event belongs to brain %q", event.BrainIdentity) }
		if event.Sequence != expected { return nil, fmt.Errorf("event sequence discontinuity: got %d want %d", event.Sequence, expected) }
		expected++
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil { return nil, fmt.Errorf("read event log: %w", err) }
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
