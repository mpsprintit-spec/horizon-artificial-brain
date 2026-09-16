package learning

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestPromotionAcceptedMutatesOnlyAfterPolicyAccepts(t *testing.T) {
	brain := knowledge.NewBrain()
	node := brain.Store("candidate")
	beforeImportance := node.Importance
	beforePlasticity := node.Plasticity

	engine := NewPromotionEngine(brain, DefaultLearningPolicy())
	decision, err := engine.Apply(node.ID, Evidence{
		Weight: 0.8, Confidence: 0.8, Reliability: 0.8,
		IndependentSources: 2,
	}, time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC))
	if err != nil { t.Fatal(err) }
	if decision != PromotionAccepted { t.Fatalf("expected accepted, got %s", decision) }
	if node.Importance <= beforeImportance || node.Plasticity <= beforePlasticity { t.Fatal("accepted evidence must strengthen the node") }
}

func TestPromotionCandidateDoesNotMutate(t *testing.T) {
	brain := knowledge.NewBrain()
	node := brain.Store("candidate")
	beforeImportance := node.Importance
	beforePlasticity := node.Plasticity

	engine := NewPromotionEngine(brain, DefaultLearningPolicy())
	decision, err := engine.Apply(node.ID, Evidence{Weight: 0.9, Confidence: 0.9, Reliability: 0.9, IndependentSources: 1}, time.Now())
	if err != nil { t.Fatal(err) }
	if decision != PromotionCandidate { t.Fatalf("expected candidate, got %s", decision) }
	if node.Importance != beforeImportance || node.Plasticity != beforePlasticity { t.Fatal("candidate evidence must not mutate the substrate") }
}

func TestPromotionRejectedSuppressesWithoutDeletingNode(t *testing.T) {
	brain := knowledge.NewBrain()
	node := brain.Store("contradicted")
	before := node.Importance

	engine := NewPromotionEngine(brain, DefaultLearningPolicy())
	decision, err := engine.Apply(node.ID, Evidence{Weight: 0.9, Confidence: 0.9, Reliability: 0.9, IndependentSources: 3, Contradictions: 1}, time.Now())
	if err != nil { t.Fatal(err) }
	if decision != PromotionRejected { t.Fatalf("expected rejected, got %s", decision) }
	if brain.Fetch("contradicted") == nil { t.Fatal("rejected candidate must remain inspectable") }
	if node.Importance >= before { t.Fatal("rejected evidence should suppress importance") }
}
