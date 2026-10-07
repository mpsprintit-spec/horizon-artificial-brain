package runtime

import (
	"strings"
	"testing"
)

func TestMonitorSnapshotUsesCanonicalRuntimeState(t *testing.T) {
	r := NewBrainRuntime(nil)
	output, err := r.CognitiveProcess(Event{ID: "monitor-test", Stimulus: []string{"monitor"}, Cycles: 1})
	if err != nil {
		t.Fatalf("cognitive process: %v", err)
	}
	snapshot, err := r.MonitorSnapshot()
	if err != nil {
		t.Fatalf("monitor snapshot: %v", err)
	}
	if snapshot.StateRevision != output.Sequence {
		t.Fatalf("snapshot revision=%d output sequence=%d", snapshot.StateRevision, output.Sequence)
	}
	if !strings.HasPrefix(snapshot.CanonicalStateHash, "sha256:") {
		t.Fatalf("invalid canonical state hash: %q", snapshot.CanonicalStateHash)
	}
	if snapshot.Counts.NeuralUnits == 0 || len(snapshot.NeuralUnits) != snapshot.Counts.NeuralUnits {
		t.Fatalf("invalid neural unit count: %+v", snapshot.Counts)
	}
}

func TestMonitorHandshakeUsesSnapshotIntegrity(t *testing.T) {
	r := NewBrainRuntime(nil)
	handshake, err := r.MonitorHandshake("test-commit")
	if err != nil {
		t.Fatalf("monitor handshake: %v", err)
	}
	if handshake.RuntimeCommit != "test-commit" || handshake.CanonicalStateHash == "" {
		t.Fatalf("invalid handshake: %+v", handshake)
	}
}
