package learning

// Evidence is the minimum policy input required to decide whether a neural
// candidate has accumulated enough independent support. It is deliberately
// separate from the neural substrate: observation frequency alone is not proof.
type Evidence struct {
	Weight            float64
	Confidence        float64
	Reliability       float64
	IndependentSources int
	Contradictions    int
}

// PromotionDecision is a policy result, not a learning mutation. The caller
// decides whether and how the accepted result should change the substrate.
type PromotionDecision string

const (
	PromotionCandidate PromotionDecision = "candidate"
	PromotionAccepted  PromotionDecision = "accepted"
	PromotionRejected  PromotionDecision = "rejected"
)

type LearningPolicy struct {
	MinWeight             float64
	MinConfidence         float64
	MinReliability        float64
	MinIndependentSources int
	MaxContradictions     int
}

func DefaultLearningPolicy() LearningPolicy {
	return LearningPolicy{
		MinWeight:             0.60,
		MinConfidence:         0.60,
		MinReliability:        0.60,
		MinIndependentSources: 2,
		MaxContradictions:     0,
	}
}

// Evaluate never mutates memory. A candidate is promoted only when all
// required evidence dimensions pass. This prevents one strong observation
// from becoming permanent knowledge by itself.
func (p LearningPolicy) Evaluate(e Evidence) PromotionDecision {
	if e.Contradictions > p.MaxContradictions {
		return PromotionRejected
	}
	if e.Weight < p.MinWeight ||
		e.Confidence < p.MinConfidence ||
		e.Reliability < p.MinReliability ||
		e.IndependentSources < p.MinIndependentSources {
		return PromotionCandidate
	}
	return PromotionAccepted
}

// MergeEvidence accumulates independent support while keeping contradiction
// count explicit. It is intentionally conservative: confidence and reliability
// use the stronger evidence, while support count is additive.
func MergeEvidence(a, b Evidence) Evidence {
	return Evidence{
		Weight:             max(a.Weight, b.Weight),
		Confidence:         max(a.Confidence, b.Confidence),
		Reliability:        max(a.Reliability, b.Reliability),
		IndependentSources: a.IndependentSources + b.IndependentSources,
		Contradictions:     a.Contradictions + b.Contradictions,
	}
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
