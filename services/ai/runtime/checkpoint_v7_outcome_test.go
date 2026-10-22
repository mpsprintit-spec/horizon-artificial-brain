package runtime

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestCheckpointV7AuthenticatesBindingsAndEvidence(t *testing.T) {
	r := NewBrainRuntime(knowledge.NewBrain())
	at := time.Date(2026, 10, 19, 9, 0, 0, 0, time.UTC)
	grounded, err := r.GroundVectorAt(knowledge.NewNeuralVector([]float64{.2, .8}), at)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.RegisterActionBinding(ActionBinding{RequestID: "request", TargetNodeIDs: []knowledge.NodeID{grounded.NodeID}}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.ObserveOutcome(OutcomeEvent{RequestID: "request", Success: true, Observation: []string{"seen"}, Source: "camera", Reliability: .9, ObservedAt: at}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "checkpoint.json")
	if err := r.Checkpoint(path); err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"binding", "evidence", "dedup"} {
		t.Run(field, func(t *testing.T) {
			var snapshot BrainSnapshot
			if err := json.Unmarshal(original, &snapshot); err != nil {
				t.Fatal(err)
			}
			switch field {
			case "binding":
				binding := snapshot.ActionBindings["request"]
				binding.Intent = "tampered"
				snapshot.ActionBindings["request"] = binding
			case "evidence":
				value := snapshot.EvidenceLedger.ByNode[grounded.NodeID]
				value.Confidence = .99
				snapshot.EvidenceLedger.ByNode[grounded.NodeID] = value
			case "dedup":
				snapshot.EvidenceLedger.Seen[grounded.NodeID][0] = "another-request|camera"
			}
			data, err := json.Marshal(snapshot)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			if err := NewBrainRuntime(nil).RestoreCheckpoint(path); err == nil || !strings.Contains(err.Error(), "checksum") {
				t.Fatalf("tampered %s accepted: %v", field, err)
			}
		})
	}
	// Even a correctly checksummed file cannot refer to nonexistent neurons.
	var snapshot BrainSnapshot
	if err := json.Unmarshal(original, &snapshot); err != nil {
		t.Fatal(err)
	}
	binding := snapshot.ActionBindings["request"]
	binding.TargetNodeIDs[0] = knowledge.NodeID(999999)
	snapshot.ActionBindings["request"] = binding
	snapshot.Checksum, err = checkpointChecksumForSchema(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	restored := NewBrainRuntime(nil)
	if err := restored.RestoreCheckpoint(path); !errors.Is(err, ErrRuntimeRecoveryRequired) || !restored.RecoveryRequired() {
		t.Fatalf("invalid restored target stayed available: %v", err)
	}
}

func TestCheckpointV6RemainsReadableButCannotSupplyOutcomeOffsetState(t *testing.T) {
	journal := openKnowledgeJournal(t)
	r := NewBrainRuntime(nil)
	// Produce a genuine pre-ordered state before testing legacy import.
	r.dynamicsEncoding = knowledge.LegacyDynamics
	r.SetEventLog(journal)
	at := time.Date(2026, 10, 19, 9, 0, 0, 0, time.UTC)
	if _, err := r.ProcessObservationEnvelope(transactionObservation("prefix", at, "seen")); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "legacy-v6.json")
	if err := r.Checkpoint(path); err != nil {
		t.Fatal(err)
	}
	snapshot, err := readCheckpointForTest(path)
	if err != nil {
		t.Fatal(err)
	}
	snapshot.SchemaVersion, snapshot.ActionBindings, snapshot.EvidenceLedger = neuralBrainSnapshotSchemaVersion, nil, nil
	snapshot.Checksum, err = checkpointChecksumV6(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	legacy := NewBrainRuntime(nil)
	if err := legacy.RestoreCheckpoint(path); err != nil {
		t.Fatalf("v6 inspection/import compatibility: %v", err)
	}
	if _, err := RebuildRuntimeFromCheckpointAndJournal(journal, path); err == nil {
		t.Fatal("v6 claimed complete action/evidence continuation")
	}
	before := continuationBytes(t, r)
	if err := r.RestoreCheckpoint(path); err == nil {
		t.Fatal("attached journal accepted manual restore")
	}
	if string(before) != string(continuationBytes(t, r)) {
		t.Fatal("rejected manual restore changed active runtime")
	}
}
