package learning

import (
	"reflect"
	"testing"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestEvidenceSnapshotOwnsCopiesAndRestoresDedup(t *testing.T) {
	ledger := NewEvidenceLedger()
	input := OutcomeEvidence{RequestID: "request-a", NodeID: 1, Source: "camera", Evidence: Evidence{Weight: .6, Confidence: .5, Reliability: .9, IndependentSources: 1}}
	if _, _, err := ledger.Record(input); err != nil {
		t.Fatal(err)
	}
	snapshot := ledger.Snapshot()
	restored, err := RestoreEvidenceLedger(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	snapshot.ByNode[1] = Evidence{}
	snapshot.Seen[1][0] = "changed"
	if !reflect.DeepEqual(ledger.Snapshot(), restored.Snapshot()) {
		t.Fatal("restore retained caller-owned snapshot maps/slices")
	}
	if _, added, err := restored.Record(input); err != nil || added {
		t.Fatal("restore forgot already applied evidence key")
	}
	input.RequestID = "request-b"
	if _, _, err := restored.Record(input); err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(ledger.Snapshot(), restored.Snapshot()) {
		t.Fatal("restored ledger shares mutable state with original")
	}
}

func TestEvidenceSnapshotRejectsMissingAndDuplicateKeys(t *testing.T) {
	value := Evidence{Weight: .6, Confidence: .5, Reliability: .9, IndependentSources: 1}
	for _, snapshot := range []EvidenceLedgerSnapshot{
		{ByNode: map[knowledge.NodeID]Evidence{1: value}},
		{ByNode: map[knowledge.NodeID]Evidence{1: value}, Seen: map[knowledge.NodeID][]string{2: {"key"}}},
		{ByNode: map[knowledge.NodeID]Evidence{1: value}, Seen: map[knowledge.NodeID][]string{1: {"key", "key"}}},
		{ByNode: map[knowledge.NodeID]Evidence{1: value}, Seen: map[knowledge.NodeID][]string{1: {""}}},
	} {
		if _, err := RestoreEvidenceLedger(snapshot); err == nil {
			t.Fatal("invalid evidence continuation accepted")
		}
	}
}
