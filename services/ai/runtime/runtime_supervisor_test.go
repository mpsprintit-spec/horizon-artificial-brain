package runtime

import (
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestRuntimeSupervisorHandoverPreservesCanonicalContinuity(t *testing.T) {
	log, err := OpenEventLog(filepath.Join(t.TempDir(), "events.jsonl"))
	if err != nil { t.Fatal(err) }
	defer log.Close()
	at := time.Date(2026, 10, 12, 8, 0, 0, 0, time.UTC)
	original := NewBrainRuntime(knowledge.NewBootstrapBrain())
	original.SetEventLog(log)
	supervisor, err := NewRuntimeSupervisor(original)
	if err != nil { t.Fatal(err) }
	checkpoint := filepath.Join(t.TempDir(), "checkpoint.json")
	first := checkpointRecoverySensor("handover:a", at, 0.2)
	second := checkpointRecoverySensor("handover:b", at.Add(time.Second), 0.3)
	third := checkpointRecoverySensor("handover:c", at.Add(2*time.Second), 0.4)
	if err := supervisor.With(func(r *BrainRuntime) error {
		_, err := NewCognitiveOrchestrator(r).ProcessSensorObservation(first)
		return err
	}); err != nil { t.Fatal(err) }
	if err := original.Checkpoint(checkpoint); err != nil { t.Fatal(err) }
	if err := supervisor.With(func(r *BrainRuntime) error {
		_, err := NewCognitiveOrchestrator(r).ProcessSensorObservation(second)
		return err
	}); err != nil { t.Fatal(err) }
	if err := supervisor.RecoverFromCheckpoint(log, checkpoint); err != nil { t.Fatal(err) }
	if !original.RecoveryRequired() { t.Fatal("old runtime not retired") }
	if original.eventLog != nil { t.Fatal("old runtime retained journal") }
	if _, _, err := original.Process(Event{ID:"old-writer", Cycles:1, Timestamp:at}); !errors.Is(err, ErrRuntimeRecoveryRequired) {
		t.Fatalf("old runtime accepted write after handover: %v", err)
	}
	var active *BrainRuntime
	if err := supervisor.With(func(r *BrainRuntime) error {
		active = r
		if r == original || r.brain == original.brain { t.Fatal("supervisor reused retired runtime") }
		if r.LastSequence() != 2 || r.eventLog != log { t.Fatal("new runtime lost journal or sequence") }
		_, err := NewCognitiveOrchestrator(r).ProcessSensorObservation(third)
		return err
	}); err != nil { t.Fatal(err) }
	if active.LastSequence() != 3 { t.Fatalf("new runtime sequence %d", active.LastSequence()) }
	events, err := ReadEventLog(log.file.Name())
	if err != nil { t.Fatal(err) }
	if len(events) != 3 { t.Fatalf("journal has %d committed events, want 3", len(events)) }
	if err := supervisor.Close(); err != nil { t.Fatal(err) }
	if err := supervisor.With(func(*BrainRuntime) error { t.Fatal("closed supervisor accepted operation"); return nil }); !errors.Is(err, ErrRuntimeSupervisorClosed) {
		t.Fatalf("closed supervisor returned %v", err)
	}
	if !active.RecoveryRequired() { t.Fatal("close did not retire active runtime") }
}

func TestRuntimeSupervisorFailedHandoverKeepsOriginal(t *testing.T) {
	log, err := OpenEventLog(filepath.Join(t.TempDir(), "events.jsonl"))
	if err != nil { t.Fatal(err) }
	defer log.Close()
	original := NewBrainRuntime(knowledge.NewBootstrapBrain())
	original.SetEventLog(log)
	supervisor, err := NewRuntimeSupervisor(original)
	if err != nil { t.Fatal(err) }
	if err := supervisor.RecoverFromCheckpoint(log, filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Fatal("missing checkpoint unexpectedly succeeded")
	}
	if original.RecoveryRequired() { t.Fatal("failed handover poisoned original") }
	if err := supervisor.With(func(r *BrainRuntime) error {
		if r != original { t.Fatal("failed handover replaced original") }
		_, err := NewCognitiveOrchestrator(r).ProcessSensorObservation(checkpointRecoverySensor("after-failure", time.Date(2026,10,12,8,0,0,0,time.UTC), 0.3))
		return err
	}); err != nil { t.Fatalf("original unusable after aborted handover: %v", err) }
}

func TestRuntimeSupervisorHandoverWaitsForInflightOperations(t *testing.T) {
	log, err := OpenEventLog(filepath.Join(t.TempDir(), "events.jsonl"))
	if err != nil { t.Fatal(err) }
	defer log.Close()
	at := time.Date(2026,10,12,8,0,0,0,time.UTC)
	original := NewBrainRuntime(knowledge.NewBootstrapBrain())
	original.SetEventLog(log)
	supervisor, err := NewRuntimeSupervisor(original)
	if err != nil { t.Fatal(err) }
	if _, err := NewCognitiveOrchestrator(original).ProcessSensorObservation(checkpointRecoverySensor("initial",at,0.2)); err != nil { t.Fatal(err) }
	checkpoint := filepath.Join(t.TempDir(), "checkpoint.json")
	if err := original.Checkpoint(checkpoint); err != nil { t.Fatal(err) }
	started := make(chan struct{})
	release := make(chan struct{})
	inflightDone := make(chan error,1)
	go func() {
		inflightDone <- supervisor.With(func(r *BrainRuntime) error {
			close(started)
			<-release
			_, err := NewCognitiveOrchestrator(r).ProcessSensorObservation(checkpointRecoverySensor("inflight",at.Add(time.Second),0.3))
			return err
		})
	}()
	<-started
	handoverDone := make(chan error,1)
	go func() { handoverDone <- supervisor.RecoverFromCheckpoint(log,checkpoint) }()
	select {
	case err := <-handoverDone:
		t.Fatalf("handover completed before in-flight operation: %v",err)
	case <-time.After(25*time.Millisecond):
	}
	close(release)
	if err := <-inflightDone; err != nil { t.Fatal(err) }
	if err := <-handoverDone; err != nil { t.Fatal(err) }
	if err := supervisor.With(func(r *BrainRuntime) error {
		if r == original || r.LastSequence() != 2 { t.Fatalf("handover lost inflight committed observation, sequence=%d",r.LastSequence()) }
		return nil
	}); err != nil { t.Fatal(err) }
}

func TestRuntimeSupervisorConcurrentReadersAndHandover(t *testing.T) {
	log, err := OpenEventLog(filepath.Join(t.TempDir(), "events.jsonl"))
	if err != nil { t.Fatal(err) }
	defer log.Close()
	original := NewBrainRuntime(knowledge.NewBootstrapBrain())
	original.SetEventLog(log)
	supervisor, err := NewRuntimeSupervisor(original)
	if err != nil { t.Fatal(err) }
	at := time.Date(2026,10,12,8,0,0,0,time.UTC)
	if _, err := NewCognitiveOrchestrator(original).ProcessSensorObservation(checkpointRecoverySensor("initial",at,0.2)); err != nil { t.Fatal(err) }
	checkpoint := filepath.Join(t.TempDir(), "checkpoint.json")
	if err := original.Checkpoint(checkpoint); err != nil { t.Fatal(err) }
	var wg sync.WaitGroup
	failures := make(chan error,32)
	for i:=0;i<16;i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j:=0;j<8;j++ {
				if err:=supervisor.With(func(r *BrainRuntime) error {
					if r.RecoveryRequired() { return ErrRuntimeRecoveryRequired }
					if r.LastSequence()!=1 { return errors.New("unexpected runtime sequence") }
					return nil
				});err!=nil { failures<-err;return }
			}
		}()
	}
	if err := supervisor.RecoverFromCheckpoint(log,checkpoint); err != nil { t.Fatal(err) }
	wg.Wait()
	close(failures)
	for err:=range failures { t.Fatal(err) }
}
