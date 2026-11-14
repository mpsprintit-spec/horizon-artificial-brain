package knowledge

import (
	"reflect"
	"slices"
	"testing"
)

func relationalProbe(encoder string) (*NativeControlState, map[NodeID]*ConceptNode) {
	s := &NativeControlState{Config: NativeControlConfig{Encoder: encoder, MaxActiveNodes: 10, MaxSynapseReads: 100}, Bias: 99}
	for i := 0; i < s.Config.relationalFeatureCount(); i++ {
		s.Features = append(s.Features, NodeID(100+i))
	}
	graph := map[NodeID]*ConceptNode{}
	for _, id := range []NodeID{1, 2, 3} {
		graph[id] = &ConceptNode{ID: id, Synapses: map[NodeID]SynapseList{}}
	}
	graph[1].Synapses[2] = SynapseList{&Synapse{TargetID: 2, Weight: .6, Confidence: 1}, &Synapse{TargetID: 2, Weight: .2, Confidence: .5, Inhibitory: true}}
	graph[1].Synapses[3] = SynapseList{&Synapse{TargetID: 3, Weight: .3, Confidence: 1, Inhibitory: true}}
	graph[2].Synapses[1] = SynapseList{&Synapse{TargetID: 1, Weight: .5, Confidence: 1}}
	return s, graph
}

func TestRelationalV2PreservesOldMomentsAndReadBudget(t *testing.T) {
	var features [2]map[NodeID]float64
	var reads [2]int
	for i, encoder := range []string{NativeRelationalEncoderV1, NativeRelationalEncoderV2} {
		s, graph := relationalProbe(encoder)
		var err error
		features[i], err = nativeRelationalFeatures(s, map[NodeID]float64{1: .8, 2: .4}, func(id NodeID) *ConceptNode { return graph[id] }, &reads[i])
		if err != nil {
			t.Fatal(err)
		}
	}
	for id := NodeID(99); id <= 103; id++ {
		if features[0][id] != features[1][id] {
			t.Fatalf("historical moment changed at %d", id)
		}
	}
	if reads[0] != reads[1] {
		t.Fatal("signed moments caused additional structural reads")
	}
	s, graph := relationalProbe(NativeRelationalEncoderV2)
	get := func(id NodeID) *ConceptNode { return graph[id] }
	a, err := nativeRelationalFeatures(s, map[NodeID]float64{1: .8, 2: .4}, get)
	if err != nil {
		t.Fatal(err)
	}
	slices.Reverse(graph[1].Synapses[2])
	b, err := nativeRelationalFeatures(s, map[NodeID]float64{2: .4, 1: .8}, get)
	if err != nil || !reflect.DeepEqual(a, b) {
		t.Fatal("channel or active-map permutation changed features", err)
	}
	s.Config.MaxSynapseReads = reads[1] - 1
	if _, err = nativeRelationalFeatures(s, map[NodeID]float64{1: .8, 2: .4}, get); err == nil {
		t.Fatal("read budget bypassed")
	}
}

func TestRelationalV2SeparatesSignWithoutClaimingTemporalOrRoleEncoding(t *testing.T) {
	s, graph := relationalProbe(NativeRelationalEncoderV2)
	get := func(id NodeID) *ConceptNode { return graph[id] }
	state := map[NodeID]float64{1: .8}
	a, err := nativeRelationalFeatures(s, state, get)
	if err != nil {
		t.Fatal(err)
	}
	graph[1].Synapses[2][0].Inhibitory = true
	b, err := nativeRelationalFeatures(s, state, get)
	if err != nil || reflect.DeepEqual(a, b) {
		t.Fatal("edge sign still lost", err)
	}
	for _, id := range s.Features[:4] {
		if a[id] != b[id] {
			t.Fatal("sign changed unsigned moment")
		}
	}
	// A remaining limitation is deliberately executable: eligibility history is
	// present in N but is not exposed by this minimal sign repair.
	graph[1].Synapses[2][0].Dynamic.Eligibility = .9
	c, err := nativeRelationalFeatures(s, state, get)
	if err != nil || !reflect.DeepEqual(b, c) {
		t.Fatal("re-audit documented temporal limitation", err)
	}
}
