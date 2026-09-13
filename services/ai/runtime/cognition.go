package runtime

import (
	"time"

	"github.com/project-horizon/horizon-core/services/ai/activation"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// CognitiveOutput is the neutral boundary between the neural runtime and
// downstream interpretation/policy layers. It contains only neural state and
// evidence; it deliberately carries no RelationKind, semantic rule, or
// execution instruction.
type CognitiveOutput struct {
	BrainIdentity string
	Sequence      uint64
	Timestamp     time.Time

	RankedNodeIDs []knowledge.NodeID
	Activations   map[knowledge.NodeID]float64
	Confidence    map[knowledge.NodeID]float64
	Resonance     float64

	PredictionError float64
	Prediction      activation.Prediction
}

func cognitiveOutputFromResult(identity string, sequence uint64, now time.Time, result activation.Result) CognitiveOutput {
	ids := make([]knowledge.NodeID, 0, len(result.RankedNodes))
	for _, node := range result.RankedNodes {
		if node != nil {
			ids = append(ids, node.ID)
		}
	}
	return CognitiveOutput{
		BrainIdentity: identity,
		Sequence:      sequence,
		Timestamp:     now,
		RankedNodeIDs: ids,
		Activations:   cloneNodeValues(result.Activations),
		Confidence:    cloneNodeValues(result.Confidence),
		Resonance:     result.Resonance,
	}
}

// CognitiveProcess performs the official neural transition and exposes its
// result through the neutral cognition boundary.
func (r *BrainRuntime) CognitiveProcess(event Event) (CognitiveOutput, error) {
	result, sequence, err := r.Process(event)
	if err != nil {
		return CognitiveOutput{}, err
	}
	now := event.Timestamp
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return cognitiveOutputFromResult(r.BrainIdentity(), sequence, now, result), nil
}

// CognitiveThink advances the same persistent neural state without requiring a
// new external stimulus.
func (r *BrainRuntime) CognitiveThink(cycles int) (CognitiveOutput, error) {
	thought, sequence, err := r.Think(cycles)
	if err != nil {
		return CognitiveOutput{}, err
	}
	return CognitiveOutput{
		BrainIdentity:   r.BrainIdentity(),
		Sequence:        sequence,
		Timestamp:       time.Now().UTC(),
		RankedNodeIDs:   rankedIDs(thought.RankedNodes),
		Activations:     cloneNodeValues(thought.Activations),
		Confidence:      cloneNodeValues(thought.Confidence),
		Resonance:       thought.Resonance,
		PredictionError: thought.PredictionError,
		Prediction:      thought.Prediction,
	}, nil
}

func rankedIDs(nodes []*knowledge.ConceptNode) []knowledge.NodeID {
	ids := make([]knowledge.NodeID, 0, len(nodes))
	for _, node := range nodes {
		if node != nil {
			ids = append(ids, node.ID)
		}
	}
	return ids
}

func cloneNodeValues(in map[knowledge.NodeID]float64) map[knowledge.NodeID]float64 {
	if in == nil {
		return map[knowledge.NodeID]float64{}
	}
	out := make(map[knowledge.NodeID]float64, len(in))
	for id, value := range in {
		out[id] = value
	}
	return out
}
