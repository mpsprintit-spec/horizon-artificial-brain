package core

import (
	"bytes"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/perception"
	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

func TestManagedPulseWALFailureDoesNotInjectPopulation(t *testing.T) {
	h := NewHorizonEngine()
	defer h.Supervisor.Close()
	journal, err := runtime.OpenEventLog(filepath.Join(t.TempDir(), "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	h.Runtime.SetEventLog(journal)
	if err := journal.Close(); err != nil {
		t.Fatal(err)
	}
	before, err := h.Knowledge.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	_, err = h.PulseSignals([]perception.PerceptionSignal{{Source: "local", Modality: "text", Tokens: []string{"uncommitted-pulse"}, ObservedAt: time.Date(2026, 10, 19, 9, 0, 0, 0, time.UTC)}})
	if !errors.Is(err, runtime.ErrRuntimeRecoveryRequired) {
		t.Fatalf("Pulse did not report uncertain write: %v", err)
	}
	after, err := h.Knowledge.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) || h.Runtime.LastSequence() != 0 {
		t.Fatal("Pulse grounded before WAL")
	}
}
