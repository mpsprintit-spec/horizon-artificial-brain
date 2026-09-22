package activation

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestActivateWithComparesObservationAgainstPreviousPrediction(t *testing.T) {
	brain := knowledge.NewBrain()
	first := brain.Store("first")
	second := brain.Store("second")
	brain.Connect(first, second, 0.9, 0.9, false)

	engine := NewEngine(brain)
	t0 := time.Unix(1000, 0).UTC()

	firstResult := engine.ActivateWith(Request{
		StimulusTokens: []string{"first"},
		Cycles: 1,
		Now: t0,
	})
	if firstResult.PredictionError != 0 {
		t.Fatalf("first observation unexpectedly has prediction error: %v", firstResult.PredictionError)
	}

	secondResult := engine.ActivateWith(Request{
		StimulusTokens: []string{"second"},
		Cycles: 1,
		Now: t0.Add(time.Second),
	})
	if secondResult.PredictionError <= 0 {
		t.Fatalf("expected second observation to compare against prior prediction, got %v", secondResult.PredictionError)
	}
}
