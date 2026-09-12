package knowledge

import (
	"testing"
	"time"
)

func TestConnectCreatesOntologyFreeConnection(t *testing.T) {
	kb := NewKnowledgeBase()
	a := kb.Store("a")
	b := kb.Store("b")

	kb.Connect(a, b, 0.7, 0.8, false)

	connections := a.SynapsesTo(b.ID)
	if len(connections) != 1 {
		t.Fatalf("expected one connection, got %d", len(connections))
	}
	c := connections[0]
	if c.Kind != "" {
		t.Fatalf("dynamic connection must not carry semantic Kind: %q", c.Kind)
	}
	if !c.IsDynamic() {
		t.Fatal("connection should be identified as dynamic")
	}
	if c.Dynamic.Frequency != 1 || c.Frequency != 1 {
		t.Fatalf("expected frequency 1, got dynamic=%d legacy=%d", c.Dynamic.Frequency, c.Frequency)
	}
	if c.Dynamic.Weight != 0.7 || c.Dynamic.Confidence != 0.8 {
		t.Fatalf("unexpected dynamic state: %+v", c.Dynamic)
	}
}

func TestConnectReinforcesExistingDynamicConnection(t *testing.T) {
	kb := NewKnowledgeBase()
	a := kb.Store("a")
	b := kb.Store("b")

	kb.Connect(a, b, 0.4, 0.5, false)
	first := a.FindDynamicSynapse(b.ID, false)
	if first == nil {
		t.Fatal("expected dynamic connection")
	}
	firstTime := first.Dynamic.LastModification
	if firstTime.IsZero() {
		t.Fatal("expected modification timestamp")
	}

	time.Sleep(time.Millisecond)
	kb.Connect(a, b, 0.8, 0.9, false)

	connections := a.SynapsesTo(b.ID)
	if len(connections) != 1 {
		t.Fatalf("expected connection reuse, got %d connections", len(connections))
	}
	c := connections[0]
	if c.Dynamic.Frequency != 2 {
		t.Fatalf("expected frequency 2, got %d", c.Dynamic.Frequency)
	}
	if c.Dynamic.Weight <= 0.4 || c.Dynamic.Weight >= 0.8 {
		t.Fatalf("expected adaptive weight between observations, got %v", c.Dynamic.Weight)
	}
	if !c.Dynamic.LastModification.After(firstTime) {
		t.Fatal("expected modification trace to advance")
	}
}

func TestConnectKindRemainsCompatibilityOnly(t *testing.T) {
	kb := NewKnowledgeBase()
	a := kb.Store("a")
	b := kb.Store("b")

	kb.ConnectKind(a, b, RelationCause, 0.6, 0.7, false)

	c := a.FindSynapse(b.ID, RelationCause, false)
	if c == nil {
		t.Fatal("expected legacy semantic connection")
	}
	if c.IsDynamic() {
		t.Fatal("legacy semantic connection must not be classified as dynamic")
	}
	if c.Dynamic.Frequency != 1 {
		t.Fatalf("expected dynamic adaptive state to be initialized, got %d", c.Dynamic.Frequency)
	}
}
