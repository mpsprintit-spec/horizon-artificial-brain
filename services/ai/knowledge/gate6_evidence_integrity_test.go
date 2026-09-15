package knowledge

import (
	"testing"
	"time"
)

func TestGate6RepeatedSameSourceDoesNotInflateIndependence(t *testing.T) {
	evidence := []ExperienceEvidence{
		{ExperienceID: "a1", Source: "camera-1", Reliability: 0.9},
		{ExperienceID: "a2", Source: "camera-1", Reliability: 0.9},
		{ExperienceID: "a3", Source: "camera-1", Reliability: 0.9},
	}
	if got := EvidenceIndependenceBounded(evidence); got != 1 {
		t.Fatalf("same source counted as %d independent groups; want 1", got)
	}
}

func TestGate6IndependentSourcesIncreaseEvidenceWithoutUnboundedDuplicateGain(t *testing.T) {
	repeated := make([]ExperienceEvidence, 0, 20)
	for i := 0; i < 20; i++ {
		repeated = append(repeated, ExperienceEvidence{ExperienceID: string(rune('a' + i)), Source: "sensor-a", Reliability: 1})
	}
	one := CalibratedConfidence(0.5, repeated)
	independent := []ExperienceEvidence{
		{ExperienceID: "a", Source: "sensor-a", Reliability: 1},
		{ExperienceID: "b", Source: "sensor-b", Reliability: 1},
		{ExperienceID: "c", Source: "sensor-c", Reliability: 1},
	}
	many := CalibratedConfidence(0.5, independent)
	if one >= 0.9 {
		t.Fatalf("duplicate same-source evidence inflated confidence to %.4f", one)
	}
	if many <= one {
		t.Fatalf("independent evidence did not increase confidence: duplicate=%.4f independent=%.4f", one, many)
	}
}

func TestGate6ContradictionPreservesEvidenceAndReducesConfidence(t *testing.T) {
	evidence := []ExperienceEvidence{
		{ExperienceID: "support", Source: "source-a", Reliability: 0.9, ContradictionSet: "claim-1", Timestamp: time.Unix(1, 0)},
		{ExperienceID: "contradiction", Source: "source-b", Reliability: 0.9, ContradictionSet: "claim-1", Timestamp: time.Unix(2, 0)},
	}
	before := CalibratedConfidence(0.7, []ExperienceEvidence{{ExperienceID: "support", Source: "source-a", Reliability: 0.9}})
	after := CalibratedConfidence(0.7, evidence)
	if len(evidence) != 2 {
		t.Fatal("contradictory evidence was deleted")
	}
	if after >= before {
		t.Fatalf("contradiction did not reduce calibrated confidence: before=%.4f after=%.4f", before, after)
	}
	if load := ContradictionLoad(evidence, "claim-1"); load <= 0 {
		t.Fatal("contradiction load was not retained")
	}
}

func TestGate6EvidenceStateSeparatesSignals(t *testing.T) {
	evidence := []ExperienceEvidence{{ExperienceID: "x", Source: "sensor-a", Reliability: 0.8}}
	state := BuildEvidenceState(0.6, 0.4, evidence, time.Unix(100, 0), time.Unix(90, 0))
	if state.ActivationStrength != 0.6 || state.StructuralStability != 0.4 {
		t.Fatal("activation and structural signals were not kept separate")
	}
	if state.SourceReliability <= 0 || state.EvidenceConfidence <= 0 {
		t.Fatal("evidence signals were not computed")
	}
}
