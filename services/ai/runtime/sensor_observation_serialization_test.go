package runtime

import (
	"bytes"
	"fmt"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// A sensor sample and ordinary Process events must acquire the same runtime
// mutation boundary. Replaying the persisted order must reproduce the neural
// substrate and activation state even when callers start concurrently.
func TestConcurrentSensorAndProcessEventsReplayDeterministically(t *testing.T) {
	path := filepath.Join(t.TempDir(), "interleaved-sensor-events.jsonl")
	log, err := OpenEventLog(path)
	if err != nil {
		t.Fatalf("open event log: %v", err)
	}
	defer log.Close()

	at := time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)
	original := NewBrainRuntime(knowledge.NewBootstrapBrain())
	original.SetClock(NewFixedClock(at))
	original.SetEventLog(log)
	orchestrator := NewCognitiveOrchestrator(original)

	const ordinaryEvents = 8
	start := make(chan struct{})
	failures := make(chan error, ordinaryEvents+1)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-start
		sensor := testSensorObservation()
		sensor.SampleID = "camera-0:concurrent"
		sensor.Timestamp = at
		_, err := orchestrator.ProcessSensorObservation(sensor)
		if err != nil {
			failures <- fmt.Errorf("sensor: %w", err)
		}
	}()

	for i := 0; i < ordinaryEvents; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			_, _, err := original.Process(Event{
				ID: fmt.Sprintf("ordinary:%d", i),
				Cycles: 1,
				Timestamp: at,
			})
			if err != nil {
				failures <- fmt.Errorf("ordinary %d: %w", i, err)
			}
		}(i)
	}
	close(start)
	wg.Wait()
	close(failures)
	for err := range failures {
		t.Error(err)
	}
	if t.Failed() {
		t.FailNow()
	}

	events, err := ReadEventLog(path)
	if err != nil {
		t.Fatalf("read event log: %v", err)
	}
	if len(events) != ordinaryEvents+1 {
		t.Fatalf("event count: got %d want %d", len(events), ordinaryEvents+1)
	}
	var sensors int
	for i, event := range events {
		if event.Sequence != uint64(i+1) {
			t.Fatalf("noncontiguous sequence at %d: got %d", i, event.Sequence)
		}
		if event.Type == EventTypeProcess && event.Event != nil && event.Event.SensorObservation != nil {
			sensors++
		}
	}
	if sensors != 1 {
		t.Fatalf("persisted sensor events: got %d want 1", sensors)
	}

	replayed := NewBrainRuntime(knowledge.NewBootstrapBrain())
	replayed.SetClock(NewFixedClock(at))
	if err := ReplayEventLog(replayed, events); err != nil {
		t.Fatalf("replay concurrent event stream: %v", err)
	}
	originalPath := filepath.Join(t.TempDir(), "original-checkpoint.json")
	replayedPath := filepath.Join(t.TempDir(), "replayed-checkpoint.json")
	if err := original.Checkpoint(originalPath); err != nil {
		t.Fatalf("checkpoint original: %v", err)
	}
	if err := replayed.Checkpoint(replayedPath); err != nil {
		t.Fatalf("checkpoint replayed: %v", err)
	}
	originalSnapshot, err := readCheckpointForTest(originalPath)
	if err != nil {
		t.Fatalf("read original checkpoint: %v", err)
	}
	replayedSnapshot, err := readCheckpointForTest(replayedPath)
	if err != nil {
		t.Fatalf("read replayed checkpoint: %v", err)
	}
	if !bytes.Equal(originalSnapshot.Brain, replayedSnapshot.Brain) {
		t.Fatalf("concurrent replay diverged from canonical substrate: original_bytes=%d replayed_bytes=%d", len(originalSnapshot.Brain), len(replayedSnapshot.Brain))
	}
	if !reflect.DeepEqual(originalSnapshot.Activation, replayedSnapshot.Activation) {
		t.Fatal("concurrent replay diverged from activation state")
	}
}
