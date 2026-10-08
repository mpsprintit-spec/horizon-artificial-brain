package knowledge

import (
	"testing"
	"time"
)

func TestMaterializeBootstrapCreatesDistributedDevelopmentalNetwork(t *testing.T) {
	brain := NewBrain()
	at := time.Unix(1767225600, 0).UTC()
	if err := MaterializeBootstrap(brain, at); err != nil {
		t.Fatal(err)
	}

	if got := len(brain.Registry.Nodes()); got < 100 {
		t.Fatalf("neural unit count = %d, want at least 100", got)
	}
	if got := len(brain.ProjectionPopulations); got != len(BootstrapNetworkDomains)+1 {
		t.Fatalf("population count = %d, want %d", got, len(BootstrapNetworkDomains)+1)
	}
	if got := len(brain.Patterns.All()); got != len(BootstrapNetworkDomains)+4 {
		t.Fatalf("temporal pattern count = %d, want %d", got, len(BootstrapNetworkDomains)+4)
	}
	if got := len(brain.BrainState.BootstrapExperiences); got != len(BootstrapNetworkDomains)+4 {
		t.Fatalf("bootstrap experience count = %d, want %d", got, len(BootstrapNetworkDomains))
	}
	if got := len(brain.BrainState.Episodes); got != len(BootstrapNetworkDomains)+4 {
		t.Fatalf("episode count = %d, want %d", got, len(BootstrapNetworkDomains))
	}

	seen := map[NodeID]int{}
	for _, population := range brain.ProjectionPopulations {
		if len(population.Units) < 2 {
			t.Fatalf("population %q has only %d units", population.Domain, len(population.Units))
		}
		for _, unit := range population.Units {
			seen[unit.NodeID]++
		}
	}
	shared := 0
	for _, count := range seen {
		if count > 1 {
			shared++
		}
	}
	if shared == 0 {
		t.Fatal("bootstrap populations are not overlapping")
	}
	for _, population := range brain.ProjectionPopulations {
		if len(population.CounterEvidenceTargets) == 0 {
			t.Fatalf("population %q has no counterevidence path", population.Domain)
		}
	}

	excitatory, inhibitory := 0, 0
	for _, node := range brain.Registry.Nodes() {
		for _, synapses := range node.Synapses {
			for _, synapse := range synapses {
				if synapse.Inhibitory {
					inhibitory++
				} else {
					excitatory++
				}
			}
		}
	}
	if excitatory == 0 || inhibitory == 0 {
		t.Fatalf("connectivity polarity missing: excitatory=%d inhibitory=%d", excitatory, inhibitory)
	}
	if brain.Registry.GetByID(brain.ProjectionPopulations[9].Units[0].NodeID).FindDynamicSynapse(
		brain.ProjectionPopulations[10].Units[0].NodeID, false) == nil {
		t.Fatal("prediction -> error pathway is missing")
	}
	if brain.Registry.GetByID(brain.ProjectionPopulations[10].Units[0].NodeID).FindDynamicSynapse(
		brain.ProjectionPopulations[9].Units[0].NodeID, true) == nil {
		t.Fatal("error -> prediction counterevidence pathway is missing")
	}

	before := len(brain.Registry.Nodes())
	if err := MaterializeBootstrap(brain, at); err != nil {
		t.Fatal(err)
	}
	if len(brain.Registry.Nodes()) != before {
		t.Fatalf("idempotent materialization changed node count: %d -> %d", before, len(brain.Registry.Nodes()))
	}
}

func TestBootstrapBrainPersistsRequiredDevelopmentalState(t *testing.T) {
	brain := NewBootstrapBrain()
	path := t.TempDir() + "/brain_memory.json"
	if err := brain.Save(path); err != nil {
		t.Fatal(err)
	}

	loaded := NewBrain()
	if err := loaded.Load(path); err != nil {
		t.Fatal(err)
	}
	if len(loaded.Registry.Nodes()) < 100 {
		t.Fatalf("loaded neural units = %d, want at least 100", len(loaded.Registry.Nodes()))
	}
	if len(loaded.BrainState.SelfModelState) == 0 ||
		len(loaded.BrainState.SocialModelState) == 0 ||
		len(loaded.BrainState.ValueState) == 0 {
		t.Fatal("developmental state maps were not persisted")
	}
	if len(loaded.BrainState.Episodes) == 0 {
		t.Fatal("bootstrap episodes were not persisted")
	}
}

func TestBootstrapAblationRetainsAlternativeSubstrate(t *testing.T) {
	brain := NewBootstrapBrain()
	population := brain.ProjectionPopulations[5]
	if len(population.Units) < 4 {
		t.Fatal("agency population is too small for ablation")
	}
	remaining := 0
	for i, unit := range population.Units {
		node := brain.Registry.GetByID(unit.NodeID)
		if node == nil {
			continue
		}
		if i < 2 {
			node.Activation = 0
			continue
		}
		remaining++
		if len(node.Synapses) == 0 {
			t.Fatalf("remaining unit %d lost all connectivity", node.ID)
		}
	}
	if remaining < 2 {
		t.Fatal("ablation removed the entire developmental pathway")
	}
	if len(brain.Registry.Nodes()) < 100 {
		t.Fatal("partial ablation erased the canonical neural substrate")
	}
}
