package dnf

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestFabricUsesCanonicalBrainWithoutSecondaryStorage(t *testing.T) {
	brain := knowledge.NewBrain()
	fabric, err := NewFabric(brain)
	if err != nil {
		t.Fatal(err)
	}
	if fabric.Brain() != brain {
		t.Fatal("DNF fabric must reference the canonical brain directly")
	}

	a := brain.Store("a")
	b := brain.Store("b")
	brain.Connect(a, b, 0.4, 0.5, false)

	structure, err := fabric.Inspect(time.Unix(10, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	if structure.Nodes != 2 {
		t.Fatalf("nodes = %d, want 2", structure.Nodes)
	}
	if structure.DynamicSynapses != 1 {
		t.Fatalf("dynamic synapses = %d, want 1", structure.DynamicSynapses)
	}
	if structure.LegacySynapses != 0 {
		t.Fatalf("legacy synapses = %d, want 0", structure.LegacySynapses)
	}
}

func TestFabricRejectsNilBrain(t *testing.T) {
	if _, err := NewFabric(nil); err == nil {
		t.Fatal("expected nil brain error")
	}
}
