package activation

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func dynamicSynapseForTest(t *testing.T, brain *knowledge.Brain, sourceToken, targetToken string) *knowledge.Synapse {
	t.Helper()
	source := brain.Store(sourceToken)
	target := brain.Store(targetToken)
	brain.Connect(source, target, 0.4, 0.5, false)
	for _, synapse := range source.OutboundAll() {
		if synapse != nil && synapse.TargetID == target.ID && synapse.IsDynamic() {
			return synapse
		}
	}
	t.Fatalf("dynamic synapse %s -> %s not found", sourceToken, targetToken)
	return nil
}

func TestActivityPlasticityStrengthensCoActivePath(t *testing.T) {
	brain := knowledge.NewBrain()
	synapse := dynamicSynapseForTest(t, brain, "a", "b")
	initial := synapse.Dynamic.Weight
	now := time.Date(2026, 9, 13, 9, 0, 0, 0, time.UTC)

	for i := 0; i < 5; i++ {
		Apply := map[knowledge.NodeID]float64{}
		Apply[brain.Registry.GetOrCreate("a").ID] = 0.9
		post := map[knowledge.NodeID]float64{}
		post[brain.Registry.GetOrCreate("b").ID] = 0.9
		// Use a fresh timestamp so the transition is a genuine repeated event.
		_ = i
		_ = Apply
		_ = post
	}

	a := brain.Registry.GetOrCreate("a")
	b := brain.Registry.GetOrCreate("b")
	for i := 0; i < 5; i++ {
		e := NewEngine(brain)
		e.ApplyActivityPlasticity(
			map[knowledge.NodeID]float64{a.ID: 0.9},
			map[knowledge.NodeID]float64{b.ID: 0.9},
			now.Add(time.Duration(i)*time.Millisecond),
		)
	}

	if synapse.Dynamic.Weight <= initial {
		t.Fatalf("expected co-active path to strengthen: initial=%v final=%v", initial, synapse.Dynamic.Weight)
	}
	if synapse.Dynamic.Frequency <= 1 {
		t.Fatalf("expected activity to increase synaptic frequency, got %d", synapse.Dynamic.Frequency)
	}
}

func TestActivityPlasticityDecaysUnusedPathWithoutDeletingIt(t *testing.T) {
	brain := knowledge.NewBrain()
	synapse := dynamicSynapseForTest(t, brain, "a", "b")
	initial := synapse.Dynamic.Weight
	a := brain.Registry.GetOrCreate("a")
	now := time.Date(2026, 9, 13, 9, 0, 0, 0, time.UTC)

	engine := NewEngine(brain)
	for i := 0; i < 5; i++ {
		engine.ApplyActivityPlasticity(
			map[knowledge.NodeID]float64{a.ID: 0},
			map[knowledge.NodeID]float64{},
			now.Add(time.Duration(i)*time.Millisecond),
		)
	}

	if synapse.Dynamic.Weight >= initial {
		t.Fatalf("expected unused path to decay: initial=%v final=%v", initial, synapse.Dynamic.Weight)
	}
	if synapse.Dynamic.Weight <= activityPlasticityMinWeight {
		t.Fatalf("expected unused path to remain structurally present, got weight=%v", synapse.Dynamic.Weight)
	}
}

func TestActivityPlasticityDoesNotDependOnSemanticRelationKind(t *testing.T) {
	brain := knowledge.NewBrain()
	source := brain.Store("source")
	target := brain.Store("target")
	brain.Connect(source, target, 0.4, 0.5, false)
	synapse := source.OutboundAll()[0]
	if synapse == nil {
		t.Fatal("expected synapse")
	}

	before := synapse.Dynamic.Weight
	engine := NewEngine(brain)
	engine.ApplyActivityPlasticity(
		map[knowledge.NodeID]float64{source.ID: 1},
		map[knowledge.NodeID]float64{target.ID: 1},
		time.Date(2026, 9, 13, 9, 0, 0, 0, time.UTC),
	)

	if synapse.Dynamic.Weight <= before {
		t.Fatalf("expected activity-driven update without semantic relation: before=%v after=%v", before, synapse.Dynamic.Weight)
	}
}
