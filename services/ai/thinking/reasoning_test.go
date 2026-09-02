package thinking

import (
	"testing"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

func TestThinkUsesContextToSelectRelevantConcept(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	learner := learning.NewLearningUnit(kb)
	learner.Assimilate("air minum segar", 0.9, 0.8)
	learner.Assimilate("air laut asin", 0.9, 0.8)

	thought, ok := NewThinkingEngine(kb).Think("air", []string{"laut"})
	if !ok {
		t.Fatal("expected thought")
	}
	if len(thought.Concepts) == 0 || thought.Concepts[0] != "laut" {
		t.Fatalf("expected context to emphasize laut, got %+v", thought)
	}
	if thought.Confidence <= 0 {
		t.Fatalf("expected confidence, got %f", thought.Confidence)
	}
}

func TestThinkFollowsMultiStepAssociations(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	learner := learning.NewLearningUnit(kb)
	learner.Assimilate("api menyebabkan panas", 0.9, 0.8)
	learner.Assimilate("panas menyebabkan asap", 0.9, 0.8)

	thought, ok := NewThinkingEngine(kb).Think("api", nil)
	if !ok {
		t.Fatal("expected thought")
	}
	found := false
	for _, concept := range thought.Concepts {
		if concept == "asap" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected multi-step inference to reach asap, got %+v", thought)
	}
}
