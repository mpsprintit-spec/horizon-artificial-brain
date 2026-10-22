package runtime

import (
	"math"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

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

type Answer struct {
	PredictionAssessment *knowledge.PredictionAssessment `json:"prediction_assessment,omitempty"`
	NodeIDs              []knowledge.NodeID
	Confidence           float64
	Uncertainty          Uncertainty
}

// CognitiveInterpretation is the typed boundary between neural state and externally usable cognition. Grounding records how raw context/data entered the shared substrate; they do not assert semantic truth or authorization.
type CognitiveInterpretation struct {
	SensorBindings          []knowledge.SensoryFieldBinding `json:"sensor_bindings,omitempty"`
	Observation             Observation
	GroundedRepresentations []knowledge.GroundedRepresentation
	State                   CognitiveState
	Interpretation          Interpretation
	Answer                  Answer
	Recommendation          *Recommendation
	Consequence             *ConsequenceAssessment
	ChangeAwareness         InternalChangeAwareness
	InquiryAgenda           *InquiryAgenda
}

func ToCognitiveInterpretation(observation Observation, interpretation Interpretation) CognitiveInterpretation {
	resonance := clamp01(interpretation.Resonance)
	predictionError := clamp01(interpretation.PredictionError)
	result := CognitiveInterpretation{
		Observation: observation,
		State: CognitiveState{
			BrainIdentity:   interpretation.BrainIdentity,
			Sequence:        interpretation.Sequence,
			ActiveNodeIDs:   append([]knowledge.NodeID(nil), interpretation.RankedNodeIDs...),
			Activations:     cloneNodeValues(interpretation.Activations),
			Confidence:      cloneNodeValues(interpretation.Confidence),
			Resonance:       resonance,
			PredictionError: predictionError,
		},
		Interpretation: interpretation,
		Answer: Answer{
			NodeIDs:    append([]knowledge.NodeID(nil), interpretation.RankedNodeIDs...),
			Confidence: bestInterpretationConfidence(interpretation),
			Uncertainty: Uncertainty{
				Level:           1 - resonance,
				PredictionError: predictionError,
				Reason:          uncertaintyReason(resonance, predictionError),
			},
		},
	}
	if assessment := interpretation.PredictionAssessment; assessment != nil {
		result.Answer.PredictionAssessment = assessment.Clone()
		confidence := assessment.Confidence
		if math.IsNaN(confidence) || math.IsInf(confidence, 0) {
			confidence = 0
		}
		result.Answer.Confidence = clamp01(confidence)
		result.Answer.Uncertainty.Level = 1 - result.Answer.Confidence
		result.Answer.Uncertainty.Reason = predictionUncertaintyReason(assessment)
	}
	return result
}

func predictionUncertaintyReason(assessment *knowledge.PredictionAssessment) string {
	switch {
	case assessment.EvidenceMass <= 0 || len(assessment.NodeIDs) == 0:
		return "no-matching-outcome-evidence"
	case assessment.ContextSupport < .5:
		return "weak-context-support"
	case len(assessment.Alternatives) > 1 && assessment.Confidence <= .5:
		return "competing-outcomes"
	case assessment.Confidence < .5:
		return "limited-outcome-evidence"
	default:
		return "supported-neural-prediction"
	}
}

func bestInterpretationConfidence(interpretation Interpretation) float64 {
	best := 0.0
	for _, id := range interpretation.RankedNodeIDs {
		if value := interpretation.Confidence[id]; value > best {
			best = value
		}
	}
	return clamp01(best)
}

func uncertaintyReason(resonance, predictionError float64) string {
	switch {
	case predictionError >= 0.5 && resonance < 0.5:
		return "high-prediction-error-and-low-resonance"
	case predictionError >= 0.5:
		return "high-prediction-error"
	case resonance < 0.5:
		return "low-resonance"
	default:
		return "within-neural-range"
	}
}
