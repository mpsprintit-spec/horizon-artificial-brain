package core

import (
	"bytes"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/perception"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

func TestManagedPulseWALFailureDoesNotInjectPopulation(t *testing.T) {
	// Retain a fixture-owned Brain only to measure state after the runtime
	// becomes unavailable. The production facade exports no such reference.
	brain := knowledge.NewBootstrapBrain()
	rt := runtime.NewBrainRuntime(brain)
	supervisor, err := runtime.NewRuntimeSupervisor(rt)
	if err != nil { t.Fatal(err) }
	h := &HorizonEngine{supervisor: supervisor, Perception: perception.UserInputPerception{}}
	defer h.Supervisor().Close()
	journal, err := runtime.OpenEventLog(filepath.Join(t.TempDir(), "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	attachJournalForTest(t,h,journal)
	if err := journal.Close(); err != nil {
		t.Fatal(err)
	}
	before, err := brain.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	_, err = h.PulseSignals([]perception.PerceptionSignal{{Source: "local", Modality: "text", Tokens: []string{"uncommitted-pulse"}, ObservedAt: time.Date(2026, 10, 19, 9, 0, 0, 0, time.UTC)}})
	if !errors.Is(err, runtime.ErrRuntimeRecoveryRequired) {
		t.Fatalf("Pulse did not report uncertain write: %v", err)
	}
	after, err := brain.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) || rt.LastSequence() != 0 {
		t.Fatal("Pulse grounded before WAL")
	}
}
