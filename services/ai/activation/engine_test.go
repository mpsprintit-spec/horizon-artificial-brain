package activation

import (
	"testing"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

func TestActivationEmergesFromLearnedGraph(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	learning.NewLearningUnit(kb).Assimilate("api panas", 0.9, 0.8)
	result := NewEngine(kb).Activate([]string{"api"}, 3)
	if !result.Converged {
		t.Fatal("expected activation to converge")
	}
	panas := kb.Fetch("panas")
	if result.Activations[panas.ID] <= 0 {
		t.Fatal("expected activation to spread to co-activated node")
	}
}
