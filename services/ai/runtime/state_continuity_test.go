package runtime

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestCognitiveStateContinuityTracksActualTransitions(t *testing.T) {
	brain := knowledge.NewBrain()
	runtime := NewBrainRuntime(brain)
	orchestrator := NewCognitiveOrchestrator(runtime)
	at := time.Date(2026, 9, 20, 5, 0, 0, 0, time.UTC)

	observation := ObservationInput{Source: "sensor", Modality: "text", Tokens: []string{"gelas", "air"}}
	first, _, err := orchestrator.ProcessObservation(Event{
		ID: "state-transition-1", Stimulus: []string{"gelas"}, Cycles: 1, Timestamp: at,
	}, observation, nil)
	if err != nil {
		t.Fatalf("first ProcessObservation: %v", err)
	}
	if first.ChangeAwareness.StateDelta.FromSequence != 0 || first.ChangeAwareness.StateDelta.ToSequence != first.State.Sequence {
		t.Fatalf("first state delta sequence = %d -> %d, want 0 -> %d",
			first.ChangeAwareness.StateDelta.FromSequence,
			first.ChangeAwareness.StateDelta.ToSequence,
			first.State.Sequence,
		)
	}
	if len(first.ChangeAwareness.StateDelta.AddedNodeIDs) == 0 {
		t.Fatal("first cognitive transition should report newly active neural nodes")
	}

	second, _, err := orchestrator.ProcessObservation(Event{
		ID: "state-transition-2", Stimulus: []string{"gelas", "air"}, Context: map[knowledge.NodeID]float64{first.State.ActiveNodeIDs[0]: 0.15}, Cycles: 1, Timestamp: at.Add(time.Second),
	}, observation, nil)
	if err != nil {
		t.Fatalf("second ProcessObservation: %v", err)
	}
	if second.ChangeAwareness.StateDelta.FromSequence != first.State.Sequence {
		t.Fatalf("second state delta starts at %d, want previous sequence %d",
			second.ChangeAwareness.StateDelta.FromSequence, first.State.Sequence)
	}
	if second.ChangeAwareness.StateDelta.ToSequence != second.State.Sequence {
		t.Fatalf("second state delta ends at %d, want current sequence %d",
			second.ChangeAwareness.StateDelta.ToSequence, second.State.Sequence)
	}
	if len(second.ChangeAwareness.StateDelta.AddedNodeIDs) != 0 {
		t.Fatalf("second repeated observation incorrectly reports added nodes: %v", second.ChangeAwareness.StateDelta.AddedNodeIDs)
	}
	if len(second.ChangeAwareness.StateDelta.ActivationDelta) == 0 {
		t.Fatal("second cognitive transition should report activation continuity/change")
	}
}
