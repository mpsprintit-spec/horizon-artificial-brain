package runtime

import (
	"errors"
	"math"
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
