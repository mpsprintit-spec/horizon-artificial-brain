package runtime

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestCheckpointRestoresOngoingRecurrentState(t *testing.T) {
	clock := NewFixedClock(time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC))
	r := NewBrainRuntime(nil)
	r.SetClock(clock)

	if _, err := r.LearnExperience(testExperience(), clock.Now()); err != nil {
		t.Fatalf("learn experience: %v", err)
	}
	if _, _, err := r.Process(Event{ID: "p1", Stimulus: []string{"saya"}, Cycles: 2, Timestamp: clock.Now()}); err != nil {
		t.Fatalf("process: %v", err)
	}
	clock.Set(clock.Now().Add(time.Second))
	if _, _, err := r.ThinkAt(2, clock.Now()); err != nil {
		t.Fatalf("think: %v", err)
	}

	path := filepath.Join(t.TempDir(), "checkpoint.json")
	if err := r.Checkpoint(path); err != nil {
		t.Fatalf("checkpoint: %v", err)
	}

	resumed := NewBrainRuntime(nil)
	resumed.SetClock(clock)
	if err := resumed.RestoreCheckpoint(path); err != nil {
		t.Fatalf("restore: %v", err)
	}

	clock.Set(clock.Now().Add(time.Second))
	original, _, err := r.ThinkAt(2, clock.Now())
	if err != nil {
		t.Fatalf("original continuation: %v", err)
	}
	resumedResult, _, err := resumed.ThinkAt(2, clock.Now())
	if err != nil {
		t.Fatalf("restored continuation: %v", err)
	}
	if !reflect.DeepEqual(original.Activations, resumedResult.Activations) ||
		!reflect.DeepEqual(original.Confidence, resumedResult.Confidence) ||
		!reflect.DeepEqual(original.Prediction.State, resumedResult.Prediction.State) ||
		!reflect.DeepEqual(original.Prediction.Confidence, resumedResult.Prediction.Confidence) ||
		original.PredictionError != resumedResult.PredictionError {
		t.Fatalf("checkpoint did not preserve recurrent continuation")
	}
}

func TestEventReplayReproducesCheckpointedRuntimeState(t *testing.T) {
	clock := NewFixedClock(time.Date(2026, 9, 13, 13, 0, 0, 0, time.UTC))
	logPath := filepath.Join(t.TempDir(), "events.jsonl")
	log, err := OpenEventLog(logPath)
	if err != nil { t.Fatalf("open event log: %v", err) }
	defer log.Close()

	r := NewBrainRuntime(nil)
	r.SetClock(clock)
	r.SetEventLog(log)
	if _, err := r.LearnExperience(testExperience(), clock.Now()); err != nil { t.Fatalf("learn: %v", err) }
	if _, _, err := r.Process(Event{ID: "p1", Stimulus: []string{"saya"}, Cycles: 2, Timestamp: clock.Now()}); err != nil { t.Fatalf("process: %v", err) }
	clock.Set(clock.Now().Add(time.Second))
	if _, _, err := r.ThinkAt(2, clock.Now()); err != nil { t.Fatalf("think: %v", err) }

	events, err := ReadEventLog(logPath)
	if err != nil { t.Fatalf("read event log: %v", err) }
	if len(events) != 3 { t.Fatalf("event count: got %d want 3", len(events)) }

	replayClock := NewFixedClock(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC))
	replayed := NewBrainRuntime(nil)
	replayed.SetClock(replayClock)
	if err := ReplayEventLog(replayed, events); err != nil { t.Fatalf("replay: %v", err) }
	if replayed.LastSequence() != r.LastSequence() { t.Fatalf("sequence mismatch: got %d want %d", replayed.LastSequence(), r.LastSequence()) }

	path := filepath.Join(t.TempDir(), "original.json")
	if err := r.Checkpoint(path); err != nil { t.Fatalf("original checkpoint: %v", err) }
	replayedPath := filepath.Join(t.TempDir(), "replayed.json")
	if err := replayed.Checkpoint(replayedPath); err != nil { t.Fatalf("replayed checkpoint: %v", err) }

	originalData, err := readCheckpointForTest(path)
	if err != nil { t.Fatalf("read original checkpoint: %v", err) }
	replayedData, err := readCheckpointForTest(replayedPath)
	if err != nil { t.Fatalf("read replayed checkpoint: %v", err) }
	brainEqual := reflect.DeepEqual(originalData.Brain, replayedData.Brain)
	activationEqual := reflect.DeepEqual(originalData.Activation, replayedData.Activation)
	if !brainEqual || !activationEqual {
		originalBrainHash := sha256.Sum256(originalData.Brain)
		replayedBrainHash := sha256.Sum256(replayedData.Brain)
		t.Fatalf("event replay did not reproduce neural and recurrent runtime state: brain_equal=%t activation_equal=%t brain_len=%d/%d brain_sha=%s/%s brain_diff=%s activation=%+v/%+v",
			brainEqual, activationEqual, len(originalData.Brain), len(replayedData.Brain),
			hex.EncodeToString(originalBrainHash[:]), hex.EncodeToString(replayedBrainHash[:]),
			originalData.Activation, replayedData.Activation)
	}
}

type checkpointBrainForTest struct {
	Nodes []json.RawMessage `json:"nodes"`
	Patterns []json.RawMessage `json:"patterns"`
	ProjectionPopulations []json.RawMessage `json:"projection_populations"`
}

func diagnoseBrainDifference(t *testing.T, original, replayed json.RawMessage) string {
	t.Helper()
	var a, b checkpointBrainForTest
	if err := json.Unmarshal(original, &a); err != nil { return "original brain decode: " + err.Error() }
	if err := json.Unmarshal(replayed, &b); err != nil { return "replayed brain decode: " + err.Error() }
	if len(a.Nodes) != len(b.Nodes) { return fmt.Sprintf("nodes length %d/%d", len(a.Nodes), len(b.Nodes)) }
	for i := range a.Nodes {
		if !reflect.DeepEqual(a.Nodes[i], b.Nodes[i]) {
			return fmt.Sprintf("node index %d differs: original=%s replayed=%s", i, a.Nodes[i], b.Nodes[i])
		}
	}
	if len(a.Patterns) != len(b.Patterns) { return fmt.Sprintf("patterns length %d/%d", len(a.Patterns), len(b.Patterns)) }
	for i := range a.Patterns {
		if !reflect.DeepEqual(a.Patterns[i], b.Patterns[i]) {
			return fmt.Sprintf("pattern index %d differs: original=%s replayed=%s", i, a.Patterns[i], b.Patterns[i])
		}
	}
	if len(a.ProjectionPopulations) != len(b.ProjectionPopulations) { return fmt.Sprintf("projection populations length %d/%d", len(a.ProjectionPopulations), len(b.ProjectionPopulations)) }
	for i := range a.ProjectionPopulations {
		if !reflect.DeepEqual(a.ProjectionPopulations[i], b.ProjectionPopulations[i]) {
			return fmt.Sprintf("projection population index %d differs: original=%s replayed=%s", i, a.ProjectionPopulations[i], b.ProjectionPopulations[i])
		}
	}
	return "unknown brain byte difference"
}

func readCheckpointForTest(path string) (BrainSnapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil { return BrainSnapshot{}, err }
	var snapshot BrainSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil { return BrainSnapshot{}, err }
	return snapshot, nil
}
