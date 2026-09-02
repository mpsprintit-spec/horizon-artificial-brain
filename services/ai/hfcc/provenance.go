package hfcc

// HasExternalKnowledge reports whether any provenance record is external.
func HasExternalKnowledge(p Provenance) bool {
	for _, r := range p.Records {
		if r.InferenceClass == InferenceExternalKnowledge {
			return true
		}
	}
	return false
}

// HasCircularDependency reports a simple cycle: claimID depends on itself through DependentOn.
func HasCircularDependency(records []ProvenanceRecord, claimID string) bool {
	seen := map[string]bool{}
	var walk func(string) bool
	walk = func(id string) bool {
		if id == claimID && seen[id] {
			return true
		}
		if seen[id] {
			return false
		}
		seen[id] = true
		for _, r := range records {
			for _, dep := range r.DependentOn {
				if dep == id || (id == claimID && containsStr(r.DependentOn, claimID)) {
					for _, d := range r.DependentOn {
						if d == claimID {
							return true
						}
						if walk(d) {
							return true
						}
					}
				}
			}
		}
		return false
	}
	// Direct self-dependence
	for _, r := range records {
		for _, d := range r.DependentOn {
			if d == claimID && r.SourceRef == claimID {
				return true
			}
			if d == claimID {
				// dependent on the claim being justified
				return true
			}
		}
	}
	return false
}

func containsStr(xs []string, v string) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

// ValidateProvenanceForClaim checks circularity and external tagging.
func ValidateProvenanceForClaim(p Provenance, claimID string) (ok bool, reason string) {
	if HasCircularDependency(p.Records, claimID) {
		return false, "circular_evidence"
	}
	return true, ""
}
