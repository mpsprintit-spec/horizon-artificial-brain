package knowledge

// ReusePopulation returns an existing population when it is structurally
// equivalent to the supplied population. A population is a distributed
// activation pattern over the single brain substrate; it is not a separate
// memory object. Repeated experiences therefore reuse the same unit IDs.
func ReusePopulation(existing []ProjectionPopulation, candidate ProjectionPopulation) (ProjectionPopulation, bool) {
	for _, population := range existing {
		if PopulationEquivalent(population, candidate) {
			return population, true
		}
	}
	return candidate, false
}

// PopulationEquivalent compares substrate membership, not activation state or
// ranking order. Population identity is the set of participating substrate
// units; activation is transient and ranking may change from one recall to the
// next.
func PopulationEquivalent(a, b ProjectionPopulation) bool {
	if len(a.Units) != len(b.Units) {
		return false
	}

	membership := make(map[NodeID]struct{}, len(a.Units))
	for _, unit := range a.Units {
		if _, duplicate := membership[unit.NodeID]; duplicate {
			return false
		}
		membership[unit.NodeID] = struct{}{}
	}

	for _, unit := range b.Units {
		if _, ok := membership[unit.NodeID]; !ok {
			return false
		}
	}
	return true
}
