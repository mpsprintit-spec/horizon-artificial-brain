package core

import (
	"testing"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

// Tests deliberately retain a runtime to exercise stale-reference rejection.
// Production integrations keep their work inside the supervisor lease.
func runtimeForTest(t *testing.T, h *HorizonEngine) *runtime.BrainRuntime {
	t.Helper()
	var result *runtime.BrainRuntime
	if err := h.Supervisor().With(func(r *runtime.BrainRuntime) error { result = r; return nil }); err != nil {
		t.Fatal(err)
	}
	return result
}
func attachJournalForTest(t *testing.T, h *HorizonEngine, log *runtime.EventLog) {
	t.Helper()
	if err := h.Supervisor().AttachEventLog(log); err != nil {
		t.Fatal(err)
	}
}
func inspectForTest(t *testing.T, h *HorizonEngine) runtime.BrainInspection {
	t.Helper()
	snapshot, err := h.InspectActiveBrain()
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
}
func unitForTest(t *testing.T, h *HorizonEngine, id knowledge.NodeID) knowledge.NeuralUnitRead {
	t.Helper()
	for _, unit := range inspectForTest(t, h).Graph.NeuralUnits() {
		if unit.ID == id {
			return unit
		}
	}
	t.Fatalf("unit %d absent from detached inspection", id)
	return knowledge.NeuralUnitRead{}
}
