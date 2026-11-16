package runtime

import (
	"errors"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// CognitiveInterpreter is the only interpretation boundary allowed to turn
// neural runtime state into a cognitive candidate. Implementations must not
// mutate the brain substrate.
type CognitiveInterpreter interface {
	Interpret(CognitiveOutput) (Interpretation, error)
}

// Interpretation is deliberately substrate-oriented. It exposes the neural
// result without assigning hard-coded semantic relations to nodes.
type Interpretation struct {
	PredictionAssessment *knowledge.PredictionAssessment `json:"prediction_assessment,omitempty"`
	BrainIdentity        string
	Sequence             uint64
	RankedNodeIDs        []knowledge.NodeID
	Activations          map[knowledge.NodeID]float64
	Confidence           map[knowledge.NodeID]float64
	Resonance            float64
	PredictionError      float64
	Source               string
}

// NeuralInterpreter is the default Gate-4 interpreter. It accepts only the
// output produced by BrainRuntime and performs no legacy semantic inference.
type NeuralInterpreter struct{}

func (NeuralInterpreter) Interpret(output CognitiveOutput) (Interpretation, error) {
	if output.BrainIdentity == "" {
		return Interpretation{}, errors.New("cognitive output has no brain identity")
	}
	return Interpretation{
		BrainIdentity:        output.BrainIdentity,
		Sequence:             output.Sequence,
		RankedNodeIDs:        append([]knowledge.NodeID(nil), output.RankedNodeIDs...),
		Activations:          cloneNodeValues(output.Activations),
		Confidence:           cloneNodeValues(output.Confidence),
		Resonance:            output.Resonance,
		PredictionError:      output.PredictionError,
		Source:               "neural",
		PredictionAssessment: output.PredictionAssessment.Clone(),
	}, nil
}

// InterpretCognitive is the typed P0 boundary. It packages the neural result
// into Observation/State/Interpretation/Answer. Recommendation remains nil
// unless a separate explicit action-producing stage creates one. Neural
// activation alone is never treated as an action instruction.
func (r *BrainRuntime) InterpretCognitive(output CognitiveOutput, observation Observation) (CognitiveInterpretation, error) {
	interpretation, err := r.Interpret(output, NeuralInterpreter{})
	if err != nil {
		return CognitiveInterpretation{}, err
	}
	result := ToCognitiveInterpretation(observation, interpretation)
	if output.Composition != nil {
		copied, _ := freezeComposition(*output.Composition)
		result.Composition = &copied
	}
	result.State.Timestamp = output.Timestamp
	return result, nil
}

// Interpret is the explicit Gate-4 entry point. Neural interpretation is
// always the default; a caller may supply a presentation adapter.
func (r *BrainRuntime) Interpret(output CognitiveOutput, interpreter CognitiveInterpreter) (Interpretation, error) {
	if interpreter == nil {
		interpreter = NeuralInterpreter{}
	}
	return interpreter.Interpret(output)
}
