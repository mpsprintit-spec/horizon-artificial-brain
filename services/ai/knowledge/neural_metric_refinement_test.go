package knowledge

import "testing"

func TestNMetricLeafRefinementProtectsOtherHeadsAndParents(t *testing.T) {
	q, other := []NodeID{10}, []NodeID{20}
	m := nModel{Circuits: []nCircuit{{Meta: NFeature{Node: 1, Kind: 1}}, {Meta: NFeature{Node: 2, Kind: 1}}}, Heads: []nHead{{Query: q, W: []float64{1, 1}, Known: true}, {Query: other, W: []float64{1, 0}, Known: true}}}
	if nMetricRefinableLeaf(m, 0, q) || !nMetricRefinableLeaf(m, 1, q) {
		t.Fatal("shared versus private direction selection")
	}
	m.Circuits = append(m.Circuits, nCircuit{Meta: NFeature{Node: 3, Kind: 3, Parents: []NodeID{2, 2}}})
	if nMetricRefinableLeaf(m, 1, q) {
		t.Fatal("parent direction offered for replacement")
	}
	p := &NMetricProposalConfig{Protocol: NMetricRefineV1, Strategy: "combined", Fractions: []float64{.25}, PreserveShared: true, RefineLeafDirections: true}
	if validateNMetricProposal(p) != nil {
		t.Fatal("valid explicit refinement rejected")
	}
	p.Protocol = NMetricBoundedV1
	if validateNMetricProposal(p) == nil {
		t.Fatal("old protocol silently enabled refinement")
	}
	p.Protocol = NMetricRefineV1
	p.PreserveShared = false
	if validateNMetricProposal(p) == nil {
		t.Fatal("unprotected refinement accepted")
	}
}
