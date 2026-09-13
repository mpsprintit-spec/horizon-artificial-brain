package knowledge

import "time"

// ExperienceEvidence records provenance and reliability without becoming a
// semantic relation or a cognitive rule. It describes where an experience
// came from and how independent/reliable that evidence is.
type ExperienceEvidence struct {
	ExperienceID      string    `json:"experience_id,omitempty"`
	Source            string    `json:"source,omitempty"`
	Modality          string    `json:"modality,omitempty"`
	Timestamp         time.Time `json:"timestamp,omitempty"`
	Reliability       float64   `json:"reliability,omitempty"`
	IndependenceGroup string    `json:"independence_group,omitempty"`
	CausalLink        string    `json:"causal_link,omitempty"`
	ContradictionSet  string    `json:"contradiction_set,omitempty"`
}

// EvidenceIndependence counts independent evidence groups. Evidence without
// a group is conservatively treated as independent only from other anonymous
// evidence; repeated evidence with the same group does not inflate the count.
func EvidenceIndependence(evidence []ExperienceEvidence) int {
	groups := make(map[string]struct{}, len(evidence))
	anonymous := 0
	for _, e := range evidence {
		if e.IndependenceGroup == "" {
			anonymous++
			continue
		}
		groups[e.IndependenceGroup] = struct{}{}
	}
	return len(groups) + anonymous
}

// EvidenceReliability returns the mean bounded source reliability.
func EvidenceReliability(evidence []ExperienceEvidence) float64 {
	if len(evidence) == 0 {
		return 0
	}
	var total float64
	for _, e := range evidence {
		if e.Reliability < 0 {
			continue
		}
		if e.Reliability > 1 {
			total++
			continue
		}
		total += e.Reliability
	}
	return total / float64(len(evidence))
}

// ContradictionLoad measures how much evidence belongs to a contradiction
// set. It preserves the evidence instead of treating contradiction as
// deletion or decay.
func ContradictionLoad(evidence []ExperienceEvidence, contradictionSet string) float64 {
	if contradictionSet == "" || len(evidence) == 0 {
		return 0
	}
	var total float64
	for _, e := range evidence {
		if e.ContradictionSet != contradictionSet {
			continue
		}
		reliability := e.Reliability
		if reliability <= 0 {
			reliability = 0.5
		}
		total += reliability
	}
	return clamp01(total / float64(len(evidence)))
}
