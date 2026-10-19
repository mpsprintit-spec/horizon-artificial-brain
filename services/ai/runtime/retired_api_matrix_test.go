package runtime

import (
	"bytes"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

func TestRetiredRuntimeSupportedMutationAndInspectionMatrix(t *testing.T) {
	at := time.Unix(100, 0).UTC()
	r := NewBrainRuntime(nil)
	grounded, err := r.GroundVectorAt(knowledge.NewNeuralVector([]float64{.2, .8}), at)
	if err != nil {
		t.Fatal(err)
	}
	checkpoint := filepath.Join(t.TempDir(), "checkpoint.json")
	if err := r.Checkpoint(checkpoint); err != nil {
		t.Fatal(err)
	}
	owner, err := NewRuntimeSupervisor(r)
	if err != nil {
		t.Fatal(err)
	}
	if err := owner.Close(); err != nil {
		t.Fatal(err)
	}
	before := continuationBytes(t, r)
	event := Event{ID: "retired", Timestamp: at, Cycles: 1}
	cases := map[string]func() error{
		"process":           func() error { _, _, err := r.Process(event); return err },
		"cognitive-process": func() error { _, err := r.CognitiveProcess(event); return err },
		"think":             func() error { _, _, err := r.ThinkAt(1, at); return err },
		"cognitive-think":   func() error { _, err := r.CognitiveThink(1); return err },
		"experience": func() error {
			_, err := r.LearnExperience(learning.Experience{Sequence: []string{"retired"}}, at)
			return err
		},
		"observation": func() error {
			_, err := r.ProcessObservationEnvelope(transactionObservation("retired", at, "unseen"))
			return err
		},
		"sensor":           func() error { _, err := r.ProcessSensorObservation(testSensorObservation()); return err },
		"knowledge-import": func() error { _, err := r.ProcessKnowledgeDocument(journalKnowledgeDocument(), at); return err },
		"ground":           func() error { _, err := r.GroundObservation("unseen", "local", "text"); return err },
		"timed-ground":     func() error { _, err := r.GroundObservationAt("unseen", "local", "text", at); return err },
		"vector":           func() error { _, err := r.GroundVectorAt(knowledge.NewNeuralVector([]float64{.4, .6}), at); return err },
		"transition":       func() error { return r.LearnObservedTransition([]knowledge.NodeID{grounded.NodeID}, at) },
		"binding": func() error {
			return r.RegisterActionBinding(ActionBinding{RequestID: "retired", TargetNodeIDs: []knowledge.NodeID{grounded.NodeID}})
		},
		"outcome": func() error {
			_, err := r.ObserveOutcome(OutcomeEvent{RequestID: "retired", Observation: []string{"seen"}, ObservedAt: at})
			return err
		},
		"claim": func() error {
			_, _, err := r.VerifyKnowledgeClaim(KnowledgeClaim{NodeID: grounded.NodeID}, learning.Evidence{IndependentSources: 2}, at)
			return err
		},
		"consequence": func() error {
			_, err := r.RecordInquiryConsequenceEvent(InquiryConsequenceEvent{Action: InquiryFocus}, at)
			return err
		},
		"plan":               func() error { _, err := r.PlanInquiry(.8, at); return err },
		"complete-inquiry":   func() error { return r.CompleteInquiry(.3, .4, at) },
		"inquiry-prediction": func() error { _, _, err := r.PredictInquiryInformationValue(InquiryFocus, .8, at); return err },
		"capture-prediction": func() error { _, err := r.CaptureInquiryPrediction(InquiryExecution{}, at); return err },
		"bootstrap":          func() error { _, err := r.ActivateBootstrap(at); return err },
		"save":               func() error { return r.SaveBrain(filepath.Join(t.TempDir(), "brain.json")) },
		"load":               func() error { return r.LoadBrain(checkpoint) },
		"checkpoint":         func() error { return r.Checkpoint(filepath.Join(t.TempDir(), "new.json")) },
		"restore":            func() error { return r.RestoreCheckpoint(checkpoint) },
		"event-index":        func() error { return r.RestoreEventIndex(nil) },
		"mutable-view": func() error {
			return r.WithBrainView(func(*knowledge.Brain) error { t.Error("retired callback invoked"); return nil })
		},
		"inspection": func() error { _, err := r.InspectBrain(); return err },
		"labels":     func() error { _, err := r.ResolveNodeLabels([]knowledge.NodeID{grounded.NodeID}, "text"); return err },
		"monitor":    func() error { _, err := r.MonitorSnapshot(); return err },
		"handshake":  func() error { _, err := r.MonitorHandshake("test"); return err },
	}
	for name, operation := range cases {
		t.Run(name, func(t *testing.T) {
			if err := operation(); !errors.Is(err, ErrRuntimeRecoveryRequired) {
				t.Fatalf("retired API returned %v", err)
			}
			if !bytes.Equal(before, continuationBytes(t, r)) {
				t.Fatal("retired API changed state")
			}
		})
	}
	if r.DNF() != nil || r.Telemetry() != nil {
		t.Fatal("retired runtime exposed live engine/hub")
	}
	if _, err := NewRuntimeSupervisor(r); !errors.Is(err, ErrRuntimeRecoveryRequired) {
		t.Fatalf("retired runtime acquired a new owner: %v", err)
	}
}
