package runtime

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestOutcomePromotionReinforcesOnlyBoundDynamicSynapse(t *testing.T) {
	rt := NewBrainRuntime(nil)
	source, _, err := rt.brain.Registry.GetOrCreate("source")
	if err != nil { t.Fatal(err) }
	target, _, err := rt.brain.Registry.GetOrCreate("target")
	if err != nil { t.Fatal(err) }
	rt.brain.Connect(source, target, 0.20, 0.30, false)
	other, _, err := rt.brain.Registry.GetOrCreate("other")
	if err != nil { t.Fatal(err) }
	rt.brain.Connect(source, other, 0.80, 0.80, false)

	before := source.Synapses[target.ID][0].Weight
	untouched := other.Importance
	now := time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC)
	binding := ActionBinding{RequestID: "req-s1", BrainIdentity: BrainIdentity, TargetNodeIDs: []knowledge.NodeID{target.ID}, Synapses: []SynapseBinding{{SourceNodeID: source.ID, TargetNodeID: target.ID}}}
	if err := rt.RegisterActionBinding(binding); err != nil { t.Fatal(err) }
	if _, err := rt.ObserveOutcome(OutcomeEvent{RequestID: "req-s1", Success: true, Observation: []string{"ok"}, Source: "sensor-a", Reliability: 0.9, ObservedAt: now}); err != nil { t.Fatal(err) }
	if source.Synapses[target.ID][0].Weight != before { t.Fatal("single candidate outcome changed synapse") }

	if err := rt.RegisterActionBinding(ActionBinding{RequestID: "req-s2", BrainIdentity: BrainIdentity, TargetNodeIDs: []knowledge.NodeID{target.ID}, Synapses: []SynapseBinding{{SourceNodeID: source.ID, TargetNodeID: target.ID}}}); err != nil { t.Fatal(err) }
	if _, err := rt.ObserveOutcome(OutcomeEvent{RequestID: "req-s2", Success: true, Observation: []string{"ok"}, Source: "sensor-b", Reliability: 0.9, ObservedAt: now.Add(time.Second)}); err != nil { t.Fatal(err) }
	if source.Synapses[target.ID][0].Weight <= before { t.Fatal("accepted evidence did not reinforce bound synapse") }
	if other.Importance != untouched { t.Fatal("unbound node was mutated") }
}
