package runtime

import "math"

// ConsequenceInput is the bounded, outcome-level signal used to derive
// valence. It describes what happened; it does not authorize future action.
type ConsequenceInput struct {
	Success bool
	Reliability float64
	Reversible bool
	Cost float64
	Risk float64
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
	PredictionError float64
	Valence float64
}

// AssessConsequence derives bounded valence from an observed outcome.
// Success/failure supplies the primary direction; cost, risk and prediction
// error modulate magnitude. Prediction error alone cannot make a successful
// outcome negative or a failed outcome positive.
func AssessConsequence(input ConsequenceInput) ConsequenceAssessment {
	reliability := clamp01(input.Reliability)
	cost := clamp01(input.Cost)
	risk := clamp01(input.Risk)
	errorSignal := clamp01(input.PredictionError)

	base := -1.0
	if input.Success {
		base = 1.0
	}
	// Reliability scales the outcome signal. Cost/risk are consequence
	// penalties rather than independent semantic judgments.
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
		PredictionError: errorSignal,
		Valence: valence,
	}
}
