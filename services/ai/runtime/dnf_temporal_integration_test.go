package runtime

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestDNFTemporalEligibilityDecaysAcrossIdleCognitiveCycles(t *testing.T) {
	brain := knowledge.NewBrain()
	source := brain.Store("source")
	target := brain.Store("target")
	brain.Connect(source, target, 0.20, 0.50, false)

	runtime := NewBrainRuntime(brain)
	if runtime == nil || runtime.DNF() == nil {
		t.Fatal("expected runtime and DNF fabric")
	}

	t0 := time.Unix(200, 0).UTC()
	if _, err := runtime.CognitiveProcess(Event{
		ID:        "temporal-seed",
		Stimulus:  []string{"source"},
		Cycles:    1,
		Timestamp: t0,
	}); err != nil {
		t.Fatal(err)
	}

	seed, err := runtime.DNF().SynapseState(source.ID, target.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if seed.Eligibility <= 0 {
		t.Fatalf("expected recent co-activity to create eligibility trace: %v", seed.Eligibility)
	}
	if !seed.LastEligibilityUpdate.Equal(t0) {
		t.Fatalf("unexpected eligibility timestamp: got %v want %v", seed.LastEligibilityUpdate, t0)
	}

	if _, err := runtime.CognitiveProcess(Event{
		ID:        "temporal-idle-1",
		Stimulus:  []string{"idle"},
		Cycles:    1,
		Timestamp: t0.Add(time.Second),
	}); err != nil {
		t.Fatal(err)
	}

	oneHalfLife, err := runtime.DNF().SynapseState(source.ID, target.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if oneHalfLife.Eligibility >= seed.Eligibility {
		t.Fatalf("eligibility did not decay during idle interval: seed=%v after=%v", seed.Eligibility, oneHalfLife.Eligibility)
	}
	if oneHalfLife.Eligibility <= 0 {
		t.Fatalf("eligibility trace vanished after one idle interval: %v", oneHalfLife.Eligibility)
	}
	if !oneHalfLife.LastEligibilityUpdate.Equal(t0.Add(time.Second)) {
		t.Fatalf("eligibility update timestamp did not advance: got %v", oneHalfLife.LastEligibilityUpdate)
	}

	if _, err := runtime.CognitiveProcess(Event{
		ID:        "temporal-idle-2",
		Stimulus:  []string{"idle"},
		Cycles:    1,
		Timestamp: t0.Add(2 * time.Second),
	}); err != nil {
		t.Fatal(err)
	}

	twoHalfLives, err := runtime.DNF().SynapseState(source.ID, target.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if twoHalfLives.Eligibility >= oneHalfLife.Eligibility {
		t.Fatalf("eligibility did not continue decaying: after1=%v after2=%v", oneHalfLife.Eligibility, twoHalfLives.Eligibility)
	}
	if twoHalfLives.Eligibility <= 0 {
		t.Fatalf("eligibility trace collapsed to zero too early: %v", twoHalfLives.Eligibility)
	}

	structure, err := runtime.DNF().Inspect(t0.Add(2 * time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if structure.Nodes != 3 || structure.DynamicSynapses != 1 {
		t.Fatalf("temporal decay unexpectedly changed DNF topology: %+v", structure)
	}
}
