package learning

import "testing"

func TestDefaultPolicyKeepsSingleObservationAsCandidate(t *testing.T) {
	p := DefaultLearningPolicy()
	got := p.Evaluate(Evidence{
		Weight:             0.8,
		Confidence:         0.8,
		Reliability:        0.8,
		IndependentSources: 1,
	})
	if got != PromotionCandidate {
		t.Fatalf("single-source evidence must remain candidate, got %q", got)
	}
}

func TestPolicyAcceptsConvergentEvidence(t *testing.T) {
	p := DefaultLearningPolicy()
	got := p.Evaluate(Evidence{
		Weight:             0.8,
		Confidence:         0.8,
		Reliability:        0.8,
		IndependentSources: 2,
	})
	if got != PromotionAccepted {
		t.Fatalf("convergent evidence should be accepted, got %q", got)
	}
}

func TestPolicyRejectsContradiction(t *testing.T) {
	p := DefaultLearningPolicy()
	got := p.Evaluate(Evidence{
		Weight:             0.9,
		Confidence:         0.9,
		Reliability:        0.9,
		IndependentSources: 3,
		Contradictions:     1,
	})
	if got != PromotionRejected {
		t.Fatalf("contradictory evidence must be rejected, got %q", got)
	}
}

func TestMergeEvidencePreservesContradictions(t *testing.T) {
	merged := MergeEvidence(
		Evidence{Weight: 0.4, Confidence: 0.5, Reliability: 0.6, IndependentSources: 1, Contradictions: 0},
		Evidence{Weight: 0.7, Confidence: 0.6, Reliability: 0.5, IndependentSources: 1, Contradictions: 1},
	)
	if merged.Weight != 0.7 || merged.Confidence != 0.6 || merged.Reliability != 0.6 {
		t.Fatal("merge must retain strongest evidence values")
	}
	if merged.IndependentSources != 2 || merged.Contradictions != 1 {
		t.Fatal("merge must retain support and contradiction counts")
	}
}
