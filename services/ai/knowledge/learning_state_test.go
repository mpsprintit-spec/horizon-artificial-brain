package knowledge_test

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestPredictionErrorUpdatesPersistentLearningState(t *testing.T) {
	brain := knowledge.NewBrain()
	now := time.Unix(123, 0).UTC()

	brain.RecordLearningSignal(0.75, now)

	if got := brain.BrainState.PredictionState["last_error"]; got != 0.75 {
		t.Fatalf("prediction error not persisted: %v", got)
	}
	if got := brain.BrainState.PlasticityState["error_drive"]; got != 0.75 {
		t.Fatalf("plasticity error drive not updated: %v", got)
	}
	if got := brain.BrainState.PlasticityState["current"]; got <= brain.BrainState.PlasticityState["baseline"] {
		t.Fatalf("plasticity did not increase after prediction error: %v", got)
	}
	if got := brain.BrainState.MemoryState["last_priority"]; got <= brain.BrainState.MemoryState["retention"] {
		t.Fatalf("memory priority did not increase after prediction error: %v", got)
	}
	if got := brain.BrainState.PredictionState["last_update_unix"]; got != float64(now.UnixNano()) {
		t.Fatalf("prediction timestamp not persisted: %v", got)
	}
}
