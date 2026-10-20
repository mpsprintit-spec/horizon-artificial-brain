package knowledge

import (
	"math"
	"testing"
)

func TestEvidenceProvenanceCalibrationRetainsHistoricalLaw(t *testing.T) {
	e := []ExperienceEvidence{{ExperienceID: "r1", Source: "camera", IndependenceGroup: "g1", CausalGroup: "scene", EpisodeID: "e1", Reliability: .9}, {ExperienceID: "r2", Source: "camera", IndependenceGroup: "g2", CausalGroup: "scene", EpisodeID: "e2", Reliability: .9}}
	one := CalibratedConfidenceWithDynamics(.5, e[:1], CurrentDynamics)
	if got := CalibratedConfidenceWithDynamics(.5, e, CurrentDynamics); got != one {
		t.Fatalf("same source inflated calibrated support: %g -> %g", one, got)
	}
	if old := CalibratedConfidenceWithDynamics(.5, e, StableProjectionDynamics); old <= one || old != CalibratedConfidence(.5, e) {
		t.Fatal("historical calibration changed")
	}
	e[1].Source = "relay"
	if CalibratedConfidenceWithDynamics(.5, e, CurrentDynamics) != one {
		t.Fatal("shared cause inflated independent support")
	}
	e[1].CausalGroup = "other-cause"
	if CalibratedConfidenceWithDynamics(.5, e, CurrentDynamics) <= one {
		t.Fatal("independent evidence provided no additional support")
	}
	for i := range e {
		e[i].IndependenceGroup = ""
	}
	if got := CalibratedConfidenceWithDynamics(.5, e, CurrentDynamics); got != .5 {
		t.Fatal("source identity alone established independence")
	}
	e[0].ContradictionSet = "conflict"
	if got := CalibratedConfidenceWithDynamics(.5, e, CurrentDynamics); got >= .5 || math.IsNaN(got) {
		t.Fatal("unknown independence concealed contradiction")
	}
}
