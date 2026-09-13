package runtime

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestCognitiveProcessUsesSingleBrainIdentityAndSubstrate(t *testing.T) {
	brain := knowledge.NewBrain()
	brain.Store("saya")
	brain.Store("ingin")
	brain.Store("belajar")
	rt := NewBrainRuntime(brain)

	out, err := rt.CognitiveProcess(Event{
		ID:        "test-1",
		Stimulus:  []string{"saya", "ingin", "belajar"},
		Cycles:    4,
		Timestamp: time.Unix(10, 0).UTC(),
	})
	if err != nil {
		t.Fatalf("CognitiveProcess failed: %v", err)
	}
	if out.BrainIdentity != "horizon-primary-brain" {
		t.Fatalf("unexpected brain identity: %q", out.BrainIdentity)
	}
	if out.Sequence != 1 {
		t.Fatalf("expected sequence 1, got %d", out.Sequence)
	}
	if len(out.Activations) == 0 || len(out.RankedNodeIDs) == 0 {
		t.Fatal("expected neural activation output")
	}
	if len(brain.Registry.Nodes()) != 3 {
		t.Fatalf("expected 3 reused substrate nodes, got %d", len(brain.Registry.Nodes()))
	}
}

func TestCognitiveThinkContinuesWithoutExternalStimulus(t *testing.T) {
	brain := knowledge.NewBrain()
	brain.Store("saya")
	brain.Store("ingin")
	brain.Store("belajar")
	rt := NewBrainRuntime(brain)

	if _, err := rt.CognitiveProcess(Event{Stimulus: []string{"saya", "ingin", "belajar"}, Cycles: 4}); err != nil {
		t.Fatalf("seed process failed: %v", err)
	}
	before := rt.LastSequence()
	out, err := rt.CognitiveThink(4)
	if err != nil {
		t.Fatalf("CognitiveThink failed: %v", err)
	}
	if out.Sequence != before+1 {
		t.Fatalf("expected internal transition sequence %d, got %d", before+1, out.Sequence)
	}
	if len(out.Activations) == 0 {
		t.Fatal("expected continuing internal activation")
	}
}
