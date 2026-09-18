package runtime

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestDNFPredictionErrorFeedsTemporalPlasticityLoop(t *testing.T) {
	brain := knowledge.NewBrain()
	source := brain.Store("source")
	target := brain.Store("target")
	brain.Connect(source, target, 0.20, 0.50, false)

	runtime := NewBrainRuntime(brain)
	if runtime == nil || runtime.DNF() == nil {
		t.Fatal("expected runtime and DNF fabric")
	}

	t0 := time.Unix(400, 0).UTC()
	first, err := runtime.CognitiveProcess(Event{
		ID:        "prediction-seed",
		Stimulus:  []string{"source"},
		Cycles:    1,
		Timestamp: t0,
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.Activations[source.ID] <= 0 {
		t.Fatalf("expected source activation: %v", first.Activations[source.ID])
	}

	// The first recurrent thought establishes a prediction trace in the same
	// canonical activation engine. No second memory or predictor is created.
	prediction, err := runtime.CognitiveThink(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(prediction.Prediction.State) == 0 {
		t.Fatal("expected recurrent prediction state")
	}

	before, err := runtime.DNF().SynapseState(source.ID, target.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if before.Eligibility <= 0 {
		t.Fatalf("expected an eligibility trace before prediction error: %v", before.Eligibility)
	}

	// A new observation makes the target unexpectedly active. ActivateWith
	// compares it with the previous recurrent prediction and routes the
	// mismatch into prediction-error plasticity.
	second, err := runtime.CognitiveProcess(Event{
		ID:        "prediction-surprise",
		Stimulus:  []string{"target"},
		Cycles:    1,
		Timestamp: t0.Add(500 * time.Millisecond),
	})
	if err != nil {
		t.Fatal(err)
	}
	if second.BrainIdentity != BrainIdentity {
		t.Fatalf("brain identity changed: got %q want %q", second.BrainIdentity, BrainIdentity)
	}

	after, err := runtime.DNF().SynapseState(source.ID, target.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if after.LastEligibilityUpdate.Before(t0) {
		t.Fatalf("prediction-error path did not advance eligibility timestamp: %v", after.LastEligibilityUpdate)
	}
	if after.Weight == before.Weight {
		t.Fatalf("prediction-error plasticity did not modify the existing pathway: before=%v after=%v", before.Weight, after.Weight)
	}
	if after.Weight > 0.95 {
		t.Fatalf("prediction-error plasticity exceeded weight bound: %v", after.Weight)
	}

	structure, err := runtime.DNF().Inspect(t0.Add(500 * time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}
	if structure.Nodes != 2 || structure.DynamicSynapses != 1 {
		t.Fatalf("prediction-error loop changed topology: %+v", structure)
	}
}
