package knowledge

import "testing"

func TestNMetricExperimentalCapacityAdmission(t *testing.T) {
	if nMetricFeatureLimit(nil) != 32 {
		t.Fatal("production default changed")
	}
	for _, limit := range []int{0, 32, 64} {
		p := &NMetricProposalConfig{Protocol: NMetricBoundedV1, Strategy: "combined", Fractions: []float64{.25}, FeatureLimit: limit}
		if err := validateNMetricProposal(p); err != nil {
			t.Fatal(limit, err)
		}
		want := limit
		if want == 0 {
			want = 32
		}
		if nMetricFeatureLimit(p) != want {
			t.Fatal("incorrect search ceiling")
		}
	}
	for _, limit := range []int{-1, 1, 31, 33, 63, 65, 1024} {
		p := &NMetricProposalConfig{Protocol: NMetricBoundedV1, Strategy: "combined", Fractions: []float64{.25}, FeatureLimit: limit}
		if validateNMetricProposal(p) == nil {
			t.Fatal("unbounded or undocumented research capacity", limit)
		}
	}
}
