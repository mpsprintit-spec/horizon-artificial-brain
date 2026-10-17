package runtime

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func ownerWithJournal(t *testing.T) (*RuntimeOwner, *BrainRuntime, *EventLog, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "events.jsonl")
	log, err := OpenEventLog(path)
	if err != nil { t.Fatal(err) }
	t.Cleanup(func(){ _ = log.Close() })
	initial := NewBrainRuntime(knowledge.NewBootstrapBrain())
	initial.SetEventLog(log)
	owner, err := NewRuntimeOwner(initial, log)
	if err != nil { t.Fatal(err) }
	return owner, initial, log, path
}

func TestRuntimeOwnerRecoveryWaitsForInflightOperation(t *testing.T) {
	owner, initial, _, _ := ownerWithJournal(t)
	at := time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)
	first := checkpointRecoverySensor("owner:first", at, 0.22)
	if err := owner.WithRuntime(func(r *BrainRuntime) error {
		_, err := NewCognitiveOrchestrator(r).ProcessSensorObservation(first)
		return err
	}); err != nil { t.Fatal(err) }
	checkpointPath := filepath.Join(t.TempDir(), "checkpoint.json")
	if err := owner.WithRuntime(func(r *BrainRuntime) error { return r.Checkpoint(checkpointPath) }); err != nil { t.Fatal(err) }

	entered := make(chan struct{})
	release := make(chan struct{})
	doneOperation := make(chan error, 1)
	go func() {
		doneOperation <- owner.WithRuntime(func(r *BrainRuntime) error {
			close(entered)
			<-release
			_, _, err := r.Process(Event{ID: "in-flight", Cycles: 1, Timestamp: at.Add(time.Second)})
			return err
		})
	}()
	select {
	case <-entered:
	case <-time.After(2*time.Second): t.Fatal("routed operation did not enter")
	}

	doneRecovery := make(chan error, 1)
	go func() { doneRecovery <- owner.RecoverFromCheckpoint(checkpointPath) }()
	select {
	case err := <-doneRecovery:
		t.Fatalf("recovery bypassed active routed operation: %v", err)
	case <-time.After(30*time.Millisecond):
	}
	close(release)
	if err := <-doneOperation; err != nil { t.Fatalf("routed operation: %v", err) }
	select {
	case err := <-doneRecovery:
		if err != nil { t.Fatalf("handover: %v", err) }
	case <-time.After(5*time.Second):
		t.Fatal("handover did not complete")
	}

	if !initial.Retired() { t.Fatal("old runtime was not retired") }
	if got := owner.Generation(); got != 2 { t.Fatalf("handover generation: %d", got) }
	if _, _, err := initial.Process(Event{ID:"retired"}); !errors.Is(err, ErrRuntimeRetired) {
		t.Fatalf("retired runtime accepted process: %v", err)
	}
	if _, err := initial.CognitiveThink(1); !errors.Is(err, ErrRuntimeRetired) {
		t.Fatalf("retired runtime accepted cognition: %v", err)
	}
	if err := owner.WithRuntime(func(r *BrainRuntime) error {
		if r == initial || r.LastSequence() != 2 {
			t.Fatalf("owner served old runtime or lost committed event: old=%t sequence=%d", r == initial, r.LastSequence())
		}
		return nil
	}); err != nil { t.Fatal(err) }
}

func TestRuntimeOwnerFailedRecoveryDoesNotRetireActiveRuntime(t *testing.T) {
	owner, initial, _, _ := ownerWithJournal(t)
	oldGeneration := owner.Generation()
	err := owner.RecoverFromCheckpoint(filepath.Join(t.TempDir(), "missing.json"))
	if err == nil { t.Fatal("expected missing checkpoint failure") }
	if owner.Generation() != oldGeneration || initial.Retired() {
		t.Fatal("failed recovery still changed the active generation")
	}
	if err := owner.WithRuntime(func(r *BrainRuntime) error {
		if r != initial { t.Fatal("failed recovery replaced active runtime") }
		_, _, err := r.Process(Event{ID:"after-failed-restore", Cycles: 1})
		return err
	}); err != nil { t.Fatalf("healthy runtime was blocked after failed recovery: %v", err) }
}

func TestRuntimeOwnerRepairsOrphanAndAtomicallyHandover(t *testing.T) {
	owner, initial, log, path := ownerWithJournal(t)
	at := time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)
	checkpointPath := filepath.Join(t.TempDir(), "checkpoint.json")
	if err := owner.WithRuntime(func(r *BrainRuntime) error {
		_, err := NewCognitiveOrchestrator(r).ProcessSensorObservation(checkpointRecoverySensor("owner:first", at, 0.25))
		if err != nil { return err }
		return r.Checkpoint(checkpointPath)
	}); err != nil { t.Fatal(err) }

	log.beforeAppend = func(e LoggedEvent) error {
		if e.Type == EventTypeSensorCommit { return errors.New("simulated commit sync failure") }
		return nil
	}
	fail := owner.WithRuntime(func(r *BrainRuntime) error {
		_, err := NewCognitiveOrchestrator(r).ProcessSensorObservation(checkpointRecoverySensor("owner:orphan", at.Add(time.Second), 0.5))
		return err
	})
	if !errors.Is(fail, ErrRuntimeRecoveryRequired) { t.Fatalf("expected recovery-required failure: %v", fail) }
	log.beforeAppend = nil

	changed := make(chan struct{})
	if err := owner.WithRuntimeGeneration(func(_ *BrainRuntime, change <-chan struct{}) error {
		go func(){ <-change; close(changed) }()
		return nil
	}); err != nil { t.Fatal(err) }
	backup, err := owner.RecoverTrailingIntentFromCheckpoint(checkpointPath)
	if err != nil { t.Fatalf("recover from orphan: %v", err) }
	if _, err := os.Stat(backup); err != nil { t.Fatalf("missing recovered-log backup: %v", err) }
	select {
	case <-changed:
	case <-time.After(2*time.Second): t.Fatal("handover did not invalidate previous generation")
	}
	if !initial.Retired() { t.Fatal("poisoned old runtime was not retired") }
	if err := owner.WithRuntime(func(r *BrainRuntime) error {
		if r == initial || r.LastSequence() != 1 { t.Fatal("owner did not expose committed recovered state") }
		_, err := NewCognitiveOrchestrator(r).ProcessSensorObservation(checkpointRecoverySensor("owner:continued", at.Add(2*time.Second), -0.4))
		return err
	}); err != nil { t.Fatalf("post-handover mutation: %v", err) }
	events, err := ReadEventLog(path)
	if err != nil || len(events) != 2 { t.Fatalf("post-handover journal invalid: count=%d err=%v", len(events), err) }
	if _, _, err := initial.ThinkAt(1, at.Add(3*time.Second)); !errors.Is(err, ErrRuntimeRetired) {
		t.Fatalf("retired poisoned runtime continued thinking: %v", err)
	}
}

func TestRuntimeOwnerRejectsWrongJournalAndRetiredRuntime(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "events.jsonl")
	log, err := OpenEventLog(logPath)
	if err != nil { t.Fatal(err) }
	defer log.Close()
	r := NewBrainRuntime(knowledge.NewBootstrapBrain())
	if _, err := NewRuntimeOwner(r, log); err == nil {
		t.Fatal("owner accepted a runtime with a different journal")
	}
	r.SetEventLog(log)
	owner, err := NewRuntimeOwner(r, log)
	if err != nil { t.Fatal(err) }
	if err := owner.WithRuntime(nil); err == nil { t.Fatal("owner allowed nil callback") }
	if err := owner.WithRuntime(func(_ *BrainRuntime) error { return nil }); err != nil { t.Fatal(err) }
	if err := owner.RecoverFromCheckpoint(""); err == nil {
		t.Fatal("empty checkpoint should fail")
	}
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := owner.WithRuntime(func(r *BrainRuntime) error { _ = r.LastSequence(); return nil }); err != nil {
				t.Errorf("concurrent owner routing failed: %v", err)
			}
		}()
	}
	wg.Wait()
}
