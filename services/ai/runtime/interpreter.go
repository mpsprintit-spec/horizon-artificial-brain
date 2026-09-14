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
	BrainIdentity   string
	Sequence        uint64
	RankedNodeIDs   []knowledge.NodeID
	Activations     map[knowledge.NodeID]float64
	Confidence      map[knowledge.NodeID]float64
	Resonance       float64
	PredictionError float64
	Source          string
}

// NeuralInterpreter is the default Gate-4 interpreter. It accepts only the
// output produced by BrainRuntime and performs no legacy semantic inference.
type NeuralInterpreter struct{}

func (NeuralInterpreter) Interpret(output CognitiveOutput) (Interpretation, error) {
	if output.BrainIdentity == "" {
		return Interpretation{}, errors.New("cognitive output has no brain identity")
	}
	return Interpretation{
		BrainIdentity:   output.BrainIdentity,
		Sequence:        output.Sequence,
		RankedNodeIDs:   append([]knowledge.NodeID(nil), output.RankedNodeIDs...),
		Activations:     cloneNodeValues(output.Activations),
		Confidence:      cloneNodeValues(output.Confidence),
		Resonance:       output.Resonance,
		PredictionError: output.PredictionError,
		Source:          "neural",
	}, nil
}

// LegacyCompatibilityInterpreter is intentionally opt-in. It is a boundary
// adapter only; it has no access to BrainRuntime mutation and cannot become
// the default source of cognition.
type LegacyCompatibilityInterpreter struct {
	Delegate func(CognitiveOutput) (Interpretation, error)
}

func (l LegacyCompatibilityInterpreter) Interpret(output CognitiveOutput) (Interpretation, error) {
	if l.Delegate == nil {
		return Interpretation{}, errors.New("legacy compatibility interpreter is disabled")
	}
	result, err := l.Delegate(output)
	if err != nil {
		return Interpretation{}, err
	}
	result.Source = "legacy-compatibility"
	return result, nil
}

// Interpret is the explicit Gate-4 entry point. Neural interpretation is
// always the default; legacy interpretation must be supplied by the caller.
func (r *BrainRuntime) Interpret(output CognitiveOutput, interpreter CognitiveInterpreter) (Interpretation, error) {
	if interpreter == nil {
		interpreter = NeuralInterpreter{}
	}
	return interpreter.Interpret(output)
}
