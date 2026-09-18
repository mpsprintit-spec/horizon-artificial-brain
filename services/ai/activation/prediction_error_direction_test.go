package activation

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestPredictionErrorPlasticityFollowsErrorDirection(t *testing.T) {
	brain := knowledge.NewBrain()
	source := brain.Store("source")
	target := brain.Store("target")
	brain.Connect(source, target, 0.50, 0.80, false)

	synapse := source.OutboundAll()[0]
	synapse.Dynamic.Eligibility = 1
	t0 := time.Unix(500, 0).UTC()
	synapse.Dynamic.LastEligibilityUpdate = t0

	engine := NewEngine(brain)

	// Positive surprise: actual target activity exceeds the prediction.
	engine.ApplyPredictionErrorPlasticity(
		map[knowledge.NodeID]float64{target.ID: 0.20},
		map[knowledge.NodeID]float64{target.ID: 0.90},
		1,
		t0,
	)
	strengthened := synapse.Dynamic.Weight
	if strengthened <= 0.50 {
		t.Fatalf("expected positive prediction error to strengthen pathway: %v", strengthened)
	}

	// Reset the same canonical synapse, then apply the opposite error.
	synapse.Dynamic.Weight = 0.50
	synapse.Weight = 0.50
	synapse.Dynamic.Confidence = 0.80
	synapse.Confidence = 0.80
	synapse.Dynamic.Eligibility = 1
	synapse.Dynamic.LastEligibilityUpdate = t0

	engine.ApplyPredictionErrorPlasticity(
		map[knowledge.NodeID]float64{target.ID: 0.90},
		map[knowledge.NodeID]float64{target.ID: 0.20},
		1,
		t0,
	)
	weakened := synapse.Dynamic.Weight
	if weakened >= 0.50 {
		t.Fatalf("expected negative prediction error to weaken pathway: %v", weakened)
	}
	if weakened < predictionMinWeight || weakened > predictionMaxWeight {
		t.Fatalf("negative prediction error escaped bounded weight range: %v", weakened)
	}
}

func TestPredictionErrorPlasticityIsNoOpWithoutError(t *testing.T) {
	brain := knowledge.NewBrain()
	source := brain.Store("source")
	target := brain.Store("target")
	brain.Connect(source, target, 0.50, 0.80, false)

	synapse := source.OutboundAll()[0]
	synapse.Dynamic.Eligibility = 1
	t0 := time.Unix(600, 0).UTC()
	synapse.Dynamic.LastEligibilityUpdate = t0
	beforeWeight := synapse.Dynamic.Weight
	beforeConfidence := synapse.Dynamic.Confidence
	beforeEligibility := synapse.Dynamic.Eligibility

	NewEngine(brain).ApplyPredictionErrorPlasticity(
		map[knowledge.NodeID]float64{target.ID: 0.50},
		map[knowledge.NodeID]float64{target.ID: 0.50},
		0,
		t0,
	)

	if synapse.Dynamic.Weight != beforeWeight ||
		synapse.Dynamic.Confidence != beforeConfidence ||
		synapse.Dynamic.Eligibility != beforeEligibility {
		t.Fatalf("zero prediction error changed synaptic state: weight=%v/%v confidence=%v/%v eligibility=%v/%v",
			synapse.Dynamic.Weight, beforeWeight,
			synapse.Dynamic.Confidence, beforeConfidence,
			synapse.Dynamic.Eligibility, beforeEligibility)
	}
}
