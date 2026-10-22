package runtime

import (
	"bytes"
	"encoding/json"
	"errors"
	"math"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

func continuationBytes(t *testing.T, r *BrainRuntime) []byte {
	t.Helper()
	brain, err := r.brain.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal([]any{json.RawMessage(brain), r.seq, r.activation.SnapshotState(), r.lastCognitiveState, r.lastObservationPopulation, r.lastObservationAt, r.processedEvents, r.inquiryValence, r.actions, r.evidence.Snapshot(), r.brain.SurfaceAnnotationsSnapshot(), r.neuralWrittenGrounding})
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestAllRuntimeCommandsFailBeforeMutationOnUncertainAppend(t *testing.T) {
	type operation func(*BrainRuntime, knowledge.NodeID, time.Time) error
	cases := map[string]operation{
		"process": func(r *BrainRuntime, id knowledge.NodeID, at time.Time) error {
			_, _, err := r.Process(Event{ID: "fail", Timestamp: at, StimulusNodeIDs: []knowledge.NodeID{id}, Cycles: 1})
			return err
		},
		"think": func(r *BrainRuntime, _ knowledge.NodeID, at time.Time) error {
			_, _, err := r.ThinkAt(2, at)
			return err
		},
		"cognitive_process": func(r *BrainRuntime, id knowledge.NodeID, at time.Time) error {
			_, err := r.CognitiveProcess(Event{ID: "fail", Timestamp: at, StimulusNodeIDs: []knowledge.NodeID{id}, Cycles: 1})
			return err
		},
		"cognitive_think": func(r *BrainRuntime, _ knowledge.NodeID, _ time.Time) error {
			_, err := r.CognitiveThink(2)
			return err
		},
		"observation": func(r *BrainRuntime, _ knowledge.NodeID, at time.Time) error {
			_, err := r.ProcessObservationEnvelope(transactionObservation("fail", at, "unseen"))
			return err
		},
		"learn": func(r *BrainRuntime, _ knowledge.NodeID, at time.Time) error {
			_, err := r.LearnExperience(learning.Experience{Sequence: []string{"unseen", "learning"}}, at)
			return err
		},
		"action_binding": func(r *BrainRuntime, id knowledge.NodeID, _ time.Time) error {
			return r.RegisterActionBinding(ActionBinding{RequestID: "another-action", TargetNodeIDs: []knowledge.NodeID{id}})
		},
		"outcome": func(r *BrainRuntime, _ knowledge.NodeID, at time.Time) error {
			_, err := r.ObserveOutcome(OutcomeEvent{RequestID: "inquiry-12-focus", Observation: []string{"seen"}, Success: true, Reliability: .9, ObservedAt: at})
			return err
		},
		"consequence": func(r *BrainRuntime, id knowledge.NodeID, at time.Time) error {
			_, err := r.RecordInquiryConsequenceEvent(InquiryConsequenceEvent{Action: InquiryFocus, Valence: .8, InformationGain: .7, Reliability: .9, TargetNodeIDs: []knowledge.NodeID{id}, OutcomeNodeIDs: []knowledge.NodeID{id}, CausalLink: "fail"}, at)
			return err
		},
		"inquiry_outcome": func(r *BrainRuntime, _ knowledge.NodeID, at time.Time) error {
			_, _, _, err := NewCognitiveOrchestrator(r).ProcessInquiryOutcome(InquiryResult{Execution: transactionInquiry(t, r, at), Event: Event{ID: "fail", Timestamp: at, Cycles: 1}, Observation: ObservationInput{Source: "camera", Modality: "vision", Tokens: []string{"unseen"}}, Success: true, Reliability: .9, InformationGain: .6}, at)
			return err
		},
	}
	for name, operation := range cases {
		t.Run(name, func(t *testing.T) {
			at := time.Date(2026, 10, 19, 9, 0, 0, 0, time.UTC)
			journal := openKnowledgeJournal(t)
			r := NewBrainRuntime(knowledge.NewBrain())
			r.SetClock(NewFixedClock(at))
			r.SetEventLog(journal)
			grounded, err := r.GroundVectorAt(knowledge.NewNeuralVector([]float64{.2, .8}), at)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := r.CognitiveProcess(Event{ID: "prefix", Timestamp: at, StimulusNodeIDs: []knowledge.NodeID{grounded.NodeID}, Cycles: 1}); err != nil {
				t.Fatal(err)
			}
			if err := r.RegisterActionBinding(ActionBinding{RequestID: "inquiry-12-focus", TargetNodeIDs: []knowledge.NodeID{grounded.NodeID}}); err != nil {
				t.Fatal(err)
			}
			before := continuationBytes(t, r)
			subscriber := r.telemetry.Subscribe()
			journal.beforeAppend = func(LoggedEvent) error { return errors.New("uncertain write/fsync") }
			if err := operation(r, grounded.NodeID, at.Add(time.Second)); !errors.Is(err, ErrRuntimeRecoveryRequired) {
				t.Fatalf("uncertain append did not retire runtime: %v", err)
			}
			if !bytes.Equal(before, continuationBytes(t, r)) {
				t.Fatal("command changed state before WAL acknowledgment")
			}
			if _, open := <-subscriber.C(); open {
				t.Fatal("failed command published successful telemetry")
			}
			if _, err := r.CognitiveThink(1); !errors.Is(err, ErrRuntimeRecoveryRequired) {
				t.Fatal("poisoned runtime accepted another transition")
			}
			if err := r.LearnObservedTransition([]knowledge.NodeID{grounded.NodeID}, at); !errors.Is(err, ErrRuntimeRecoveryRequired) {
				t.Fatal("partial temporal helper hid runtime unavailability")
			}
			if _, err := r.PlanInquiry(.8, at); !errors.Is(err, ErrRuntimeRecoveryRequired) {
				t.Fatal("partial inquiry helper hid runtime unavailability")
			}
			journal.beforeAppend = nil
			rebuilt, err := RebuildRuntimeFromCommittedJournal(journal, knowledge.NewBrain)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(before, continuationBytes(t, rebuilt)) {
				t.Fatal("committed prefix no longer reconstructs after append failure")
			}
		})
	}
}

func TestDurableObservationWithLostAcknowledgmentReplaysExactlyOnce(t *testing.T) {
	at := time.Date(2026, 10, 19, 9, 0, 0, 0, time.UTC)
	journal := openKnowledgeJournal(t)
	live := NewBrainRuntime(knowledge.NewBrain())
	live.SetEventLog(journal)
	journal.beforeAppend = func(command LoggedEvent) error {
		data, err := json.Marshal(command)
		if err != nil {
			return err
		}
		if _, err := journal.file.Write(append(data, '\n')); err != nil {
			return err
		}
		if err := journal.file.Sync(); err != nil {
			return err
		}
		return errors.New("acknowledgment lost after durable write")
	}
	input := transactionObservation("lost-ack", at, "observed")
	if _, err := live.ProcessObservationEnvelope(input); !errors.Is(err, ErrRuntimeRecoveryRequired) {
		t.Fatal(err)
	}
	if live.LastSequence() != 0 {
		t.Fatal("live state applied unacknowledged command")
	}
	journal.beforeAppend = nil
	recovered, err := RebuildRuntimeFromCommittedJournal(journal, knowledge.NewBrain)
	if err != nil {
		t.Fatal(err)
	}
	reference := NewBrainRuntime(knowledge.NewBrain())
	if _, err := reference.ProcessObservationEnvelope(input); err != nil {
		t.Fatal(err)
	}
	assertCompleteContinuation(t, reference, recovered)
	before := continuationBytes(t, recovered)
	if _, err := recovered.ProcessObservationEnvelope(input); err == nil {
		t.Fatal("lost-ack retry applied twice")
	}
	if !bytes.Equal(before, continuationBytes(t, recovered)) {
		t.Fatal("retry changed recovered state")
	}
}

func TestObservationInvalidLateFieldAndConcurrentDuplicateLeaveNoPartialGrounding(t *testing.T) {
	at := time.Date(2026, 10, 19, 9, 0, 0, 0, time.UTC)
	for _, bad := range []ObservationInput{
		{Tokens: []string{"valid-prefix", " "}},
		{Tokens: []string{"valid-prefix"}, ContextTokens: []string{""}},
		{Tokens: []string{"valid-prefix"}, DataTokens: []string{""}},
	} {
		r := NewBrainRuntime(nil)
		before := continuationBytes(t, r)
		if _, _, err := NewCognitiveOrchestrator(r).ProcessObservation(Event{ID: "invalid", Timestamp: at}, bad, nil); err == nil {
			t.Fatal("invalid late surface accepted")
		}
		if !bytes.Equal(before, continuationBytes(t, r)) || r.RecoveryRequired() {
			t.Fatal("input validation mutated or retired runtime")
		}
	}
	r := NewBrainRuntime(nil)
	journal := openKnowledgeJournal(t)
	r.SetEventLog(journal)
	before := continuationBytes(t, r)
	if _, _, err := NewCognitiveOrchestrator(r).ProcessObservation(Event{ID: "nan", Timestamp: at}, ObservationInput{Tokens: []string{"valid-prefix"}}, &learning.Experience{Sequence: []string{"bad"}, Weight: math.NaN()}); err == nil {
		t.Fatal("NaN experience accepted")
	}
	if !bytes.Equal(before, continuationBytes(t, r)) {
		t.Fatal("invalid experience changed state")
	}
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := r.ProcessObservationEnvelope(transactionObservation("same", at, "seen"))
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	successes := 0
	for err := range errs {
		if err == nil {
			successes++
		}
	}
	if successes != 1 || r.LastSequence() != 1 {
		t.Fatal("concurrent duplicate projected more than once")
	}
	data, err := os.ReadFile(journal.file.Name())
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Count(data, []byte{'\n'}) != 1 {
		t.Fatal("invalid or duplicate command reached WAL")
	}
}
