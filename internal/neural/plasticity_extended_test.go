package neural

import "testing"

func TestStructuralAdaptationCreatesConnectionFromPersistentCoactivity(t *testing.T) {
	cfg := DefaultConfig(); cfg.StructuralRate = 0.02
	n, err := New(cfg); if err != nil { t.Fatal(err) }
	a, _ := n.AddNeuron(); b, _ := n.AddNeuron()
	for i := 0; i < 20; i++ { _ = n.SetActivity(a, 1); _ = n.SetActivity(b, 1); n.Tick() }
	before := n.SynapseCount(); n.StructuralAdaptation(); if n.SynapseCount() <= before { t.Fatalf("expected topology growth") }
}

func TestNegativeRewardDepressesEligibleConnection(t *testing.T) {
	n, _ := New(DefaultConfig()); a, _ := n.AddNeuron(); b, _ := n.AddNeuron(); id, _ := n.Connect(a, b, 0.8)
	for i := 0; i < 5; i++ { _ = n.SetActivity(a, 1); _ = n.SetActivity(b, 1); n.Tick(); n.Learn(-1) }
	s, _ := n.Synapse(id); if s.Weight >= 0.8 { t.Fatalf("expected depression, got %v", s.Weight) }
}
