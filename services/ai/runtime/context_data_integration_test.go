package runtime

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestCognitiveProcessContextAndDataInfluenceActivation(t *testing.T) {
	without := knowledge.NewBrain()
	without.Store("stimulus")
	without.Store("context")
	without.Store("data")

	with := knowledge.NewBrain()
	with.Store("stimulus")
	with.Store("context")
	with.Store("data")

	withoutRuntime := NewBrainRuntime(without)
	withRuntime := NewBrainRuntime(with)

	stamp := time.Unix(1000, 0).UTC()
	base, err := withoutRuntime.CognitiveProcess(Event{
		ID: "without-context-data",
		Stimulus: []string{"stimulus"},
		Cycles: 1,
		Timestamp: stamp,
	})
	if err != nil {
		t.Fatalf("baseline cognitive process failed: %v", err)
	}

	contextID := with.Registry.Get("context").ID
	dataID := with.Registry.Get("data").ID
	withResult, err := withRuntime.CognitiveProcess(Event{
		ID: "with-context-data",
		Stimulus: []string{"stimulus"},
		Context: map[knowledge.NodeID]float64{contextID: 0.35, dataID: 0.20},
		ContextTokens: []string{"context"},
		DataTokens: []string{"data"},
		Cycles: 1,
		Timestamp: stamp,
	})
	if err != nil {
		t.Fatalf("context/data cognitive process failed: %v", err)
	}

	baseContext := base.Activations[contextID]
	baseData := base.Activations[dataID]
	withContext := withResult.Activations[contextID]
	withData := withResult.Activations[dataID]

	if withContext <= baseContext {
		t.Fatalf("context did not influence activation: without=%v with=%v", baseContext, withContext)
	}
	if withData <= baseData {
		t.Fatalf("data did not influence activation: without=%v with=%v", baseData, withData)
	}
}
