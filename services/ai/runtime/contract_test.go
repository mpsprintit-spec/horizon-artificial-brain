package runtime

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestCognitiveOutputExposesCognitiveState(t *testing.T) {
	nodeID := knowledge.NodeID(7)
	output := CognitiveOutput{
		BrainIdentity: BrainIdentity,
		Sequence: 3,
		Timestamp: time.Unix(10, 0).UTC(),
		RankedNodeIDs: []knowledge.NodeID{nodeID},
		Activations: map[knowledge.NodeID]float64{nodeID: 0.8},
		Confidence: map[knowledge.NodeID]float64{nodeID: 0.6},
		Resonance: 0.9,
		PredictionError: 0.1,
	}
	state := output.State()
	if state.BrainIdentity != BrainIdentity || state.Sequence != 3 {
		t.Fatalf("unexpected cognitive state identity/sequence: %+v", state)
	}
	if len(state.ActiveNodeIDs) != 1 || state.ActiveNodeIDs[0] != nodeID {
		t.Fatalf("active nodes not preserved: %+v", state.ActiveNodeIDs)
	}
	if state.Activations[nodeID] != 0.8 || state.Confidence[nodeID] != 0.6 {
		t.Fatalf("neural values not preserved: %+v / %+v", state.Activations, state.Confidence)
	}
}

func TestObserveOutcomeWritesEventAndLearnsSameBrain(t *testing.T) {
	brain := knowledge.NewBrain()
	rt := NewBrainRuntime(brain)
	path := filepath.Join(t.TempDir(), "events.jsonl")
	log, err := OpenEventLog(path)
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	rt.SetEventLog(log)

	now := time.Unix(100, 0).UTC()
	seq, err := rt.ObserveOutcome(OutcomeEvent{
		RequestID: "req-1",
		BrainIdentity: BrainIdentity,
		Success: true,
		Observation: []string{"motor", "completed"},
		Source: "test",
		Modality: "execution-outcome",
		ObservedAt: now,
		Reliability: 0.9,
	})
	if err != nil {
		t.Fatal(err)
	}
	if seq != 1 || rt.LastSequence() != 1 {
		t.Fatalf("unexpected runtime sequence: got %d", seq)
	}
	events, err := ReadEventLog(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Type != EventTypeOutcome || events[0].Outcome == nil {
		t.Fatalf("outcome was not persisted as a first-class event: %+v", events)
	}
	if events[0].Outcome.RequestID != "req-1" {
		t.Fatalf("wrong outcome request ID: %+v", events[0].Outcome)
	}
	if brain.Registry.Get("motor") == nil || brain.Registry.Get("completed") == nil {
		t.Fatal("outcome was not learned into the canonical Brain")
	}
}
