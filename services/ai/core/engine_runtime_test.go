package core

import "testing"

func TestHorizonEngineUsesSingleBrainRuntime(t *testing.T) {
	h := NewHorizonEngine()
	if h == nil || h.Runtime == nil {
		t.Fatal("expected brain runtime")
	}
	if h.Knowledge == nil {
		t.Fatal("expected engine brain reference")
	}
	if h.Learning == nil || h.Learning.Kb != h.Knowledge {
		t.Fatal("learning must operate on the same persistent brain")
	}
	if h.Thinking == nil || h.Thinking.Activation == nil || h.Thinking.Activation.Memory != h.Knowledge {
		t.Fatal("thinking activation must operate on the same persistent brain during migration")
	}
}
