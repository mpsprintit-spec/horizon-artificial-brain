package runtime

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestInterpretCognitiveProducesTypedAnswerWithoutRecommendation(t *testing.T) {
	now := time.Unix(100, 0).UTC()
	output := CognitiveOutput{
		BrainIdentity: BrainIdentity,
		Sequence: 7,
		Timestamp: now,
		RankedNodeIDs: []knowledge.NodeID{3, 8},
		Activations: map[knowledge.NodeID]float64{3: 0.9, 8: 0.4},
		Confidence: map[knowledge.NodeID]float64{3: 0.7, 8: 0.2},
		Resonance: 0.8,
		PredictionError: 0.1,
	}

	rt := NewBrainRuntime(knowledge.NewBrain())
	result, err := rt.InterpretCognitive(output, Observation{Source: "user", Modality: "text", Tokens: []string{"halo"}})
	if err != nil { t.Fatal(err) }
	if result.State.BrainIdentity != BrainIdentity { t.Fatalf("brain identity = %q", result.State.BrainIdentity) }
	if result.State.Sequence != 7 { t.Fatalf("sequence = %d", result.State.Sequence) }
	if len(result.Answer.NodeIDs) != 2 { t.Fatalf("answer node count = %d", len(result.Answer.NodeIDs)) }
	if result.Answer.Confidence != 0.7 { t.Fatalf("answer confidence = %v", result.Answer.Confidence) }
	if result.Recommendation != nil { t.Fatal("neural interpretation must not manufacture a recommendation") }
	if result.Answer.Uncertainty.PredictionError != 0.1 { t.Fatalf("prediction error = %v", result.Answer.Uncertainty.PredictionError) }
}

func TestInterpretCognitiveRejectsMissingBrainIdentity(t *testing.T) {
	rt := NewBrainRuntime(knowledge.NewBrain())
	_, err := rt.InterpretCognitive(CognitiveOutput{}, Observation{})
	if err == nil { t.Fatal("expected missing brain identity error") }
}
