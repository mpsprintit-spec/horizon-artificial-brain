package knowledge

import "time"

// EvidenceState keeps learning signals distinct. It is diagnostic neural state,
// not a semantic rule or a replacement for the distributed substrate.
type EvidenceState struct {
	ActivationStrength  float64   `json:"activation_strength"`
	EvidenceConfidence  float64   `json:"evidence_confidence"`
	StructuralStability float64   `json:"structural_stability"`
	SourceReliability   float64   `json:"source_reliability"`
	Recency             float64   `json:"recency"`
	ContradictionLoad   float64   `json:"contradiction_load"`
	Calibration         float64   `json:"calibration"`
	UpdatedAt           time.Time `json:"updated_at,omitempty"`
}

// evidenceGroup identifies the conservative unit of independent support.
// Explicit independence groups take precedence over source identity.
func evidenceGroup(e ExperienceEvidence) string {
	if e.IndependenceGroup != "" {
		return "group:" + e.IndependenceGroup
	}
	if e.Source != "" {
		return "source:" + e.Source
	}
	if e.ExperienceID != "" {
		return "experience:" + e.ExperienceID
	}
	if !e.Timestamp.IsZero() {
		return "time:" + e.Timestamp.UTC().Format(time.RFC3339Nano)
	}
	return "anonymous"
}

func EvidenceIndependenceBounded(evidence []ExperienceEvidence) int {
	groups := make(map[string]struct{}, len(evidence))
	for _, e := range evidence {
		groups[evidenceGroup(e)] = struct{}{}
	}
	return len(groups)
}

// BuildEvidenceState separates activation, evidence, structural stability,
// reliability, recency, contradiction and calibration into independent
// observable signals. It does not create semantic relations or replace the
// neural substrate.
func BuildEvidenceState(activationStrength, structuralStability float64, evidence []ExperienceEvidence, now, reference time.Time) EvidenceState {
	now = now.UTC()
	reference = reference.UTC()
	recency := 0.0
	if !reference.IsZero() {
		age := now.Sub(reference)
		if age < 0 {
			age = 0
		}
		recency = 1 / (1 + age.Hours()/24)
	}

	contradiction := 0.0
	for _, e := range evidence {
		if e.ContradictionSet == "" {
			continue
		}
		load := ContradictionLoad(evidence, e.ContradictionSet)
		if load > contradiction {
			contradiction = load
		}
	}

	return EvidenceState{
		ActivationStrength:  clamp01(activationStrength),
		EvidenceConfidence:  CalibratedConfidence(0.5, evidence),
		StructuralStability: clamp01(structuralStability),
		SourceReliability:   EvidenceReliability(evidence),
		Recency:             recency,
		ContradictionLoad:   contradiction,
		Calibration:         clamp01(1 - contradiction),
		UpdatedAt:           now,
	}
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
