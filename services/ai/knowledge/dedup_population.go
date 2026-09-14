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

// PopulationEquivalent compares substrate membership, not activation state.
// Activation is transient and may differ each time the same learned pattern is
// recalled. Unit identity is therefore the structural identity of a population.
func PopulationEquivalent(a, b ProjectionPopulation) bool {
	if len(a.Units) != len(b.Units) {
		return false
	}
	for i := range a.Units {
		if a.Units[i].NodeID != b.Units[i].NodeID {
			return false
		}
	}
	return true
}
