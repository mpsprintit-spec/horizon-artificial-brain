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
	a := brain.Store("a")
	b := brain.Store("b")
	engine := NewEngine(brain)
	for i := 0; i < 5; i++ {
		engine.ApplyActivityPlasticity(
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
	a := brain.Store("a")
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

func TestDecayEligibilityUsesElapsedTime(t *testing.T) {
	base := time.Date(2026, 9, 13, 9, 0, 0, 0, time.UTC)
	oneHalfLife := decayEligibility(1, base, base.Add(activityPlasticityEligibilityHalfLife))
	twoHalfLives := decayEligibility(1, base, base.Add(2*activityPlasticityEligibilityHalfLife))

	if oneHalfLife < 0.49 || oneHalfLife > 0.51 {
		t.Fatalf("expected one half-life to retain about half the eligibility, got %v", oneHalfLife)
	}
	if twoHalfLives < 0.24 || twoHalfLives > 0.26 {
		t.Fatalf("expected two half-lives to retain about quarter eligibility, got %v", twoHalfLives)
	}
	if twoHalfLives >= oneHalfLife {
		t.Fatalf("expected longer elapsed time to produce stronger decay: one=%v two=%v", oneHalfLife, twoHalfLives)
	}
}

func TestActivityPlasticityRecentTraceRetainsMoreEligibilityThanLongIdleTrace(t *testing.T) {
	base := time.Date(2026, 9, 13, 9, 0, 0, 0, time.UTC)

	recent := knowledge.NewBrain()
	recentSynapse := dynamicSynapseForTest(t, recent, "recent-a", "recent-b")
	recentSynapse.Dynamic.Eligibility = 1
	recentSynapse.Dynamic.LastEligibilityUpdate = base

	idle := knowledge.NewBrain()
	idleSynapse := dynamicSynapseForTest(t, idle, "idle-a", "idle-b")
	idleSynapse.Dynamic.Eligibility = 1
	idleSynapse.Dynamic.LastEligibilityUpdate = base

	recentEngine := NewEngine(recent)
	idleEngine := NewEngine(idle)
	recentA := recent.Store("recent-a")
	recentB := recent.Store("recent-b")
	idleA := idle.Store("idle-a")
	idleB := idle.Store("idle-b")

	recentEngine.ApplyActivityPlasticity(
		map[knowledge.NodeID]float64{recentA.ID: 1},
		map[knowledge.NodeID]float64{recentB.ID: 1},
		base.Add(100*time.Millisecond),
	)
	idleEngine.ApplyActivityPlasticity(
		map[knowledge.NodeID]float64{idleA.ID: 1},
		map[knowledge.NodeID]float64{idleB.ID: 1},
		base.Add(5*activityPlasticityEligibilityHalfLife),
	)

	if recentSynapse.Dynamic.Eligibility <= idleSynapse.Dynamic.Eligibility {
		t.Fatalf("expected recent activity to retain a stronger eligibility trace: recent=%v idle=%v", recentSynapse.Dynamic.Eligibility, idleSynapse.Dynamic.Eligibility)
	}
}

func TestEligibilityDecayNeverDeletesSynapse(t *testing.T) {
	brain := knowledge.NewBrain()
	synapse := dynamicSynapseForTest(t, brain, "a", "b")
	source := brain.Store("a")
	target := brain.Store("b")
	base := time.Date(2026, 9, 13, 9, 0, 0, 0, time.UTC)
	synapse.Dynamic.Eligibility = 1
	synapse.Dynamic.LastEligibilityUpdate = base

	NewEngine(brain).ApplyActivityPlasticity(
		map[knowledge.NodeID]float64{source.ID: 0},
		map[knowledge.NodeID]float64{target.ID: 0},
		base.Add(30*activityPlasticityEligibilityHalfLife),
	)

	found := false
	for _, candidate := range source.OutboundAll() {
		if candidate != nil && candidate.TargetID == target.ID && candidate.IsDynamic() {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected time decay to preserve the dynamic synapse")
	}
}
