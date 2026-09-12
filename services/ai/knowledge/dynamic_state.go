package knowledge

import "time"

// DynamicState is the primitive state carried by a connection in Horizon's
// distributed neural substrate. It intentionally contains no semantic
// ontology such as cause, has, is-a, or can-do.
type DynamicState struct {
	Weight           float64   `json:"weight"`
	Activation       float64   `json:"activation"`
	Eligibility      float64   `json:"eligibility"`
	Confidence       float64   `json:"confidence"`
	Frequency        int64     `json:"frequency"`
	LastActivation   time.Time `json:"last_activation,omitempty"`
	LastModification time.Time `json:"last_modification,omitempty"`
}

// Reinforce updates adaptive state from an experience without assigning a
// programmer-defined meaning to the connection.
func (s *DynamicState) Reinforce(weight, confidence, eligibility float64, now time.Time) {
	if s == nil {
		return
	}
	if s.Frequency <= 0 {
		s.Weight = clamp01(weight)
		s.Confidence = clamp01(confidence)
	} else {
		n := float64(s.Frequency)
		s.Weight = clamp01((s.Weight*n + weight) / (n + 1))
		s.Confidence = clamp01(1 - (1-s.Confidence)*(1-confidence))
	}
	s.Eligibility = clamp01(eligibility)
	s.Frequency++
	s.LastActivation = now
	s.LastModification = now
}
