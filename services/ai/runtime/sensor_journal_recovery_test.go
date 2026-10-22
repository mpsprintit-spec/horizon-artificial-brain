package runtime

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestRecoverTrailingSensorIntentPreservesCommittedPrefix(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sensor-events.jsonl")
	log, err := OpenEventLog(path)
	if err != nil { t.Fatal(err) }
	defer log.Close()

	ordinary := LoggedEvent{
		SchemaVersion: EventLogSchemaVersion, BrainIdentity: BrainIdentity,
		Sequence: 1, Type: EventTypeProcess, Timestamp: time.Date(2026, 10, 12, 8, 0, 0, 0, time.UTC),
		Event: &Event{ID: "ordinary:1", Cycles: 1},
	}
	if err := log.Append(ordinary); err != nil { t.Fatal(err) }
	intent := sensorJournalTestIntent()
	intent.Sequence = 2
	if err := log.Append(intent); err != nil { t.Fatal(err) }
	before, err := os.ReadFile(path)
	if err != nil { t.Fatal(err) }
	if _, err := ReadEventLog(path); !errors.Is(err, ErrUncommittedSensorTransaction) {
		t.Fatalf("orphan intent should fail strict read: %v", err)
	}

	backupPath, err := log.RecoverTrailingSensorIntent()
	if err != nil { t.Fatalf("recover trailing intent: %v", err) }
	backup, err := os.ReadFile(backupPath)
	if err != nil { t.Fatal(err) }
	if !bytes.Equal(backup, before) { t.Fatal("backup differs from original journal") }
	events, err := ReadEventLog(path)
	if err != nil { t.Fatalf("read recovered journal: %v", err) }
	if len(events) != 1 || events[0].Sequence != 1 || events[0].Event.ID != "ordinary:1" {
		t.Fatalf("committed prefix was not preserved: %+v", events)
	}
	// The exclusive writer can safely continue from the committed sequence.
	if err := log.Append(LoggedEvent{
		SchemaVersion: EventLogSchemaVersion, BrainIdentity: BrainIdentity,
		Sequence: 2, Type: EventTypeThink, Timestamp: ordinary.Timestamp,
		Event: &Event{Cycles: 1},
	}); err != nil { t.Fatalf("append after recovery: %v", err) }
	events, err = ReadEventLog(path)
	if err != nil || len(events) != 2 { t.Fatalf("journal unusable after recovery: %v, events=%d", err, len(events)) }
}

func TestRecoverTrailingSensorIntentRejectsCommittedOrMalformedJournal(t *testing.T) {
	for _, tc := range []struct{
		name string
		write func(*testing.T, *EventLog)
	}{
		{"committed-record", func(t *testing.T, log *EventLog) {
			if err := log.Append(LoggedEvent{
				SchemaVersion: EventLogSchemaVersion, BrainIdentity: BrainIdentity,
				Sequence: 1, Type: EventTypeThink, Event: &Event{Cycles: 1},
			}); err != nil { t.Fatal(err) }
		}},
		{"vector-smuggling", func(t *testing.T, log *EventLog) {
			intent := sensorJournalTestIntent()
			vector := knowledge.NewNeuralVector([]float64{0.3, -0.2})
			intent.Vector = &vector
			if err := log.Append(intent); err != nil { t.Fatal(err) }
		}},
		{"wrong-sequence", func(t *testing.T, log *EventLog) {
			intent := sensorJournalTestIntent()
			intent.Sequence = 3
			if err := log.Append(intent); err != nil { t.Fatal(err) }
		}},
		{"mismatched-commit", func(t *testing.T, log *EventLog) {
			intent := sensorJournalTestIntent()
			if err := log.Append(intent); err != nil { t.Fatal(err) }
			if err := log.Append(LoggedEvent{
				SchemaVersion: EventLogSchemaVersion, BrainIdentity: BrainIdentity,
				Sequence: 1, Type: EventTypeSensorCommit, Timestamp: intent.Timestamp,
				IntentHash: "invalid",
			}); err != nil { t.Fatal(err) }
		}},
		{"interrupted-tail", func(t *testing.T, log *EventLog) {
			if _, err := log.file.Write([]byte("{\"type\":")); err != nil { t.Fatal(err) }
			if err := log.file.Sync(); err != nil { t.Fatal(err) }
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "events.jsonl")
			log, err := OpenEventLog(path)
			if err != nil { t.Fatal(err) }
			defer log.Close()
			tc.write(t, log)
			before, err := os.ReadFile(path)
			if err != nil { t.Fatal(err) }
			if _, err := log.RecoverTrailingSensorIntent(); err == nil {
				t.Fatal("unsafe journal recovery unexpectedly succeeded")
			}
			after, err := os.ReadFile(path)
			if err != nil { t.Fatal(err) }
			if !bytes.Equal(before, after) { t.Fatal("rejected recovery modified the original journal") }
		})
	}
}

func TestRecoverTrailingSensorIntentOnlyUnderExclusiveWriter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	log, err := OpenEventLog(path)
	if err != nil { t.Fatal(err) }
	if err := log.Append(sensorJournalTestIntent()); err != nil { t.Fatal(err) }
	if err := log.Close(); err != nil { t.Fatal(err) }
	if _, err := log.RecoverTrailingSensorIntent(); err == nil {
		t.Fatal("closed writer was allowed to recover journal")
	}
}
