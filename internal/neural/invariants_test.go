package neural

import "testing"

func TestDuplicateConnectionsRejected(t *testing.T) {
	n, _ := New(DefaultConfig()); a, _ := n.AddNeuron(); b, _ := n.AddNeuron()
	if _, err := n.Connect(a, b, 0.1); err != nil { t.Fatal(err) }
	if _, err := n.Connect(a, b, 0.2); err != ErrDuplicateConnection { t.Fatalf("expected duplicate rejection, got %v", err) }
}

func TestInvalidConfigurationRejected(t *testing.T) {
	cfg := DefaultConfig(); cfg.TraceDecay = 2
	if _, err := New(cfg); err != ErrInvalidConfig { t.Fatalf("expected invalid config, got %v", err) }
}
