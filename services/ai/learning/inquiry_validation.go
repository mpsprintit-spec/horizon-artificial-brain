package learning

import (
	"errors"
	"strings"
	"time"
)

// InquirySource is the minimal external provenance contract required by the
// learning layer. The websearch package adapts to this shape at the boundary,
// avoiding a dependency cycle between perception and learning.
type InquirySource struct {
	Source     string
	Reputation float64
	Confidence float64
}

// InquiryCandidate is an untrusted experience assembled from an external
// inquiry. Creating a candidate does not mutate the neural substrate.
type InquiryCandidate struct {
	Experience Experience
	Evidence   Evidence
	Sources    []string
}

// BuildInquiryCandidate converts independent external observations into a
// candidate experience. Each distinct non-empty source contributes at most one
// independent support unit.
func BuildInquiryCandidate(sequence []string, results []InquirySource, modality string, now time.Time) (InquiryCandidate, error) {
	if len(sequence) == 0 {
		return InquiryCandidate{}, errors.New("inquiry candidate sequence is empty")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()

	var weight, confidence, reliability float64
	var sources []string
	seen := make(map[string]struct{})
	for _, result := range results {
		source := strings.TrimSpace(result.Source)
		if source == "" {
			continue
		}
		if _, exists := seen[source]; exists {
			continue
		}
		seen[source] = struct{}{}
		sources = append(sources, source)
		if result.Confidence > confidence {
			confidence = result.Confidence
		}
		if result.Reputation > reliability {
			reliability = result.Reputation
		}
	}
	if len(sources) == 0 {
		return InquiryCandidate{}, errors.New("inquiry produced no attributable sources")
	}
	if confidence == 0 {
		confidence = 0.5
	}
	if reliability == 0 {
		reliability = 0.5
	}
	weight = confidence
	if reliability < weight {
		weight = reliability
	}

	return InquiryCandidate{
		Experience: Experience{
			Sequence:          append([]string(nil), sequence...),
			Weight:            clamp01(weight),
			Confidence:        clamp01(confidence),
			ExperienceID:      "inquiry-" + sources[0],
			Source:            sources[0],
			Modality:          modality,
			Timestamp:         now,
			Reliability:       clamp01(reliability),
			IndependenceGroup: "inquiry",
		},
		Evidence: Evidence{
			Weight:             clamp01(weight),
			Confidence:         clamp01(confidence),
			Reliability:        clamp01(reliability),
			IndependentSources: len(sources),
		},
		Sources: sources,
	}, nil
}

// ValidateInquiry applies the existing learning policy without mutating Brain.
// Rejected/candidate results must be handled by the caller; only an accepted
// result may be passed to LearnExperience.
func ValidateInquiry(candidate InquiryCandidate, policy LearningPolicy) PromotionDecision {
	return policy.Evaluate(candidate.Evidence)
}
