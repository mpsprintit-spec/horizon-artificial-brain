package knowledge

import (
	"testing"
	"time"
)

func TestRepresentationDuplicateIgnoresNodeIdentity(t *testing.T) {
	a := newRepresentationNode(1, []float64{0.2, 0.4, 0.8})
	b := newRepresentationNode(99, []float64{0.2, 0.4, 0.8})
	if !RepresentationDuplicate(a, b, 0) {
		t.Fatal("identical representations with different node IDs must be detected as duplicates")
	}
}

func TestSharedRepresentationCountReusesUnits(t *testing.T) {
	p1 := ProjectionPopulation{Units: []PopulationUnit{{NodeID: 1}, {NodeID: 2}, {NodeID: 3}}}
	p2 := ProjectionPopulation{Units: []PopulationUnit{{NodeID: 2}, {NodeID: 3}, {NodeID: 4}}}
	if got := SharedRepresentationCount([]ProjectionPopulation{p1, p2}); got != 4 {
		t.Fatalf("shared units counted more than once: got %d want 4", got)
	}
}

func TestPatternDuplicateIgnoresLearningState(t *testing.T) {
	a := &PatternSynapse{
		Members: []NodeID{1, 2},
		Sequence: []PatternStep{{NodeID: 1, Position: 0, Delta: 0}, {NodeID: 2, Position: 1, Delta: time.Second}},
		Weight: 0.2, Confidence: 0.1, Frequency: 1,
	}
	b := &PatternSynapse{
		Members: []NodeID{1, 2},
		Sequence: []PatternStep{{NodeID: 1, Position: 0, Delta: 0}, {NodeID: 2, Position: 1, Delta: time.Second}},
		Weight: 0.9, Confidence: 0.8, Frequency: 20,
	}
	if !PatternDuplicate(a, b, 0) {
		t.Fatal("same structural pattern must not be duplicated because learning state differs")
	}
}

func TestPatternDuplicateDetectsDifferentStructure(t *testing.T) {
	a := &PatternSynapse{Members: []NodeID{1, 2}, Sequence: []PatternStep{{NodeID: 1, Position: 0}, {NodeID: 2, Position: 1}}}
	b := &PatternSynapse{Members: []NodeID{1, 3}, Sequence: []PatternStep{{NodeID: 1, Position: 0}, {NodeID: 3, Position: 1}}}
	if PatternDuplicate(a, b, 0) {
		t.Fatal("different structural patterns must remain distinct")
	}
}
