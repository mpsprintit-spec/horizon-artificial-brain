package hfcc

// CompareStructures performs structural comparison only (not string equality of pretty-print).
// Provenance/history comparison is a separate concern and not mixed into Structural result.
func CompareStructures(a, b StructureNode) ComparisonResult {
	fa, fb := StructuralFingerprint(a), StructuralFingerprint(b)
	if fa == fb {
		return ComparisonResult{Structural: StructIdentical}
	}
	// Equivalent: same shape ignoring optional participation labels
	if fpIgnoreParticipation(a) == fpIgnoreParticipation(b) {
		return ComparisonResult{Structural: StructEquivalent, Notes: []string{"differs only in optional participation annotations"}}
	}
	if sharesKindRoot(a, b) {
		return ComparisonResult{Structural: StructPartial, Notes: []string{"shared root kind with different nesting or participants"}}
	}
	return ComparisonResult{Structural: StructDifferent}
}

func sharesKindRoot(a, b StructureNode) bool {
	return a.Kind != "" && a.Kind == b.Kind
}

func fpIgnoreParticipation(n StructureNode) string {
	x := CloneStructure(n)
	stripPart(&x)
	return StructuralFingerprint(x)
}

func stripPart(n *StructureNode) {
	for i := range n.Participants {
		n.Participants[i].Participation = nil
	}
	for i := range n.Children {
		stripPart(&n.Children[i])
	}
}
