package activation

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestLearnFromExperienceStrengthensExistingStructure(t *testing.T) {
	memory := knowledge.NewKnowledgeBase()
	a := memory.Store("a")
	b := memory.Store("b")
	memory.Connect(a, b, 0.30, 0.50, false)

	engine := NewEngine(memory)
	before := a.FindDynamicSynapse(b.ID, false).Dynamic.Weight
	engine.LearnFromExperience(Experience{
		Activations: map[knowledge.NodeID]float64{a.ID: 0.9, b.ID: 0.9},
		Confidence:  map[knowledge.NodeID]float64{a.ID: 1, b.ID: 1},
		Now:         time.Unix(100, 0).UTC(),
	})
	after := a.FindDynamicSynapse(b.ID, false).Dynamic.Weight
	if after <= before {
		t.Fatalf("expected existing connection to strengthen: before=%v after=%v", before, after)
	}
}

func TestLearnFromExperienceCanGrowRepeatedStrongCoactivation(t *testing.T) {
	memory := knowledge.NewKnowledgeBase()
	a := memory.Store("a")
	b := memory.Store("b")
	engine := NewEngine(memory)

	engine.LearnFromExperience(Experience{
		Activations: map[knowledge.NodeID]float64{a.ID: 0.95, b.ID: 0.95},
		Now:         time.Unix(100, 0).UTC(),
	})
	if a.FindDynamicSynapse(b.ID, false) == nil {
		t.Fatal("expected strong coactivation to create a structural connection")
	}
}

func TestLearnFromExperienceDoesNotGrowWeakSingleObservation(t *testing.T) {
	memory := knowledge.NewKnowledgeBase()
	a := memory.Store("a")
	b := memory.Store("b")
	engine := NewEngine(memory)

	engine.LearnFromExperience(Experience{
		Activations: map[knowledge.NodeID]float64{a.ID: 0.45, b.ID: 0.45},
		Now:         time.Unix(100, 0).UTC(),
	})
	if a.FindDynamicSynapse(b.ID, false) != nil {
		t.Fatal("weak coactivation must not create a new permanent connection")
	}
}
