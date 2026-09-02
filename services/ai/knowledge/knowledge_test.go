package knowledge

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTokenRegistryPreventsDuplicateNodes(t *testing.T) {
	kb := NewKnowledgeBase()
	first := kb.Store(" Api ")
	second := kb.Store("api")
	if first.ID != second.ID {
		t.Fatalf("expected one unique node for canonical token, got %d and %d", first.ID, second.ID)
	}
	if len(kb.Registry.Nodes()) != 1 {
		t.Fatalf("expected one node, got %d", len(kb.Registry.Nodes()))
	}
}

func TestConnectStrengthensExistingSynapse(t *testing.T) {
	kb := NewKnowledgeBase()
	api := kb.Store("api")
	panas := kb.Store("panas")
	kb.Connect(api, panas, 0.4, 0.7, false)
	kb.Connect(api, panas, 0.8, 0.9, false)
	s := api.FindSynapse(panas.ID, RelationAssociation, false)
	if s == nil || s.Frequency != 2 {
		t.Fatalf("expected reinforced synapse frequency 2, got %v", s)
	}
}

func TestMultipleRelationKindsSurvive(t *testing.T) {
	kb := NewKnowledgeBase()
	a := kb.Store("a")
	b := kb.Store("b")
	kb.ConnectKind(a, b, RelationHas, 0.9, 0.9, false)
	kb.ConnectKind(a, b, RelationCause, 0.8, 0.8, false)
	kb.ConnectKind(a, b, RelationHas, 0.5, 0.5, true) // inhibitory polarity independent
	if a.FindSynapse(b.ID, RelationHas, false) == nil {
		t.Fatal("HAS excitatory missing")
	}
	if a.FindSynapse(b.ID, RelationCause, false) == nil {
		t.Fatal("CAUSE missing — overwritten by HAS")
	}
	if a.FindSynapse(b.ID, RelationHas, true) == nil {
		t.Fatal("HAS inhibitory missing")
	}
	if len(a.SynapsesTo(b.ID)) != 3 {
		t.Fatalf("expected 3 independent channels, got %d", len(a.SynapsesTo(b.ID)))
	}
}

func TestSaveLoadPreservesMultiRelation(t *testing.T) {
	kb := NewKnowledgeBase()
	a := kb.Store("cat")
	b := kb.Store("leg")
	kb.ConnectKind(a, b, RelationHas, 0.9, 0.9, false)
	kb.ConnectKind(a, b, RelationCause, 0.7, 0.7, false)
	dir := t.TempDir()
	path := filepath.Join(dir, "mem.json")
	if err := kb.Save(path); err != nil {
		t.Fatal(err)
	}
	kb2 := NewKnowledgeBase()
	if err := kb2.Load(path); err != nil {
		t.Fatal(err)
	}
	cat := kb2.Fetch("cat")
	leg := kb2.Fetch("leg")
	if cat.FindSynapse(leg.ID, RelationHas, false) == nil || cat.FindSynapse(leg.ID, RelationCause, false) == nil {
		t.Fatal("multi-relation lost after save/load")
	}
	_ = os.Remove(path)
}

func TestLegacySingleSynapseJSONLoads(t *testing.T) {
	// legacy: synapses value is object not array
	raw := `{
	  "nodes": [{
	    "id": 1, "token": "x", "activation": 0, "resting_activation": 0.05, "threshold": 0.25,
	    "frequency": 1, "importance": 0.5, "plasticity": 0.3,
	    "synapses": {
	      "2": {"target_id": 2, "kind": "has", "weight": 0.9, "frequency": 1, "confidence": 0.9, "inhibitory": false}
	    }
	  }, {
	    "id": 2, "token": "y", "activation": 0, "resting_activation": 0.05, "threshold": 0.25,
	    "frequency": 1, "importance": 0.5, "plasticity": 0.3, "synapses": {}
	  }],
	  "patterns": []
	}`
	dir := t.TempDir()
	path := filepath.Join(dir, "legacy.json")
	if err := os.WriteFile(path, []byte(raw), 0644); err != nil {
		t.Fatal(err)
	}
	kb := NewKnowledgeBase()
	if err := kb.Load(path); err != nil {
		t.Fatal(err)
	}
	x := kb.Fetch("x")
	y := kb.Fetch("y")
	if x == nil || y == nil {
		t.Fatal("nodes missing")
	}
	if x.FindSynapse(y.ID, RelationHas, false) == nil {
		t.Fatal("legacy single synapse not migrated")
	}
}
