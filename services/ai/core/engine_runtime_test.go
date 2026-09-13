package core

import "testing"

func TestHorizonEngineUsesSingleBrainRuntime(t *testing.T) {
	h := NewHorizonEngine()
	if h == nil || h.Runtime == nil {
		t.Fatal("expected brain runtime")
	}
	if h.Runtime.Brain != h.Knowledge {
		t.Fatal("runtime and engine must reference the same persistent brain")
	}
	if h.Learning == nil || h.Learning.Kb != h.Runtime.Brain {
		t.Fatal("learning must operate on the runtime brain")
	}
	if h.Thinking == nil || h.Thinking.Activation == nil || h.Thinking.Activation.Memory != h.Runtime.Brain {
		t.Fatal("thinking activation must operate on the runtime brain")
	}
}
