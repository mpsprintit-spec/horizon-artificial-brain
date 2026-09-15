package runtime

import (
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestGate7RestartRecoveryPreservesNeuralAndRecurrentState(t *testing.T) {
	clock := NewFixedClock(time.Date(2026, 9, 15, 14, 0, 0, 0, time.UTC))
	original := NewBrainRuntime(nil)
	original.SetClock(clock)

	if _, err := original.LearnExperience(testExperience(), clock.Now()); err != nil {
		t.Fatalf("learn: %v", err)
	}
	if _, _, err := original.Process(Event{ID: "restart-process", Stimulus: []string{"saya"}, Cycles: 2, Timestamp: clock.Now()}); err != nil {
		t.Fatalf("process: %v", err)
	}
	clock.Set(clock.Now().Add(time.Second))
	if _, _, err := original.ThinkAt(2, clock.Now()); err != nil {
		t.Fatalf("think: %v", err)
	}

	path := filepath.Join(t.TempDir(), "checkpoint.json")
	if err := original.Checkpoint(path); err != nil {
		t.Fatalf("checkpoint: %v", err)
	}
	checkpointSequence := original.LastSequence()

	resumed := NewBrainRuntime(nil)
	resumed.SetClock(clock)
	if err := resumed.RestoreCheckpoint(path); err != nil {
		t.Fatalf("restore: %v", err)
	}
	if resumed.LastSequence() != checkpointSequence {
		t.Fatalf("restored sequence: got %d want %d", resumed.LastSequence(), checkpointSequence)
	}

	clock.Set(clock.Now().Add(time.Second))
	originalResult, originalSeq, err := original.ThinkAt(2, clock.Now())
	if err != nil {
		t.Fatalf("original continuation: %v", err)
	}
	resumedResult, resumedSeq, err := resumed.ThinkAt(2, clock.Now())
	if err != nil {
		t.Fatalf("resumed continuation: %v", err)
	}
	if originalSeq != resumedSeq {
		t.Fatalf("continuation sequence: got %d want %d", resumedSeq, originalSeq)
	}
	if !reflect.DeepEqual(originalResult.Activations, resumedResult.Activations) ||
		!reflect.DeepEqual(originalResult.Confidence, resumedResult.Confidence) ||
		!reflect.DeepEqual(originalResult.Prediction.State, resumedResult.Prediction.State) ||
		!reflect.DeepEqual(originalResult.Prediction.Confidence, resumedResult.Prediction.Confidence) ||
		originalResult.PredictionError != resumedResult.PredictionError {
		t.Fatal("restart recovery changed recurrent continuation")
	}
}

func TestGate7CheckpointSequenceMatchesAcceptedEvents(t *testing.T) {
	clock := NewFixedClock(time.Date(2026, 9, 15, 15, 0, 0, 0, time.UTC))
	r := NewBrainRuntime(nil)
	r.SetClock(clock)

	if _, err := r.LearnExperience(testExperience(), clock.Now()); err != nil {
		t.Fatalf("learn: %v", err)
	}
	if _, _, err := r.Process(Event{ID: "sequence-process", Stimulus: []string{"saya"}, Cycles: 1, Timestamp: clock.Now()}); err != nil {
		t.Fatalf("process: %v", err)
	}
	if _, _, err := r.ThinkAt(1, clock.Now().Add(time.Second)); err != nil {
		t.Fatalf("think: %v", err)
	}

	if got := r.LastSequence(); got != 3 {
		t.Fatalf("runtime sequence: got %d want 3", got)
	}
	path := filepath.Join(t.TempDir(), "checkpoint.json")
	if err := r.Checkpoint(path); err != nil {
		t.Fatalf("checkpoint: %v", err)
	}

	resumed := NewBrainRuntime(nil)
	if err := resumed.RestoreCheckpoint(path); err != nil {
		t.Fatalf("restore: %v", err)
	}
	if got := resumed.LastSequence(); got != 3 {
		t.Fatalf("checkpoint sequence: got %d want 3", got)
	}
}
