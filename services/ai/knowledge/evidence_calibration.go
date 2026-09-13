package knowledge

import "math"

// CalibratedConfidence converts learned confidence into an evidence-aware
// confidence. Repetition from the same evidence source/independence group does
// not count as independent confirmation. This is calibration metadata, not a
// semantic rule and it does not alter the neural topology.
func CalibratedConfidence(base float64, evidence []ExperienceEvidence) float64 {
	base = clamp01(base)
	if len(evidence) == 0 {
		return base
	}

	groups := make(map[string]struct{}, len(evidence))
	reliability := 0.0
	count := 0
	for _, item := range evidence {
		group := item.IndependenceGroup
		if group == "" {
			group = item.Source + "|" + item.Modality
		}
		if group == "" {
			group = "unknown"
		}
		groups[group] = struct{}{}
		reliability += clamp01(item.Reliability)
		count++
	}
	if count == 0 {
		return base
	}

	meanReliability := reliability / float64(count)
	independenceSupport := 1 - math.Exp(-float64(len(groups)))
	return clamp01(base * meanReliability * independenceSupport)
}
