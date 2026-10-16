package runtime

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

func sensorJournalTestIntent() LoggedEvent {
	sensor := testSensorObservation()
	return LoggedEvent{
		SchemaVersion: EventLogSchemaVersion,
		BrainIdentity: BrainIdentity,
		Sequence: 1,
		Type: EventTypeSensorIntent,
		Timestamp: sensor.Timestamp,
		Event: &Event{
			ID: sensor.SampleID, Source: sensor.Source,
			Modality: sensor.Modality, Timestamp: sensor.Timestamp,
			Cycles: 1, SensorObservation: &sensor,
		},
	}
}

func TestSensorIntentAndCommitAreOneLogicalReplayEvent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sensor-journal.jsonl")
	log, err := OpenEventLog(path)
	if err != nil { t.Fatal(err) }
	defer log.Close()

	r := NewBrainRuntime(knowledge.NewBootstrapBrain())
	r.SetEventLog(log)
	if _, err := NewCognitiveOrchestrator(r).ProcessSensorObservation(testSensorObservation()); err != nil {
		t.Fatalf("process journaled observation: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil { t.Fatal(err) }
	if got := bytes.Count(raw, []byte{'\n'}); got != 2 {
		t.Fatalf("physical journal records: got %d want 2", got)
	}
	if !bytes.Contains(raw, []byte("\"sensor_intent\"")) || !bytes.Contains(raw, []byte("\"sensor_commit\"")) {
		t.Fatalf("intent and commit markers were not both persisted: %s", raw)
	}
	events, err := ReadEventLog(path)
	if err != nil { t.Fatalf("read committed journal: %v", err) }
	if len(events) != 1 || events[0].Sequence != 1 || events[0].Type != EventTypeProcess ||
		events[0].Event == nil || events[0].Event.SensorObservation == nil {
		t.Fatalf("expected one replayable sensor process event, got %+v", events)
	}
	replayed := NewBrainRuntime(knowledge.NewBootstrapBrain())
	if err := ReplayEventLog(replayed, events); err != nil { t.Fatalf("replay committed sensor: %v", err) }
	if replayed.LastSequence() != r.LastSequence() {
		t.Fatalf("sequence differs after replay: %d vs %d", replayed.LastSequence(), r.LastSequence())
	}
}

func TestReadEventLogRejectsIncompleteAndMismatchedSensorJournal(t *testing.T) {
	for _, scenario := range []struct {
		name string
		commit bool
		hash string
	}{
		{name: "orphan-intent"},
		{name: "wrong-commit-hash", commit: true, hash: "wrong"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "incomplete.jsonl")
			log, err := OpenEventLog(path)
			if err != nil { t.Fatal(err) }
			defer log.Close()
			intent := sensorJournalTestIntent()
			if err := log.Append(intent); err != nil { t.Fatal(err) }
			if scenario.commit {
				if err := log.Append(LoggedEvent{
					SchemaVersion: EventLogSchemaVersion,
					BrainIdentity: BrainIdentity,
					Sequence: 1, Type: EventTypeSensorCommit,
					Timestamp: intent.Timestamp, IntentHash: scenario.hash,
				}); err != nil { t.Fatal(err) }
			}
			_, err = ReadEventLog(path)
			if !errors.Is(err, ErrUncommittedSensorTransaction) {
				t.Fatalf("expected unresolved journal, got %v", err)
			}
		})
	}
}

func TestSensorCommitFailureRequiresRecoveryAndRejectsFurtherMutations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "commit-failure.jsonl")
	log, err := OpenEventLog(path)
	if err != nil { t.Fatal(err) }
	defer log.Close()
	log.beforeAppend = func(event LoggedEvent) error {
		if event.Type == EventTypeSensorCommit { return errors.New("commit sync unavailable") }
		return nil
	}
	brain := knowledge.NewBootstrapBrain()
	r := NewBrainRuntime(brain)
	r.SetEventLog(log)
	nodesBefore := len(brain.Registry.Nodes())
	_, err = NewCognitiveOrchestrator(r).ProcessSensorObservation(testSensorObservation())
	if !errors.Is(err, ErrRuntimeRecoveryRequired) {
		t.Fatalf("expected poisoned runtime on commit failure, got %v", err)
	}
	if !r.RecoveryRequired() { t.Fatal("runtime was not marked recovery-required") }
	if got := len(brain.Registry.Nodes()); got <= nodesBefore {
		t.Fatalf("fault injection did not reach post-projection commit: before=%d after=%d", nodesBefore, got)
	}
	if _, err := ReadEventLog(path); !errors.Is(err, ErrUncommittedSensorTransaction) {
		t.Fatalf("uncommitted intent must not be replayed as success: %v", err)
	}
	if _, _, err := r.Process(Event{ID:"after-failed-commit"}); !errors.Is(err, ErrRuntimeRecoveryRequired) {
		t.Fatalf("Process allowed mutation after failed commit: %v", err)
	}
	if _, _, err := r.ThinkAt(1, time.Time{}); !errors.Is(err, ErrRuntimeRecoveryRequired) {
		t.Fatalf("ThinkAt allowed mutation after failed commit: %v", err)
	}
	if _, err := r.CognitiveThink(1); !errors.Is(err, ErrRuntimeRecoveryRequired) {
		t.Fatalf("CognitiveThink allowed mutation after failed commit: %v", err)
	}
	if _, err := r.LearnExperience(learning.Experience{}, time.Time{}); !errors.Is(err, ErrRuntimeRecoveryRequired) {
		t.Fatalf("LearnExperience allowed mutation after failed commit: %v", err)
	}
	next := testSensorObservation()
	next.SampleID = "camera-0:next"
	next.Timestamp = next.Timestamp.Add(time.Second)
	if _, err := NewCognitiveOrchestrator(r).ProcessSensorObservation(next); !errors.Is(err, ErrRuntimeRecoveryRequired) {
		t.Fatalf("second sensor observation allowed after failed commit: %v", err)
	}
}

func TestSensorIntentAppendFailureRejectsProjectionAndFurtherWrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "intent-failure.jsonl")
	log, err := OpenEventLog(path)
	if err != nil { t.Fatal(err) }
	defer log.Close()
	log.beforeAppend = func(event LoggedEvent) error {
		if event.Type == EventTypeSensorIntent { return fmt.Errorf("intent write failed") }
		return nil
	}
	brain := knowledge.NewBootstrapBrain()
	r := NewBrainRuntime(brain)
	r.SetEventLog(log)
	before := len(brain.Registry.Nodes())
	_, err = NewCognitiveOrchestrator(r).ProcessSensorObservation(testSensorObservation())
	if !errors.Is(err, ErrRuntimeRecoveryRequired) {
		t.Fatalf("expected recovery-required failure, got %v", err)
	}
	if got := len(brain.Registry.Nodes()); got != before {
		t.Fatalf("intent failure still projected sensor: before=%d after=%d", before, got)
	}
	if r.LastSequence() != 0 { t.Fatalf("intent failure advanced sequence: %d", r.LastSequence()) }
	if _, _, err := r.Process(Event{ID:"subsequent"}); !errors.Is(err, ErrRuntimeRecoveryRequired) {
		t.Fatalf("runtime accepted a later mutation: %v", err)
	}
}
