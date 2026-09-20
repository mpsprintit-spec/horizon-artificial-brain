package runtime

import (
	"testing"
	"time"
)

func TestProcessObservationExposesGroundingChangeAwareness(t *testing.T) {
	at := time.Date(2026, 9, 20, 4, 0, 0, 0, time.UTC)
	runtime := NewBrainRuntime(nil)
	orchestrator := NewCognitiveOrchestrator(runtime)

	interpretation, learned, err := orchestrator.ProcessObservation(
		Event{ID: "awareness-1", Timestamp: at, Cycles: 1},
		ObservationInput{
			Source: "test-sensor",
			Modality: "text",
			Tokens: []string{"cahaya"},
			DataTokens: []string{"cahaya"},
		},
		nil,
	)
	if err != nil {
		t.Fatalf("ProcessObservation() error = %v", err)
	}
	if learned {
		t.Fatal("ProcessObservation() learned = true, want false")
	}
	if interpretation.ChangeAwareness.BrainIdentity != BrainIdentity {
		t.Fatalf("BrainIdentity = %q, want %q", interpretation.ChangeAwareness.BrainIdentity, BrainIdentity)
	}
	if !interpretation.ChangeAwareness.Timestamp.Equal(at) {
		t.Fatalf("awareness timestamp = %s, want %s", interpretation.ChangeAwareness.Timestamp, at)
	}
	if len(interpretation.ChangeAwareness.KnowledgeChanges) != 1 {
		t.Fatalf("knowledge changes = %d, want 1", len(interpretation.ChangeAwareness.KnowledgeChanges))
	}
	change := interpretation.ChangeAwareness.KnowledgeChanges[0]
	if len(change.Claims) != 1 {
		t.Fatalf("claims = %d, want 1", len(change.Claims))
	}
	if change.Claims[0].Origin.Source != "test-sensor" {
		t.Fatalf("claim source = %q, want test-sensor", change.Claims[0].Origin.Source)
	}
	if !change.Claims[0].Origin.ObservedAt.Equal(at) {
		t.Fatalf("claim timestamp = %s, want %s", change.Claims[0].Origin.ObservedAt, at)
	}
	if change.Claims[0].Verified {
		t.Fatal("new grounding claim marked verified")
	}
	if len(change.ChangedNodeIDs) != 1 {
		t.Fatalf("changed node IDs = %d, want 1", len(change.ChangedNodeIDs))
	}
	if interpretation.ChangeAwareness.StateDelta.ToSequence != interpretation.State.Sequence {
		t.Fatalf("state delta ToSequence = %d, want %d", interpretation.ChangeAwareness.StateDelta.ToSequence, interpretation.State.Sequence)
	}
}
