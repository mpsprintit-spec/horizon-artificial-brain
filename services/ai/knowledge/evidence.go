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
