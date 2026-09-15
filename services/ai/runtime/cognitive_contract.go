package runtime

import "github.com/project-horizon/horizon-core/services/ai/knowledge"

// Observation is the substrate-neutral record of what the runtime received or
// detected. It deliberately carries no semantic rule or action permission.
type Observation struct {
	Source        string
	Modality      string
	Tokens        []string
	ContextTokens []string
	DataTokens    []string
}

// Uncertainty describes limitations of the current neural state. It is not an
// authorization signal and must not be converted into permission to act.
type Uncertainty struct {
	Level           float64
	PredictionError float64
	Reason          string
}

// Answer is a presentation candidate derived from the current cognitive
// interpretation. It is distinct from an action recommendation.
type Answer struct {
	NodeIDs     []knowledge.NodeID
	Confidence  float64
	Uncertainty Uncertainty
}

// CognitiveInterpretation is the typed boundary between neural state and
// externally usable cognition. An empty Recommendation means no action has
// been proposed; ordinary answers therefore never require safety execution.
type CognitiveInterpretation struct {
	Observation    Observation
	State          CognitiveState
	Interpretation Interpretation
	Answer         Answer
	Recommendation *Recommendation
}

// ToCognitiveInterpretation packages an already validated neural
// Interpretation without introducing semantic rules. Recommendation remains
// nil because neural activation alone is not sufficient authorization or an
// action instruction.
func ToCognitiveInterpretation(observation Observation, interpretation Interpretation) CognitiveInterpretation {
	return CognitiveInterpretation{
		Observation: observation,
		State: CognitiveState{
			BrainIdentity:  interpretation.BrainIdentity,
			Sequence:       interpretation.Sequence,
			ActiveNodeIDs:  append([]knowledge.NodeID(nil), interpretation.RankedNodeIDs...),
			Activations:    cloneNodeValues(interpretation.Activations),
			Confidence:     cloneNodeValues(interpretation.Confidence),
			Resonance:      interpretation.Resonance,
			PredictionError: interpretation.PredictionError,
		},
		Interpretation: interpretation,
		Answer: Answer{
			NodeIDs:    append([]knowledge.NodeID(nil), interpretation.RankedNodeIDs...),
			Confidence: bestInterpretationConfidence(interpretation),
			Uncertainty: Uncertainty{
				Level:           1 - interpretation.Resonance,
				PredictionError: interpretation.PredictionError,
			},
		},
	}
}

func bestInterpretationConfidence(interpretation Interpretation) float64 {
	best := 0.0
	for _, id := range interpretation.RankedNodeIDs {
		if value := interpretation.Confidence[id]; value > best {
			best = value
		}
	}
	return best
}
