package knowledge

import (
	"testing"
	"time"
)

func TestGetOrCreateRepresentationAtPreservesEventTime(t *testing.T) {
	registry := NewTokenRegistry()
	at := time.Date(2026, 9, 19, 12, 34, 56, 0, time.UTC)

	node, created, _, err := registry.GetOrCreateRepresentationAt(
		NewNeuralVector([]float64{0.25, -0.5, 0.75}),
		0.9,
		at,
	)
	if err != nil {
		t.Fatalf("GetOrCreateRepresentationAt() error = %v", err)
	}
	if !created {
		t.Fatal("GetOrCreateRepresentationAt() created = false, want true")
	}
	if !node.LastActivation.Equal(at) {
		t.Fatalf("LastActivation = %s, want %s", node.LastActivation, at)
	}
	if len(node.UsageHistory) != 1 || !node.UsageHistory[0].Equal(at) {
		t.Fatalf("UsageHistory = %#v, want one event timestamp %s", node.UsageHistory, at)
	}
	if !node.LastActivation.Equal(at) {
		t.Fatalf("construction timestamp was not preserved: got %s, want %s", node.LastActivation, at)
	}
}
