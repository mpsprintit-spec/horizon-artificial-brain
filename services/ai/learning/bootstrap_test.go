package learning

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestLearnExperienceUsesOntologyFreeSubstrate(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	unit := NewLearningUnit(kb)

	unit.LearnExperience(Experience{
		Sequence:   []string{"pengamatan", "keadaan", "perubahan"},
		Weight:     0.8,
		Confidence: 0.7,
	}, time.Now().UTC())

	observation := kb.Fetch("pengamatan")
	state := kb.Fetch("keadaan")
	change := kb.Fetch("perubahan")
	if observation == nil || state == nil || change == nil {
		t.Fatal("experience did not create neural units")
	}
	first := observation.FindDynamicSynapse(state.ID, false)
	second := state.FindDynamicSynapse(change.ID, false)
	if first == nil || second == nil {
		t.Fatal("experience did not form temporal substrate connections")
	}
	if first.Kind != "" || second.Kind != "" {
		t.Fatal("found semantic relation in ontology-free experience substrate")
	}
	patterns := kb.Patterns.MatchTrace([]knowledge.PatternStep{
		{NodeID: observation.ID, Position: 0, Activation: 1},
		{NodeID: state.ID, Position: 1, Activation: 1},
		{NodeID: change.ID, Position: 2, Activation: 1},
	}, nil)
	if len(patterns) == 0 {
		t.Fatal("learned experience was not represented as a temporal pattern")
	}
}

func TestBootstrapBasicExperiencesLoadsDataCorpus(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	unit := NewLearningUnit(kb)
	n, err := unit.BootstrapBasicExperiences(time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if n < 20 {
		t.Fatalf("expected foundational corpus, got %d experiences", n)
	}
	if len(kb.Registry.Nodes()) == 0 {
		t.Fatal("bootstrap produced no neural units")
	}
	if len(kb.Patterns.All()) == 0 {
		t.Fatal("bootstrap produced no learned patterns")
	}
}
