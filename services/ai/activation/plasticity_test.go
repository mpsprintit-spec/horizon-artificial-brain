package activation

import (
	"sync"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/memory"
)

func TestPredictionErrorPlasticityAdaptsExistingConnection(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	a := kb.Store("a")
	b := kb.Store("b")
	kb.Connect(a, b, 0.40, 0.50, false)

	synapse := a.Synapses[b.ID][0]
	before := synapse.Dynamic.Weight

	e := NewEngine(kb)
	e.ApplyPredictionErrorPlasticity(
		map[knowledge.NodeID]float64{a.ID: 0.2, b.ID: 0.1},
		map[knowledge.NodeID]float64{a.ID: 0.2, b.ID: 0.9},
		0.8,
		time.Now().UTC(),
	)

	if synapse.Dynamic.Weight <= before {
		t.Fatalf("expected surprising activation to strengthen existing path: before=%v after=%v", before, synapse.Dynamic.Weight)
	}
	if synapse.Dynamic.Weight > 0.95 || synapse.Dynamic.Weight < 0.05 {
		t.Fatalf("plasticity escaped bounded range: %v", synapse.Dynamic.Weight)
	}
}

func TestPredictionErrorPlasticityDoesNotCreateNewConnection(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	a := kb.Store("a")
	b := kb.Store("b")

	e := NewEngine(kb)
	e.ApplyPredictionErrorPlasticity(
		map[knowledge.NodeID]float64{a.ID: 0.2, b.ID: 0.0},
		map[knowledge.NodeID]float64{a.ID: 0.2, b.ID: 0.9},
		1.0,
		time.Now().UTC(),
	)

	if len(a.Synapses[b.ID]) != 0 {
		t.Fatal("prediction error must not invent a connection without structural growth evidence")
	}
}

func TestConsequencePlasticityUsesInformationGainWithoutCreatingConnections(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	a := kb.Store("a")
	b := kb.Store("b")
	kb.Connect(a, b, 0.40, 0.50, false)
	synapse := a.Synapses[b.ID][0]
	synapse.Dynamic.Eligibility = 1.0
	synapse.Dynamic.LastEligibilityUpdate = time.Now().UTC()

	e := NewEngine(kb)
	before := synapse.Dynamic.Weight
	e.ApplyConsequencePlasticity(0.8, 1.0, time.Now().UTC())
	if synapse.Dynamic.Weight <= before {
		t.Fatalf("positive consequence should strengthen eligible connection: before=%v after=%v", before, synapse.Dynamic.Weight)
	}

	before = synapse.Dynamic.Weight
	e.ApplyConsequencePlasticity(-0.8, 1.0, time.Now().UTC())
	if synapse.Dynamic.Weight >= before {
		t.Fatalf("negative consequence should weaken eligible connection: before=%v after=%v", before, synapse.Dynamic.Weight)
	}
	if len(a.Synapses[b.ID]) != 1 {
		t.Fatal("consequence plasticity must not create a connection")
	}
}

// TestCanonicalSynapseMutationSerializesPlasticityAndMemoryOptimization
// exercises two mutation paths that previously touched the same canonical
// synapse state without a shared ownership boundary.
func TestCanonicalSynapseMutationSerializesPlasticityAndMemoryOptimization(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	a := kb.Store("a")
	b := kb.Store("b")
	kb.Connect(a, b, 0.40, 0.50, false)

	synapse := a.Synapses[b.ID][0]
	synapse.Dynamic.Eligibility = 1.0
	synapse.Dynamic.LastEligibilityUpdate = time.Unix(1, 0).UTC()
	synapse.Dynamic.LastActivation = time.Unix(1, 0).UTC()

	activationEngine := NewEngine(kb)
	memoryEngine := memory.NewEngine(kb)
	now := time.Unix(100, 0).UTC()

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			for j := 0; j < 25; j++ {
				activationEngine.ApplyPredictionErrorPlasticity(
					map[knowledge.NodeID]float64{a.ID: 0.2, b.ID: 0.1},
					map[knowledge.NodeID]float64{a.ID: 0.2, b.ID: 0.9},
					0.8,
					now,
				)
			}
		}()
		go func() {
			defer wg.Done()
			for j := 0; j < 25; j++ {
				memoryEngine.Optimize(now, time.Second, 0.01)
			}
		}()
	}
	wg.Wait()

	if synapse.Dynamic.Weight < 0.05 || synapse.Dynamic.Weight > 0.95 {
		t.Fatalf("canonical synapse escaped bounded range after concurrent mutation: %v", synapse.Dynamic.Weight)
	}
}
