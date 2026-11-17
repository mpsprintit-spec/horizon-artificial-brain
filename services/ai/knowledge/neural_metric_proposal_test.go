package knowledge

import (
	"math"
	"reflect"
	"testing"
)

func TestNMetricBoundedProposalPreservesIncumbent(t *testing.T) {
	q := []NodeID{10, 11, 12, 13}
	a := nModel{MetricEnabled: true, Heads: []nHead{{Query: q, W: []float64{.2}, B: .1, Known: true, Metric: true}}, Circuits: []nCircuit{{Meta: NFeature{Node: 1, Kind: 1}, Axis: []float64{1}, Delta: .01}}}
	b := nClone(a)
	b.Heads[0].W[0] = .8
	b.Heads[0].B = .5
	p := &NMetricProposalConfig{Protocol: NMetricBoundedV1, Strategy: "bounded-joint", Fractions: []float64{.25}}
	steps := nMetricProposals(a, b, q, nil, p, true)
	if len(steps) != 2 || math.Abs(steps[1].Model.Heads[0].W[0]-.35) > 1e-15 || steps[1].Model.Heads[0].B != .2 {
		t.Fatal(steps)
	}
	if a.Heads[0].W[0] != .2 || b.Heads[0].W[0] != .8 {
		t.Fatal("proposal mutated model")
	}
	b.Heads[0].Metric = false
	if len(nMetricProposals(a, b, q, nil, p, true)) != 1 {
		t.Fatal("mixed readout bases interpolated")
	}
	// A too-large refit can worsen a new observation while a bounded step helps.
	a.Heads[0].B = 0
	a.Heads[0].W[0] = 0
	b = nClone(a)
	b.Heads[0].B = 1
	rows := []NWitness{{X: []float64{0}, Query: q, Outcome: .2}}
	steps = nMetricProposals(a, b, q, nil, p, true)
	if nError(steps[1].Model, rows) >= nError(a, rows) || nError(steps[0].Model, rows) <= nError(a, rows) {
		t.Fatal("bounded gain control failed")
	}
}

func TestNMetricDecoupledFeatureAndIndependentEvidence(t *testing.T) {
	q := []NodeID{10, 11, 12, 13}
	a := nModel{MetricEnabled: true, Heads: []nHead{{Query: q, W: []float64{.2}, B: .1, Known: true, Metric: true}}, Circuits: []nCircuit{{Meta: NFeature{Node: 1, Kind: 1}, Axis: []float64{1, 0}, Delta: .001}}}
	b := nClone(a)
	b.Circuits = append(b.Circuits, nCircuit{Meta: NFeature{Node: 2, Kind: 1}, Axis: []float64{0, 1}, Delta: .001})
	b.Heads[0].W = append(b.Heads[0].W, 1)
	fit := []NWitness{{EpisodeID: "fit", X: []float64{.2, .4}, Query: q, Outcome: .3}}
	p := &NMetricProposalConfig{Protocol: NMetricBoundedV1, Strategy: "new-feature", Fractions: []float64{.25}}
	steps := nMetricProposals(a, b, q, fit, p, true)
	if len(steps) != 2 {
		t.Fatal(steps)
	}
	for _, s := range steps {
		if s.Model.Heads[0].B != a.Heads[0].B || s.Model.Heads[0].W[0] != a.Heads[0].W[0] {
			t.Fatal("old parameters changed in decoupled proposal")
		}
	}
	if nError(steps[1].Model, fit) > 1e-20 {
		t.Fatal("new coefficient did not fit residual")
	}
	check := []NWitness{{EpisodeID: "check", X: []float64{.5, .5}, Query: q, Outcome: .4}}
	witnesses := []NWitness{check[0], {EpisodeID: "old", X: []float64{.1, .1}, Query: q}, {EpisodeID: "retention-only", Role: "evaluation", X: []float64{.9, .9}, Query: q}}
	before := nMetricHash(witnesses)
	out, e, err := nMetricIndependentEvidence(fit, check, witnesses, q)
	if err != nil || len(out) != 2 || e.WitnessFit != 1 || e.WitnessExcluded != 2 || nMetricHash(witnesses) != before {
		t.Fatal(out, e, err)
	}
	alias := check[0]
	alias.EpisodeID = "renamed"
	if _, _, err = nMetricIndependentEvidence([]NWitness{alias}, check, nil, q); err == nil {
		t.Fatal("renaming an episode concealed identical measurements")
	}
	clone := NDiagnosis{Evidence: e}.Clone()
	clone.Evidence.Fit = 999
	if reflect.DeepEqual(clone.Evidence, e) {
		t.Fatal("evidence alias")
	}
}

func TestNMetricProposalConfigValidation(t *testing.T) {
	for _, p := range []*NMetricProposalConfig{{Protocol: "unknown"}, {Protocol: NMetricBoundedV1, Strategy: "combined"}, {Protocol: NMetricBoundedV1, Strategy: "combined", Fractions: []float64{.25, .1}}, {Protocol: NMetricBoundedV1, Strategy: "combined", Fractions: []float64{math.NaN()}}} {
		if validateNMetricProposal(p) == nil {
			t.Fatal(p)
		}
	}
}
