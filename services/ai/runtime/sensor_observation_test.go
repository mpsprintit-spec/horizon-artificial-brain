package runtime

import (
	"bytes"
	"errors"
	"math"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func testSensorObservation() SensorObservation {
	return SensorObservation{
		Modality: "vision", Source: "camera-0",
		Timestamp: time.Date(2026, 10, 12, 8, 0, 0, 0, time.UTC),
		SampleID: "camera-0:1",
		ChannelStates: []float64{0.2, -0.4, 0.8},
		SpatialCoordinates: []float64{0.1, -0.2, 0.3},
		TemporalDelta: []float64{0.05, -0.1},
		Confidence: 0.9,
		Provenance: SensorProvenance{Adapter: "test-camera", CalibrationID: "cal-1"},
	}
}

func TestSensorObservationRequiresStructuredNumericState(t *testing.T) {
	sensor := testSensorObservation()
	sensor.ChannelStates = nil
	sensor.SpatialCoordinates = nil
	sensor.TemporalDelta = nil
	if err := sensor.Validate(); !errors.Is(err, ErrInvalidSensorObservation) {
		t.Fatalf("expected invalid sensor state, got %v", err)
	}
}

func TestSensorObservationRejectsNonFiniteAndOutOfRangeValues(t *testing.T) {
	for _, value := range []float64{math.NaN(), math.Inf(1), 1.01, -1.01} {
		sensor := testSensorObservation()
		sensor.ChannelStates[0] = value
		if err := sensor.Validate(); !errors.Is(err, ErrInvalidSensorObservation) {
			t.Fatalf("value %v: expected validation failure, got %v", value, err)
		}
	}
}

func TestSensorObservationVectorPreservesSpatialAndTemporalDifferences(t *testing.T) {
	base := testSensorObservation()
	baseVector, err := base.NeuralVector()
	if err != nil { t.Fatal(err) }

	moved := testSensorObservation()
	moved.SpatialCoordinates[0] = 0.7
	movedVector, err := moved.NeuralVector()
	if err != nil { t.Fatal(err) }
	if baseVector.Similarity(movedVector) >= 1 {
		t.Fatal("different spatial evidence was collapsed into the same numeric pattern")
	}

	motion := testSensorObservation()
	motion.TemporalDelta[0] = -0.8
	motionVector, err := motion.NeuralVector()
	if err != nil { t.Fatal(err) }
	if baseVector.Similarity(motionVector) >= 1 {
		t.Fatal("different temporal evidence was collapsed into the same numeric pattern")
	}
}

func TestProcessSensorObservationProjectsIntoCanonicalBrain(t *testing.T) {
	brain := knowledge.NewBootstrapBrain()
	runtime := NewBrainRuntime(brain)
	if runtime == nil { t.Fatal("expected canonical brain runtime") }
	orchestrator := NewCognitiveOrchestrator(runtime)

	before := len(brain.Registry.Nodes())
	_, err := orchestrator.ProcessSensorObservation(testSensorObservation())
	if err != nil { t.Fatalf("process sensor observation: %v", err) }
	if runtime.LastSequence() == 0 {
		t.Fatal("expected sensor input to advance canonical runtime state")
	}
	if len(brain.Registry.Nodes()) <= before {
		t.Fatal("expected sensor input to project numeric state into the canonical brain")
	}
}


func TestSensorObservationEventReplayReconstructsCanonicalBrain(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sensor-events.jsonl")
	log, err := OpenEventLog(path)
	if err != nil { t.Fatalf("open event log: %v", err) }
	defer log.Close()

	at := time.Date(2026, 10, 12, 8, 0, 0, 0, time.UTC)
	sensor := testSensorObservation()
	sensor.Timestamp = at
	sensor.Provenance.ParentSampleIDs = []string{"frame-previous"}
	originalBrain := knowledge.NewBootstrapBrain()
	original := NewBrainRuntime(originalBrain)
	original.SetClock(NewFixedClock(at))
	original.SetEventLog(log)
	if _, err := NewCognitiveOrchestrator(original).ProcessSensorObservation(sensor); err != nil {
		t.Fatalf("process sensor observation: %v", err)
	}

	events, err := ReadEventLog(path)
	if err != nil { t.Fatalf("read event log: %v", err) }
	var sensorEvent *LoggedEvent
	for i := range events {
		if events[i].Type == EventTypeProcess && events[i].Event != nil && events[i].Event.SensorObservation != nil {
			sensorEvent = &events[i]
			break
		}
	}
	if sensorEvent == nil {
		t.Fatalf("sensor observation was not persisted as a replayable process event: %+v", events)
	}
	persisted := sensorEvent.Event.SensorObservation
	if persisted.SampleID != sensor.SampleID || persisted.Provenance.CalibrationID != sensor.Provenance.CalibrationID ||
		len(persisted.Provenance.ParentSampleIDs) != 1 || persisted.Provenance.ParentSampleIDs[0] != "frame-previous" {
		t.Fatalf("sensor provenance did not survive event serialization: %+v", persisted)
	}

	replayedBrain := knowledge.NewBootstrapBrain()
	replayed := NewBrainRuntime(replayedBrain)
	replayed.SetClock(NewFixedClock(at))
	if err := ReplayEventLog(replayed, events); err != nil {
		t.Fatalf("replay sensor event: %v", err)
	}

	originalPath := filepath.Join(t.TempDir(), "original-checkpoint.json")
	replayedPath := filepath.Join(t.TempDir(), "replayed-checkpoint.json")
	if err := original.Checkpoint(originalPath); err != nil { t.Fatalf("checkpoint original: %v", err) }
	if err := replayed.Checkpoint(replayedPath); err != nil { t.Fatalf("checkpoint replayed: %v", err) }
	originalSnapshot, err := readCheckpointForTest(originalPath)
	if err != nil { t.Fatalf("read original checkpoint: %v", err) }
	replayedSnapshot, err := readCheckpointForTest(replayedPath)
	if err != nil { t.Fatalf("read replayed checkpoint: %v", err) }
	if !bytes.Equal(originalSnapshot.Brain, replayedSnapshot.Brain) {
		t.Fatalf("sensor replay did not reproduce canonical neural substrate: original_bytes=%d replayed_bytes=%d", len(originalSnapshot.Brain), len(replayedSnapshot.Brain))
	}
	if !reflect.DeepEqual(originalSnapshot.Activation, replayedSnapshot.Activation) {
		t.Fatal("sensor replay did not reproduce recurrent activation state")
	}
}


func TestDuplicateSensorSampleIsRejectedBeforeSecondProjection(t *testing.T) {
	brain := knowledge.NewBootstrapBrain()
	runtime := NewBrainRuntime(brain)
	orchestrator := NewCognitiveOrchestrator(runtime)
	sensor := testSensorObservation()
	if _, err := orchestrator.ProcessSensorObservation(sensor); err != nil {
		t.Fatalf("first sensor sample: %v", err)
	}
	nodesAfterFirst := len(brain.Registry.Nodes())
	sequenceAfterFirst := runtime.LastSequence()
	if _, err := orchestrator.ProcessSensorObservation(sensor); err == nil {
		t.Fatal("expected duplicate sensor sample ID to be rejected")
	}
	if got := len(brain.Registry.Nodes()); got != nodesAfterFirst {
		t.Fatalf("duplicate sample mutated neural registry: before=%d after=%d", nodesAfterFirst, got)
	}
	if got := runtime.LastSequence(); got != sequenceAfterFirst {
		t.Fatalf("duplicate sample advanced runtime sequence: before=%d after=%d", sequenceAfterFirst, got)
	}
}


func TestConcurrentDuplicateSensorSamplesOnlyProjectOnce(t *testing.T) {
	brain := knowledge.NewBootstrapBrain()
	runtime := NewBrainRuntime(brain)
	orchestrator := NewCognitiveOrchestrator(runtime)
	sensor := testSensorObservation()
	const workers = 8
	var wg sync.WaitGroup
	var mu sync.Mutex
	successes := 0
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := orchestrator.ProcessSensorObservation(sensor)
			mu.Lock()
			defer mu.Unlock()
			if err == nil {
				successes++
			}
		}()
	}
	wg.Wait()
	if successes != 1 {
		t.Fatalf("expected exactly one successful projection for duplicate concurrent samples, got %d", successes)
	}
	if got := runtime.LastSequence(); got == 0 {
		t.Fatal("expected the accepted sensor sample to advance runtime state")
	}
}
