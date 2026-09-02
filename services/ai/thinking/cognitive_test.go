package thinking

import (
	"testing"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestCognitivePipelineBuildsStateHypothesisAndTrace(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	a := kb.Store("horizon")
	b := kb.Store("memory")
	c := kb.Store("reasoning")
	kb.Connect(a, b, 0.9, 0.9, false)
	kb.Connect(b, c, 0.8, 0.8, false)
	engine := NewThinkingEngine(kb)
	thought, ok := engine.Think("horizon", nil)
	if !ok {
		t.Fatal("expected thought")
	}
	if len(thought.Hypotheses) == 0 {
		t.Fatal("expected hypotheses")
	}
	if len(engine.LastState.FocusedNodes) == 0 {
		t.Fatal("expected focused nodes")
	}
	if len(engine.LastTrace.Steps) == 0 {
		t.Fatal("expected path trace")
	}
}

func TestWorkingMemoryClear(t *testing.T) {
	wm := newWorkingMemory([]string{"ctx"})
	wm.Activation[1] = 1
	wm.Hypotheses = []Hypothesis{{Confidence: 1}}
	wm.Clear()
	if len(wm.Activation) != 0 || len(wm.Hypotheses) != 0 {
		t.Fatal("working memory was not cleared")
	}
}
