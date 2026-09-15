package knowledge

import "time"

// EvidenceState keeps learning signals distinct. It is diagnostic neural state,
// not a semantic rule or a replacement for the distributed substrate.
type EvidenceState struct {
	ActivationStrength float64   `json:"activation_strength"`
	EvidenceConfidence float64   `json:"evidence_confidence"`
	StructuralStability float64  `json:"structural_stability"`
	SourceReliability float64    `json:"source_reliability"`
	Recency float64              `json:"recency"`
	ContradictionLoad float64    `json:"contradiction_load"`
	Calibration float64          `json:"calibration"`
	UpdatedAt time.Time           `json:"updated_at,omitempty"`
}

// EvidenceIndependence now treats an explicit independence group as primary,
// source as the conservative fallback, and experience ID as a final fallback.
// Repeated observations from one source therefore do not masquerade as
// independent evidence merely because they have different experience IDs.
func evidenceGroup(e ExperienceEvidence) string {
	if e.IndependenceGroup != "" { return "group:" + e.IndependenceGroup }
	if e.Source != "" { return "source:" + e.Source }
	if e.ExperienceID != "" { return "experience:" + e.ExperienceID }
	if !e.Timestamp.IsZero() { return "time:" + e.Timestamp.UTC().Format(time.RFC3339Nano) }
	return "anonymous"
}

func EvidenceIndependenceBounded(evidence []ExperienceEvidence) int {
	groups := make(map[string]struct{}, len(evidence))
	for _, e := range evidence { groups[evidenceGroup(e)] = struct{}{} }
	return len(groups)
}

// CalibratedConfidence combines bounded independent evidence with reliability
// and contradiction load. Duplicate evidence from the same source has one
// effective contribution, so repeated exposure cannot drive confidence toward
// one without additional independent evidence.
func CalibratedConfidence(base float64, evidence []ExperienceEvidence) float64 {
	base = clamp01(base)
	if len(evidence) == 0 { return base }
	independent := EvidenceIndependenceBounded(evidence)
	reliability := EvidenceReliability(evidence)
	contradiction := 0.0
	sets := make(map[string]struct{})
	for _, e := range evidence { if e.ContradictionSet != "" { sets[e.ContradictionSet] = struct{}{} } }
	for set := range sets { contradiction = maxFloat(contradiction, ContradictionLoad(evidence, set)) }
	// Saturating evidence gain: each independent group adds less than the last.
	gain := (1 - base) * (1 - expDecay(float64(independent)*reliability))
	confidence := base + gain
	confidence *= 1 - 0.5*contradiction
	return clamp01(confidence)
}

func expDecay(x float64) float64 {
	if x <= 0 { return 1 }
	// Rational decay avoids requiring a heavyweight math dependency while keeping
	// the evidence contribution bounded and diminishing.
	return 1 / (1 + x)
}

func maxFloat(a, b float64) float64 { if a > b { return a }; return b }
