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

// EvidenceIndependenceBounded counts effective evidence groups. Explicit
// independence groups take precedence; source is the conservative fallback.
// This prevents repeated observations from one source from masquerading as
// independent confirmation.
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
	sets := make(map[string]struct{})
	for _, e := range evidence {
		if e.ContradictionSet != "" {
			sets[e.ContradictionSet] = struct{}{}
		}
	}
	for set := range sets {
		contradiction = maxFloat(contradiction, ContradictionLoad(evidence, set))
	}

	return EvidenceState{
		ActivationStrength:  clamp01(activationStrength),
		EvidenceConfidence: calibratedConfidenceState(evidence),
		StructuralStability: clamp01(structuralStability),
		SourceReliability:   EvidenceReliability(evidence),
		Recency:             recency,
		ContradictionLoad:   contradiction,
		Calibration:         clamp01(1 - contradiction),
		UpdatedAt:           now,
	}
}

// calibratedConfidenceState provides the evidence-aware confidence used by
// EvidenceState without introducing a second CalibratedConfidence definition.
func calibratedConfidenceState(evidence []ExperienceEvidence) float64 {
	if len(evidence) == 0 {
		return 0
	}
	independent := EvidenceIndependenceBounded(evidence)
	reliability := EvidenceReliability(evidence)
	contradiction := 0.0
	sets := make(map[string]struct{})
	for _, e := range evidence {
		if e.ContradictionSet != "" {
			sets[e.ContradictionSet] = struct{}{}
		}
	}
	for set := range sets {
		contradiction = maxFloat(contradiction, ContradictionLoad(evidence, set))
	}
	base := 0.5
	gain := (1 - base) * (1 - expDecay(float64(independent)*reliability))
	return clamp01((base + gain) * (1 - 0.5*contradiction))
}

func expDecay(x float64) float64 {
	if x <= 0 {
		return 1
	}
	return 1 / (1 + x)
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
