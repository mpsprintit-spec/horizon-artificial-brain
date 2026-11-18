package knowledge

import (
	"reflect"
	"testing"
)

func TestAdaptiveRepresentationResourceAndInterventionContract(t *testing.T) {
	p := &NMetricProposalConfig{Protocol: NMetricAdaptiveV1, Strategy: "combined", Fractions: []float64{.25, .5}, PreserveShared: true, Ridge: .001}
	if err := validateNMetricProposal(p); err != nil {
		t.Fatal(err)
	}
	if nMetricFeatureLimit(p) <= 32 || p.FixedBasis {
		t.Fatal("adaptive default retains historical fixed basis")
	}
	for _, limit := range []int{33, 97, 255} {
		p.FeatureLimit = limit
		if err := validateNMetricProposal(p); err != nil {
			t.Fatal(limit, err)
		}
	}
	p.FeatureLimit = 257
	if validateNMetricProposal(p) == nil {
		t.Fatal("physical bound missing")
	}
	p.FeatureLimit = 97
	p.PreserveShared = false
	if validateNMetricProposal(p) == nil {
		t.Fatal("unprotected shared growth accepted")
	}
	p.PreserveShared = true
	p.ExcludedFeatures = []NodeID{2, 2}
	if validateNMetricProposal(p) == nil {
		t.Fatal("duplicate causal lesion accepted")
	}
	p.ExcludedFeatures = []NodeID{2}
	m := nModel{Circuits: []nCircuit{{Meta: NFeature{Node: 1, Kind: 1}}, {Meta: NFeature{Node: 2, Kind: 1}}, {Meta: NFeature{Node: 3, Kind: 3, Parents: []NodeID{1, 2}}}}}
	ex := nMetricExcluded(m, p)
	if ex[1] || !ex[2] || !ex[3] {
		t.Fatal("selective ancestry intervention incorrect", ex)
	}
}

func TestAdaptiveSearchMemoizationPreservesValuesAndIsInvalidatedByClone(t *testing.T) {
	axis := make([]float64, nDimensions)
	axis[0] = 1
	m := nModel{MetricEnabled: true, Circuits: []nCircuit{{Meta: NFeature{Node: 1, Kind: 1}, Axis: axis, Delta: .1}}}
	x := make([]float64, nDimensions)
	x[0] = .27
	unc := make([]float64, nDimensions)
	unc[0] = .03
	want := nMetricEvaluate(m, x, unc)
	m.evaluationCache = make(map[nMetricEvaluationKey]nMetricEvaluation)
	if !reflect.DeepEqual(want, nMetricEvaluate(m, x, unc)) || !reflect.DeepEqual(want, nMetricEvaluate(m, x, unc)) {
		t.Fatal("memoization changed values")
	}
	clone := nClone(m)
	clone.Circuits[0].Delta = .3
	if clone.evaluationCache != nil || reflect.DeepEqual(want, nMetricEvaluate(clone, x, unc)) {
		t.Fatal("circuit clone retained stale memoization")
	}
}
