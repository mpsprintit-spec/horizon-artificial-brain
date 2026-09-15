package knowledge

import "math"

// CalibratedConfidence converts learned confidence into bounded evidence-aware
// confidence. Evidence groups are counted once, while reliability and
// contradiction remain separate signals. Repeated observations from one
// source/group therefore cannot masquerade as independent confirmation.
func CalibratedConfidence(base float64, evidence []ExperienceEvidence) float64 {
	base = clamp01(base)
	if len(evidence) == 0 {
		return base
	}

	independent := EvidenceIndependenceBounded(evidence)
	if independent <= 0 {
		return base
	}

	reliability := EvidenceReliability(evidence)
	// Saturating support: additional independent groups have diminishing
	// influence rather than pushing confidence linearly toward one.
	support := 1 - math.Exp(-float64(independent))
	confidence := base + (1-base)*support*reliability

	// Contradictory evidence is retained and reduces current calibrated
	// confidence; it never deletes or overwrites the underlying evidence.
	contradiction := 0.0
	sets := make(map[string]struct{})
	for _, item := range evidence {
		if item.ContradictionSet != "" {
			sets[item.ContradictionSet] = struct{}{}
		}
	}
	for set := range sets {
		contradiction = math.Max(contradiction, ContradictionLoad(evidence, set))
	}
	confidence *= 1 - 0.5*contradiction
	return clamp01(confidence)
}
