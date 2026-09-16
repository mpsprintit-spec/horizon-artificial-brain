package runtime

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

func TestObserveOutcomeRequiresExplicitBinding(t *testing.T) {
	rt := NewBrainRuntime(nil)
	_, err := rt.ObserveOutcome(OutcomeEvent{RequestID: "req-1", Success: true, Observation: []string{"ok"}, Source: "sensor", Reliability: 0.9})
	if err == nil { t.Fatal("expected missing action binding error") }
}

func TestObserveOutcomeStaysCandidateUntilIndependentEvidence(t *testing.T) {
	rt := NewBrainRuntime(nil)
	node, _, err := rt.brain.Registry.GetOrCreate("target")
	if err != nil { t.Fatal(err) }
	beforeImportance := node.Importance
	now := time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC)

	if err := rt.RegisterActionBinding(ActionBinding{RequestID: "req-1", BrainIdentity: BrainIdentity, TargetNodeIDs: []knowledge.NodeID{node.ID}}); err != nil { t.Fatal(err) }
	if _, err := rt.ObserveOutcome(OutcomeEvent{RequestID: "req-1", Success: true, Observation: []string{"ok"}, Source: "sensor-a", Reliability: 0.9, ObservedAt: now}); err != nil { t.Fatal(err) }
	if node.Importance != beforeImportance { t.Fatalf("candidate outcome mutated importance: got %v want %v", node.Importance, beforeImportance) }

	if err := rt.RegisterActionBinding(ActionBinding{RequestID: "req-2", BrainIdentity: BrainIdentity, TargetNodeIDs: []knowledge.NodeID{node.ID}}); err != nil { t.Fatal(err) }
	if _, err := rt.ObserveOutcome(OutcomeEvent{RequestID: "req-2", Success: true, Observation: []string{"ok"}, Source: "sensor-b", Reliability: 0.9, ObservedAt: now.Add(time.Second)}); err != nil { t.Fatal(err) }
	if node.Importance <= beforeImportance { t.Fatalf("accepted evidence did not promote node: got %v want > %v", node.Importance, beforeImportance) }
}

func TestObserveOutcomeDoesNotDoubleCountSameSource(t *testing.T) {
	rt := NewBrainRuntime(nil)
	node, _, err := rt.brain.Registry.GetOrCreate("target")
	if err != nil { t.Fatal(err) }
	now := time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC)
	binding := ActionBinding{RequestID: "req-1", BrainIdentity: BrainIdentity, TargetNodeIDs: []knowledge.NodeID{node.ID}}
	if err := rt.RegisterActionBinding(binding); err != nil { t.Fatal(err) }

	outcome := OutcomeEvent{RequestID: "req-1", Success: true, Observation: []string{"ok"}, Source: "sensor-a", Reliability: 0.9, ObservedAt: now}
	if _, err := rt.ObserveOutcome(outcome); err != nil { t.Fatal(err) }
	if _, err := rt.ObserveOutcome(outcome); err != nil { t.Fatal(err) }

	combined, err := rt.evidence.Record(learning.OutcomeEvidence{RequestID: "req-1", NodeID: node.ID, Source: "sensor-a", Evidence: learning.Evidence{Weight: 0.6, Confidence: 0.5, Reliability: 0.9, IndependentSources: 1}})
	if err != nil { t.Fatal(err) }
	if combined.IndependentSources != 1 { t.Fatalf("same source was double-counted: got %d", combined.IndependentSources) }
}
