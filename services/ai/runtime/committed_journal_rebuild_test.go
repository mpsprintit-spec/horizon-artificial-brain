package runtime

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestRebuildRuntimeAfterFailedSensorCommitReproducesCommittedState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	log, err := OpenEventLog(path)
	if err != nil { t.Fatal(err) }
	defer log.Close()
	start := time.Date(2026, 10, 12, 8, 0, 0, 0, time.UTC)
	original := NewBrainRuntime(knowledge.NewBootstrapBrain())
	original.SetClock(NewFixedClock(start))
	original.SetEventLog(log)
	orchestrator := NewCognitiveOrchestrator(original)

	first := testSensorObservation()
	first.SampleID = "camera-0:rebuild-1"
	first.Timestamp = start
	if _, err := orchestrator.ProcessSensorObservation(first); err != nil {
		t.Fatalf("first observation: %v", err)
	}
	if _, _, err := original.Process(Event{
		ID: "ordinary:between", Cycles: 1, Timestamp: start.Add(50 * time.Millisecond),
	}); err != nil {
		t.Fatalf("interleaved process: %v", err)
	}
	second := testSensorObservation()
	second.SampleID = "camera-0:rebuild-2"
	second.Timestamp = start.Add(100 * time.Millisecond)
	second.ChannelStates[0] = -0.75
	if _, err := orchestrator.ProcessSensorObservation(second); err != nil {
		t.Fatalf("second observation: %v", err)
	}

	// A checkpoint taken before the injected failure is an independent
	// reference for the expected committed substrate and recurrent state.
	beforePath := filepath.Join(t.TempDir(), "before-failure.json")
	if err := original.Checkpoint(beforePath); err != nil { t.Fatal(err) }
	before, err := readCheckpointForTest(beforePath)
	if err != nil { t.Fatal(err) }
	sequence := original.LastSequence()
	if sequence != 3 { t.Fatalf("unexpected committed sequence %d", sequence) }

	// Inject only a sensor commit failure; projection and cognition must have
	// run, so the old runtime must be abandoned rather than reused.
	log.beforeAppend = func(event LoggedEvent) error {
		if event.Type == EventTypeSensorCommit { return errors.New("commit persistence failed") }
		return nil
	}
	third := testSensorObservation()
	third.SampleID = "camera-0:uncommitted"
	third.Timestamp = start.Add(200 * time.Millisecond)
	if _, err := orchestrator.ProcessSensorObservation(third); !errors.Is(err, ErrRuntimeRecoveryRequired) {
		t.Fatalf("expected poisoned runtime after commit failure: %v", err)
	}
	log.beforeAppend = nil
	if !original.RecoveryRequired() { t.Fatal("original runtime was not poisoned") }
	if original.LastSequence() != sequence+1 { t.Fatalf("fault did not reach post-process state") }

	factory := func() *knowledge.Brain { return knowledge.NewBootstrapBrain() }
	if rebuilt, err := RebuildRuntimeFromCommittedJournal(log, factory); !errors.Is(err, ErrUncommittedSensorTransaction) || rebuilt != nil {
		t.Fatalf("incomplete journal was accepted: runtime=%v err=%v", rebuilt, err)
	}

	backupPath, err := log.RecoverTrailingSensorIntent()
	if err != nil { t.Fatalf("repair orphaned final intent: %v", err) }
	if _, err := os.Stat(backupPath); err != nil { t.Fatalf("missing recovery backup: %v", err) }

	fresh, err := RebuildRuntimeFromCommittedJournal(log, factory)
	if err != nil { t.Fatalf("rebuild from committed journal: %v", err) }
	if fresh == original || fresh.brain == original.brain {
		t.Fatal("recovery reused the potentially mutated runtime or brain")
	}
	if fresh.eventLog != nil { t.Fatal("recovered runtime must not be published with journal attached") }
	if fresh.RecoveryRequired() { t.Fatal("recovered runtime is poisoned") }
	if got := fresh.LastSequence(); got != sequence {
		t.Fatalf("recovered sequence %d, want %d", got, sequence)
	}
	if !fresh.lastObservationAt.Equal(second.Timestamp) {
		t.Fatalf("last committed observation timestamp got %s want %s", fresh.lastObservationAt, second.Timestamp)
	}
	afterPath := filepath.Join(t.TempDir(), "after-rebuild.json")
	if err := fresh.Checkpoint(afterPath); err != nil { t.Fatal(err) }
	after, err := readCheckpointForTest(afterPath)
	if err != nil { t.Fatal(err) }
	if !bytes.Equal(before.Brain, after.Brain) {
		t.Fatalf("rebuild brain differs from pre-failure checkpoint: bytes=%d/%d", len(before.Brain), len(after.Brain))
	}
	if !reflect.DeepEqual(before.Activation, after.Activation) {
		t.Fatal("rebuild activation differs from pre-failure checkpoint")
	}

	// Publish only the new runtime, then attach the existing exclusive writer
	// for live continuation; the old instance must remain poisoned.
	fresh.SetEventLog(log)
	next := testSensorObservation()
	next.SampleID = "camera-0:after-rebuild"
	next.Timestamp = start.Add(300 * time.Millisecond)
	if _, err := NewCognitiveOrchestrator(fresh).ProcessSensorObservation(next); err != nil {
		t.Fatalf("recovered runtime cannot continue: %v", err)
	}
	events, err := ReadEventLog(path)
	if err != nil { t.Fatalf("read continued journal: %v", err) }
	if len(events) != int(sequence+1) || events[len(events)-1].Event.SensorObservation.SampleID != next.SampleID {
		t.Fatalf("continued journal lost transaction or sequence: %+v", events)
	}
	if _, _, err := original.Process(Event{ID:"old-runtime-after-recovery"}); !errors.Is(err, ErrRuntimeRecoveryRequired) {
		t.Fatalf("poisoned runtime became writable: %v", err)
	}
}

func TestRebuildRuntimeRejectsInvalidJournalWithoutPublishing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	log, err := OpenEventLog(path)
	if err != nil { t.Fatal(err) }
	defer log.Close()
	if err := log.Append(LoggedEvent{
		SchemaVersion: EventLogSchemaVersion, BrainIdentity: BrainIdentity,
		Sequence: 1, Type: "unsupported-replay-event", Timestamp: time.Now().UTC(),
	}); err != nil { t.Fatal(err) }
	if got, err := RebuildRuntimeFromCommittedJournal(log, func() *knowledge.Brain { return knowledge.NewBootstrapBrain() }); err == nil || got != nil {
		t.Fatalf("invalid replay was published: runtime=%v err=%v", got, err)
	}
}

func TestRebuildRuntimeRequiresExplicitFoundationAndExclusiveWriter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	log, err := OpenEventLog(path)
	if err != nil { t.Fatal(err) }
	if rebuilt, err := RebuildRuntimeFromCommittedJournal(log, nil); err == nil || rebuilt != nil {
		t.Fatal("missing foundation factory was accepted")
	}
	if rebuilt, err := RebuildRuntimeFromCommittedJournal(log, func() *knowledge.Brain { return nil }); err == nil || rebuilt != nil {
		t.Fatal("nil foundation was accepted")
	}
	if err := log.Close(); err != nil { t.Fatal(err) }
	if rebuilt, err := RebuildRuntimeFromCommittedJournal(log, func() *knowledge.Brain { return knowledge.NewBootstrapBrain() }); err == nil || rebuilt != nil {
		t.Fatal("closed writer was accepted")
	}
}
