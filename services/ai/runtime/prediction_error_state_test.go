package runtime

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestCognitiveProcessPropagatesPredictionError(t *testing.T) {
	now := time.Date(2026, 9, 22, 14, 0, 0, 0, time.UTC)
	brain := knowledge.NewBrain()
	first, _, err := brain.Registry.GetOrCreate("first")
	if err != nil { t.Fatal(err) }
	second, _, err := brain.Registry.GetOrCreate("second")
	if err != nil { t.Fatal(err) }
	brain.Connect(first, second, 0.9, 0.9, false)

	runtime := NewBrainRuntime(brain)
	firstOutput, err := runtime.CognitiveProcess(Event{Stimulus: []string{"first"}, Cycles: 1, Timestamp: now})
	if err != nil { t.Fatal(err) }
	if firstOutput.PredictionError != 0 {
		t.Fatalf("first cognitive observation unexpectedly has prediction error: %v", firstOutput.PredictionError)
	}

	secondOutput, err := runtime.CognitiveProcess(Event{Stimulus: []string{"second"}, Cycles: 1, Timestamp: now.Add(time.Second)})
	if err != nil { t.Fatal(err) }
	if secondOutput.PredictionError <= 0 {
		t.Fatalf("expected cognitive prediction error after prediction mismatch, got %v", secondOutput.PredictionError)
	}
}
