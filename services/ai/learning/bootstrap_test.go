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

func TestLearnExperienceReusesDistributedStructure(t *testing.T) {
	brain := knowledge.NewKnowledgeBase()
	unit := NewLearningUnit(brain)
	now := time.Unix(100, 0).UTC()
	experience := Experience{
		Sequence:   []string{"air", "gelas", "meja"},
		Weight:     0.8,
		Confidence: 0.9,
	}

	unit.LearnExperience(experience, now)
	unit.LearnExperience(experience, now.Add(time.Second))

	air := brain.Fetch("air")
	gelas := brain.Fetch("gelas")
	meja := brain.Fetch("meja")
	if air == nil || gelas == nil || meja == nil {
		t.Fatal("expected all experienced units to exist")
	}
	if got := len(brain.Registry.Nodes()); got != 3 {
		t.Fatalf("expected repeated experience to reuse 3 units, got %d", got)
	}
	if air.Frequency != 2 || gelas.Frequency != 2 || meja.Frequency != 2 {
		t.Fatalf("expected one occurrence per experience: air=%d gelas=%d meja=%d", air.Frequency, gelas.Frequency, meja.Frequency)
	}

	edge := air.FindDynamicSynapse(gelas.ID, false)
	if edge == nil {
		t.Fatal("expected air->gelas dynamic connection")
	}
	if edge.Frequency != 2 {
		t.Fatalf("expected repeated connection to be reinforced, got frequency %d", edge.Frequency)
	}

	patterns := brain.Patterns.All()
	if len(patterns) != 1 {
		t.Fatalf("expected one reused temporal pattern, got %d", len(patterns))
	}
	if patterns[0].Frequency != 2 {
		t.Fatalf("expected temporal pattern frequency 2, got %d", patterns[0].Frequency)
	}
}

func TestLearnExperiencePreservesTemporalDifference(t *testing.T) {
	brain := knowledge.NewKnowledgeBase()
	unit := NewLearningUnit(brain)
	now := time.Unix(200, 0).UTC()

	unit.LearnExperience(Experience{
		Sequence:   []string{"api", "panas", "naik"},
		Weight:     0.8,
		Confidence: 0.9,
	}, now)
	unit.LearnExperience(Experience{
		Sequence:   []string{"api", "naik", "panas"},
		Weight:     0.8,
		Confidence: 0.9,
	}, now.Add(time.Second))

	patterns := brain.Patterns.All()
	if len(patterns) != 2 {
		t.Fatalf("expected two distinct temporal patterns, got %d", len(patterns))
	}
	if patterns[0].Frequency != 1 || patterns[1].Frequency != 1 {
		t.Fatalf("expected each distinct temporal pattern to have frequency 1: %+v", patterns)
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


func TestBootstrapBasicExperiencesIncludesDistributedFoundationalVectors(t *testing.T) {
	brain := knowledge.NewKnowledgeBase()
	unit := NewLearningUnit(brain)
	count, err := unit.BootstrapBasicExperiences(time.Unix(300, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	if count < 30 {
		t.Fatalf("expected language and distributed foundational experiences, got %d", count)
	}
	if len(brain.ProjectionPopulations) == 0 {
		t.Fatal("expected foundational bootstrap to create distributed projection populations")
	}
	if len(brain.Patterns.All()) < 20 {
		t.Fatalf("expected foundational bootstrap to create many temporal patterns, got %d", len(brain.Patterns.All()))
	}
	var represented int
	for _, node := range brain.Registry.Nodes() {
		if len(node.Representation) > 0 {
			represented++
		}
	}
	if represented == 0 {
		t.Fatal("expected numeric foundational experiences to create represented neural units")
	}
}
