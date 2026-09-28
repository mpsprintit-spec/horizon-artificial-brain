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
	if got := len(brain.ProjectionPopulations); got != len(BootstrapNetworkDomains) {
		t.Fatalf("population count = %d, want %d", got, len(BootstrapNetworkDomains))
	}
	if got := len(brain.Patterns.All()); got != len(BootstrapNetworkDomains) {
		t.Fatalf("temporal pattern count = %d, want %d", got, len(BootstrapNetworkDomains))
	}
	if got := len(brain.BrainState.BootstrapExperiences); got != len(BootstrapNetworkDomains) {
		t.Fatalf("bootstrap experience count = %d, want %d", got, len(BootstrapNetworkDomains))
	}
	if got := len(brain.BrainState.Episodes); got != len(BootstrapNetworkDomains) {
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
