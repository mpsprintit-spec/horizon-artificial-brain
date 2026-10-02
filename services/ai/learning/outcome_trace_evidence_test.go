package learning

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestLearnOutcomeTracePreservesEvidenceInCanonicalPattern(t *testing.T) {
	brain := knowledge.NewBrain()
	learner := NewLearningUnit(brain)
	target := brain.Store("target")
	outcome := brain.Store("outcome")
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)

	evidence := knowledge.ExperienceEvidence{
		ExperienceID:      "exp-1",
		Source:            "inquiry-consequence",
		Modality:          "outcome",
		Timestamp:         now,
		Reliability:       0.9,
		IndependenceGroup: "causal-1",
		CausalLink:        "inquiry:focus",
		ContradictionSet:  "set-1",
	}

	learner.LearnOutcomeTrace(
		[]knowledge.NodeID{target.ID},
		[]knowledge.NodeID{outcome.ID},
		0.8,
		0.9,
		evidence,
		now,
	)

	patterns := brain.Patterns.All()
	if len(patterns) != 1 {
		t.Fatalf("expected one canonical pattern trace, got %d", len(patterns))
	}
	if len(patterns[0].Evidence) != 1 {
		t.Fatalf("expected one preserved evidence record, got %d", len(patterns[0].Evidence))
	}
	got := patterns[0].Evidence[0]
	if got.ExperienceID != evidence.ExperienceID ||
		got.Source != evidence.Source ||
		got.IndependenceGroup != evidence.IndependenceGroup ||
		got.CausalLink != evidence.CausalLink ||
		got.ContradictionSet != evidence.ContradictionSet {
		t.Fatalf("evidence provenance was not preserved: got %#v", got)
	}
}
