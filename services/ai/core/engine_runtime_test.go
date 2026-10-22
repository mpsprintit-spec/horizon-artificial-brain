package core

import (
	"github.com/project-horizon/horizon-core/services/ai/runtime"
	"testing"
)

func TestHorizonEngineUsesSingleBrainRuntime(t *testing.T) {
	h := NewHorizonEngine()
	if h == nil || h.Supervisor() == nil {
		t.Fatal("expected supervised brain runtime")
	}
	owner := h.Supervisor()
	if owner != h.Supervisor() {
		t.Fatal("facade replaced its owner handle")
	}
	before := inspectForTest(t, h)
	output, err := h.ThinkActive(1)
	if err != nil {
		t.Fatal(err)
	}
	after := inspectForTest(t, h)
	if output.BrainIdentity != runtime.BrainIdentity || after.Revision != before.Revision+1 {
		t.Fatal("facade thought did not advance the inspected canonical runtime")
	}
}
