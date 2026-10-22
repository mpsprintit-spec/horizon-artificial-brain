package runtime

import (
	"bytes"
	"errors"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

type transactionClock struct{ at time.Time }

func (c transactionClock) Now() time.Time { return c.at }

func transactionObservation(id string, at time.Time, tokens ...string) ObservationEnvelope {
	return ObservationEnvelope{SchemaVersion: 1, BrainIdentity: BrainIdentity, EventID: id, Source: "transaction-test", Modality: "text", Timestamp: at, Tokens: tokens}
}

func TestObservationDuplicateRejectsBeforeGrounding(t *testing.T) {
	r := NewBrainRuntime(nil)
	at := time.Date(2026, 10, 19, 9, 0, 0, 0, time.UTC)
	if _, err := r.ProcessObservationEnvelope(transactionObservation("same-id", at, "known")); err != nil {
		t.Fatal(err)
	}
	before, _ := r.brain.CanonicalJSON()
	activationBefore := r.activation.SnapshotState()
	if _, err := r.ProcessObservationEnvelope(transactionObservation("same-id", at.Add(time.Second), "unseen-duplicate")); err == nil {
		t.Fatal("duplicate accepted")
	}
	after, _ := r.brain.CanonicalJSON()
	if !bytes.Equal(before, after) || !reflect.DeepEqual(activationBefore, r.activation.SnapshotState()) || r.LastSequence() != 1 {
		t.Fatal("duplicate observation mutated the Brain before rejection")
	}
}

func TestObservationWALFailurePrecedesGroundingAndRetiresAvailability(t *testing.T) {
	journal, err := OpenEventLog(filepath.Join(t.TempDir(), "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
	r := NewBrainRuntime(nil)
	r.SetEventLog(journal)
	journal.beforeAppend = func(LoggedEvent) error { return errors.New("injected uncertain append") }
	before, _ := r.brain.CanonicalJSON()
	_, err = r.ProcessObservationEnvelope(transactionObservation("wal-failure", time.Date(2026, 10, 19, 9, 0, 0, 0, time.UTC), "new-before-wal"))
	if !errors.Is(err, ErrRuntimeRecoveryRequired) {
		t.Errorf("uncertain observation did not require recovery: %v", err)
	}
	after, _ := r.brain.CanonicalJSON()
	if !bytes.Equal(before, after) || r.LastSequence() != 0 {
		t.Fatal("WAL failure changed the neural state")
	}
}

func TestCognitiveProcessAndThinkReplayTheirCompleteTransitions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	journal, err := OpenEventLog(path)
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
	brain := knowledge.NewBrain()
	r := NewBrainRuntime(brain)
	r.SetEventLog(journal)
	at := time.Date(2026, 10, 19, 9, 0, 0, 0, time.UTC)
	grounded, err := r.GroundVectorAt(knowledge.NewNeuralVector([]float64{.2, .8}), at)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.CognitiveProcess(Event{ID: "cognitive-process", Timestamp: at.Add(time.Second), StimulusNodeIDs: []knowledge.NodeID{grounded.NodeID}, Cycles: 1}); err != nil {
		t.Fatal(err)
	}
	r.SetClock(transactionClock{at.Add(2 * time.Second)})
	if _, err := r.CognitiveThink(2); err != nil {
		t.Fatal(err)
	}
	events, err := ReadEventLog(path)
	if err != nil {
		t.Fatal(err)
	}
	replayed := NewBrainRuntime(knowledge.NewBrain())
	if err := ReplayEventLog(replayed, events); err != nil {
		t.Fatal(err)
	}
	want, _ := brain.CanonicalJSON()
	got, _ := replayed.brain.CanonicalJSON()
	if !bytes.Equal(want, got) || !reflect.DeepEqual(r.lastCognitiveState, replayed.lastCognitiveState) || !reflect.DeepEqual(r.activation.SnapshotState(), replayed.activation.SnapshotState()) {
		t.Fatal("cognitive replay lost learning, memory, curiosity, or recurrent state")
	}
}

func TestCheckpointRestoresActionBindingAndOutcomeEvidenceContinuation(t *testing.T) {
	r := NewBrainRuntime(knowledge.NewBrain())
	at := time.Date(2026, 10, 19, 9, 0, 0, 0, time.UTC)
	grounded, err := r.GroundVectorAt(knowledge.NewNeuralVector([]float64{.2, .8}), at)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.RegisterActionBinding(ActionBinding{RequestID: "action-1", TargetNodeIDs: []knowledge.NodeID{grounded.NodeID}}); err != nil {
		t.Fatal(err)
	}
	for _, source := range []string{"sensor-a", "sensor-b"} {
		if _, err := r.ObserveOutcome(OutcomeEvent{RequestID: "action-1", Success: true, Observation: []string{"seen"}, Source: source, Reliability: .9, ObservedAt: at}); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(t.TempDir(), "checkpoint.json")
	if err := r.Checkpoint(path); err != nil {
		t.Fatal(err)
	}
	restored := NewBrainRuntime(knowledge.NewBrain())
	if err := restored.RestoreCheckpoint(path); err != nil {
		t.Fatal(err)
	}
	outcome := OutcomeEvent{RequestID: "action-1", Success: true, Observation: []string{"seen"}, Source: "sensor-c", Reliability: .9, ObservedAt: at.Add(time.Second)}
	if _, err := r.ObserveOutcome(outcome); err != nil {
		t.Fatal(err)
	}
	if _, err := restored.ObserveOutcome(outcome); err != nil {
		t.Fatal(err)
	}
	want, _ := r.brain.CanonicalJSON()
	got, _ := restored.brain.CanonicalJSON()
	if !bytes.Equal(want, got) {

		t.Fatal("checkpoint lost cumulative outcome evidence")
	}
}
