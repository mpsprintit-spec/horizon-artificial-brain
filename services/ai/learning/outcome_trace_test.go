package learning

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestLearnOutcomeTracePreservesContradictoryOutcomes(t *testing.T) {
	brain := knowledge.NewBrain()
	unit := NewLearningUnit(brain)
	target, _, err := brain.Registry.GetOrCreate("target")
	if err != nil {
		t.Fatal(err)
	}
	outcomeB, _, err := brain.Registry.GetOrCreate("outcome-b")
	if err != nil {
		t.Fatal(err)
	}
	outcomeC, _, err := brain.Registry.GetOrCreate("outcome-c")
	if err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC)
	evidence := knowledge.ExperienceEvidence{
		ExperienceID: "inquiry-1",
		Source: "inquiry-consequence",
		Modality: "outcome",
		Timestamp: now,
		Reliability: 0.9,
		CausalLink: "inquiry:req-1",
	}

	unit.LearnOutcomeTrace([]knowledge.NodeID{target.ID}, []knowledge.NodeID{outcomeB.ID}, 0.8, 0.9, evidence, now)
	unit.LearnOutcomeTrace([]knowledge.NodeID{target.ID}, []knowledge.NodeID{outcomeB.ID}, 0.8, 0.9, evidence, now.Add(time.Second))
	unit.LearnOutcomeTrace([]knowledge.NodeID{target.ID}, []knowledge.NodeID{outcomeC.ID}, 0.8, 0.9, evidence, now.Add(2*time.Second))

	b := target.FindDynamicSynapse(outcomeB.ID, false)
	c := target.FindDynamicSynapse(outcomeC.ID, false)
	if b == nil || c == nil {
		t.Fatal("contradictory outcomes must remain as separate dynamic connections")
	}
	if b.Dynamic.Frequency < 2 {
		t.Fatalf("repeated B outcome was not reinforced: frequency=%d", b.Dynamic.Frequency)
	}
	if c.Dynamic.Frequency != 1 {
		t.Fatalf("C outcome frequency=%d, want 1", c.Dynamic.Frequency)
	}
}
