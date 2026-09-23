package runtime

import "math"

// ConsequenceInput is the bounded, outcome-level signal used to derive a
// consequence profile. It describes what happened; it does not authorize
// future action.
type ConsequenceInput struct {
	Success bool
	Reliability float64
	Reversible bool
	Cost float64
	Risk float64
	InformationGain float64
	PredictionError float64
}

// ConsequenceAssessment is a derived consequence signal. Valence is normalized
// to [-1,1] and is deliberately kept separate from confidence and authorization.
type ConsequenceAssessment struct {
	Success bool
	Reliability float64
	Reversible bool
	Cost float64
	Risk float64
	InformationGain float64
	PredictionError float64
	Valence float64
}

// AssessConsequence derives bounded valence from an observed outcome.
// Success/failure supplies the primary direction; reliability, cost and risk
// modulate magnitude. Information gain is retained as an outcome dimension,
// but is not silently treated as positive or negative valence.
func AssessConsequence(input ConsequenceInput) ConsequenceAssessment {
	reliability := clamp01(input.Reliability)
	cost := clamp01(input.Cost)
	risk := clamp01(input.Risk)
	informationGain := clamp01(input.InformationGain)
	errorSignal := clamp01(input.PredictionError)

	base := -1.0
	if input.Success {
		base = 1.0
	}
	modifier := 1.0 - (0.35 * cost) - (0.25 * risk)
	if input.Success {
		modifier += 0.15 * errorSignal
	} else {
		modifier -= 0.15 * errorSignal
	}
	valence := base * reliability * modifier
	if math.IsNaN(valence) || math.IsInf(valence, 0) {
		valence = 0
	}
	if valence < -1 { valence = -1 }
	if valence > 1 { valence = 1 }
	return ConsequenceAssessment{
		Success: input.Success,
		Reliability: reliability,
		Reversible: input.Reversible,
		Cost: cost,
		Risk: risk,
		InformationGain: informationGain,
		PredictionError: errorSignal,
		Valence: valence,
	}
}
