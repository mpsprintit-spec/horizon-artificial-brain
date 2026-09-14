package knowledge

import "math"

// RepresentationDuplicate reports whether two numeric neural units are
// effectively the same learned representation. Node identity is deliberately
// ignored: identical IDs are not the criterion for duplicate knowledge.
func RepresentationDuplicate(a, b *ConceptNode, tolerance float64) bool {
	if a == nil || b == nil || len(a.Representation) == 0 || len(a.Representation) != len(b.Representation) {
		return false
	}
	if tolerance < 0 {
		tolerance = 0
	}
	if tolerance > 1 {
		tolerance = 1
	}
	return NewNeuralVector(a.Representation).Similarity(NewNeuralVector(b.Representation)) >= 1-tolerance
}

// SharedRepresentationCount counts unique substrate units in a population.
// Reusing a unit in multiple experiences does not increase this count.
func SharedRepresentationCount(populations []ProjectionPopulation) int {
	seen := make(map[NodeID]struct{})
	for _, population := range populations {
		for _, unit := range population.Units {
			seen[unit.NodeID] = struct{}{}
		}
	}
	return len(seen)
}

// PatternDuplicate compares structural population membership and temporal
// sequence. Weight, confidence and frequency are state of the same pattern,
// not grounds for creating another copy.
func PatternDuplicate(a, b *PatternSynapse, tolerance float64) bool {
	if a == nil || b == nil || len(a.Members) != len(b.Members) || len(a.Sequence) != len(b.Sequence) {
		return false
	}
	for i := range a.Members {
		if a.Members[i] != b.Members[i] {
			return false
		}
	}
	for i := range a.Sequence {
		if a.Sequence[i].NodeID != b.Sequence[i].NodeID || a.Sequence[i].Position != b.Sequence[i].Position {
			return false
		}
		if math.Abs(a.Sequence[i].Delta.Seconds()-b.Sequence[i].Delta.Seconds()) > tolerance {
			return false
		}
	}
	return true
}
