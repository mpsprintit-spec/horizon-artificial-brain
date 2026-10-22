package runtime

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

func assertCompleteContinuation(t *testing.T, live, restored *BrainRuntime) {
	t.Helper()
	assertRecoveredNeuralState(t, live, restored)
	checks := []struct {
		name      string
		want, got any
	}{
		{"activation", live.activation.SnapshotState(), restored.activation.SnapshotState()},
		{"inquiry preferences", live.inquiryValence, restored.inquiryValence},
		{"actions", live.actions, restored.actions},
		{"evidence and dedup keys", live.evidence.Snapshot(), restored.evidence.Snapshot()},
		{"adapter provenance", live.brain.SurfaceAnnotationsSnapshot(), restored.brain.SurfaceAnnotationsSnapshot()},
		{"written grounding contract", live.neuralWrittenGrounding, restored.neuralWrittenGrounding},
	}
	for _, check := range checks {
		if !reflect.DeepEqual(check.want, check.got) {
			t.Fatalf("recovery lost %s", check.name)
		}
	}
}

func transactionInquiry(t *testing.T, r *BrainRuntime, at time.Time) InquiryExecution {
	t.Helper()
	execution, err := r.CaptureInquiryPrediction(InquiryExecution{
		Proposal: InquiryProposal{BrainIdentity: BrainIdentity, Sequence: 12, CandidateID: "focus", Action: InquiryFocus, Reversibility: true},
		Request:  ExecutionRequest{RequestID: "inquiry-12-focus", BrainIdentity: BrainIdentity, Expiry: at.Add(time.Minute), authorized: true},
	}, at)
	if err != nil {
		t.Fatal(err)
	}
	return execution
}

func TestMixedCommandsRecoverCompleteStateFromGenesisAndCheckpoint(t *testing.T) {
	at := time.Date(2026, 10, 19, 9, 0, 0, 0, time.UTC)
	clock := NewFixedClock(at)
	journal := openKnowledgeJournal(t)
	live := NewBrainRuntime(knowledge.NewBootstrapBrain())
	live.SetClock(clock)
	live.SetEventLog(journal)
	orch := NewCognitiveOrchestrator(live)
	grounded, err := live.GroundVectorAt(knowledge.NewNeuralVector([]float64{.2, .8}), at)
	if err != nil {
		t.Fatal(err)
	}
	envelope := transactionObservation("text-prefix", at.Add(time.Second), "observed", "surface")
	envelope.Payload = json.RawMessage(`{ "capture": "<local>" }`)
	if _, err := live.ProcessObservationEnvelope(envelope); err != nil {
		t.Fatal(err)
	}
	experience := learning.Experience{Sequence: []string{"observed", "changes"}, Weight: .6, Confidence: .4, Reliability: .8}
	if _, learned, err := orch.ProcessObservation(Event{ID: "inline-learning", Timestamp: at.Add(2 * time.Second), Cycles: 1}, ObservationInput{Source: "local", Modality: "text", Tokens: []string{"changes"}, ContextTokens: []string{"observed"}, DataTokens: []string{"sample"}}, &experience); err != nil || !learned {
		t.Fatalf("inline learning: learned=%v err=%v", learned, err)
	}
	if experience.ExperienceID != "" || !experience.Timestamp.IsZero() {
		t.Fatal("observation changed caller-owned experience provenance")
	}
	if _, err := live.ProcessKnowledgeDocument(journalKnowledgeDocument(), at.Add(3*time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := live.CognitiveProcess(Event{ID: "direct-cognitive", Timestamp: at.Add(4 * time.Second), StimulusNodeIDs: []knowledge.NodeID{grounded.NodeID}, Cycles: 2}); err != nil {
		t.Fatal(err)
	}
	clock.Set(at.Add(5 * time.Second))
	if _, err := live.CognitiveThink(2); err != nil {
		t.Fatal(err)
	}
	if _, _, err := live.Process(Event{ID: "legacy-process", Stimulus: []string{"observed"}, Timestamp: at.Add(6 * time.Second), Cycles: 1}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := live.thinkAtWithContext(1, map[knowledge.NodeID]float64{grounded.NodeID: .35}, at.Add(7*time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := live.RegisterActionBinding(ActionBinding{RequestID: "inquiry-12-focus", Intent: "inquiry:focus", TargetNodeIDs: []knowledge.NodeID{grounded.NodeID}}); err != nil {
		t.Fatal(err)
	}
	if _, err := live.ObserveOutcome(OutcomeEvent{RequestID: "inquiry-12-focus", Success: true, Observation: []string{"observed"}, Source: "observer-a", Reliability: .9, ObservedAt: at.Add(8 * time.Second)}); err != nil {
		t.Fatal(err)
	}
	checkpoint := filepath.Join(t.TempDir(), "prefix-checkpoint.json")
	if err := live.Checkpoint(checkpoint); err != nil {
		t.Fatal(err)
	}
	request := InquiryRequest{BrainIdentity: BrainIdentity, RequestID: "inquiry-observation", Action: InquiryFocus, Uncertainty: .8}
	if _, _, err := orch.ProcessInquiryObservation(request, Event{ID: "inquiry-observation", Timestamp: at.Add(9 * time.Second), Cycles: 1}, ObservationInput{Source: "local", Modality: "vision", Tokens: []string{"focus-result"}}, nil); err != nil {
		t.Fatal(err)
	}
	execution := transactionInquiry(t, live, at.Add(10*time.Second))
	result := InquiryResult{Execution: execution, Event: Event{ID: "causal-outcome", Timestamp: at.Add(11 * time.Second), Cycles: 1}, Observation: ObservationInput{Source: "observer-b", Modality: "vision", Tokens: []string{"unexpected"}}, Success: true, Reliability: .9, InformationGain: .6, Cost: .1, Risk: .2}
	before := live.LastSequence()
	interpretation, _, sequence, err := orch.ProcessInquiryOutcome(result, at.Add(11*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if sequence != before+1 || interpretation.State.Sequence != sequence || interpretation.Consequence == nil {
		t.Fatal("inquiry outcome did not commit one complete revision")
	}
	beforeRetry := continuationBytes(t, live)
	retry := result
	retry.Observation.Source, retry.Observation.Tokens = "another-observer", []string{"must-not-ground"}
	if _, _, _, err := orch.ProcessInquiryOutcome(retry, at.Add(11*time.Second)); err == nil {
		t.Fatal("duplicate inquiry outcome accepted")
	}
	if string(beforeRetry) != string(continuationBytes(t, live)) {
		t.Fatal("duplicate inquiry outcome changed evidence before rejection")
	}
	if _, err := live.RecordInquiryConsequenceEvent(InquiryConsequenceEvent{Action: InquiryFocus, Valence: -.4, InformationGain: .3, Reliability: .7, TargetNodeIDs: []knowledge.NodeID{grounded.NodeID}, OutcomeNodeIDs: groundedPopulationNodeIDs(interpretation.GroundedRepresentations), CausalLink: "follow-up"}, at.Add(12*time.Second)); err != nil {
		t.Fatal(err)
	}
	sensor := factorizedSensor()
	sensor.SampleID, sensor.Timestamp = "mixed-sensor", at.Add(13*time.Second)
	if _, err := live.ProcessSensorObservation(sensor); err != nil {
		t.Fatal(err)
	}
	if _, err := live.LearnExperience(experience, at.Add(14*time.Second)); err != nil {
		t.Fatal(err)
	}

	genesis, err := RebuildRuntimeFromCommittedJournal(journal, knowledge.NewBootstrapBrain)
	if err != nil {
		t.Fatal(err)
	}
	suffix, err := RebuildRuntimeFromCheckpointAndJournal(journal, checkpoint)
	if err != nil {
		t.Fatal(err)
	}
	assertCompleteContinuation(t, live, genesis)
	assertCompleteContinuation(t, live, suffix)

	// The processed-ID prefix includes the compound inquiry command too.
	afterOutcome := filepath.Join(t.TempDir(), "after-outcome.json")
	if err := live.Checkpoint(afterOutcome); err != nil {
		t.Fatal(err)
	}
	anchored, err := RebuildRuntimeFromCheckpointAndJournal(journal, afterOutcome)
	if err != nil {
		t.Fatal(err)
	}
	assertCompleteContinuation(t, live, anchored)

	// All recovered runtimes must make the same next transition and reject
	// the same retry, including evidence dedup after checkpoint restoration.
	live.SetEventLog(nil)
	var want CognitiveInterpretation
	for i, candidate := range []*BrainRuntime{live, genesis, suffix, anchored} {
		out, err := candidate.ProcessObservationEnvelope(transactionObservation("continuation", at.Add(15*time.Second), "unexpected"))
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			want = out
		} else if !reflect.DeepEqual(want, out) {
			t.Fatal("recovery changed the next cognitive output")
		}
		if _, err := candidate.ProcessObservationEnvelope(transactionObservation("causal-outcome", at.Add(16*time.Second), "duplicate-new-surface")); err == nil {
			t.Fatal("recovered dedup index forgot compound inquiry outcome")
		}
	}
	assertCompleteContinuation(t, live, genesis)
	assertCompleteContinuation(t, live, suffix)
}
