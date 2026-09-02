package hfcc

// Reconstruct attempts to recover a target structure from an ablated/derived candidate
// using only allowed evidence. Does not consult Horizon Knowledge/FUG.
//
// allowedExternal: if false, any need for outside knowledge → EXTERNAL_KNOWLEDGE_REQUIRED.
func Reconstruct(original, derived Candidate, target StructureNode, allowedExternal bool) ReconstructionResult {
	wantPart := CollectParticipationPairs(target)
	havePart := CollectParticipationPairs(derived.Structure)
	origPart := CollectParticipationPairs(original.Structure)

	// Relocation: original had participation; derived lost the field but alternate structural channel still carries role cues.
	if len(origPart) > 0 && len(havePart) == 0 {
		relocated := false
		for _, w := range origPart {
			if structureMentions(derived.Structure, w) {
				relocated = true
				break
			}
		}
		if relocated {
			return InformationRelocated
		}
	}

	// If derived already contains full structural fingerprint of target → reconstructable from remaining channels
	if StructuralFingerprint(derived.Structure) == StructuralFingerprint(target) {
		return Recontructable
	}

	// Participation-only target recovery
	if len(wantPart) > 0 {
		if subsetPairs(wantPart, havePart) {
			return Recontructable
		}
		relocated := true
		for _, w := range wantPart {
			if !structureMentions(derived.Structure, w) {
				relocated = false
				break
			}
		}
		if relocated && len(wantPart) > 0 {
			return InformationRelocated
		}
	}

	// Nested operator recovery: if nesting was flattened, cannot reconstruct order of operators
	if isFlatBag(derived.Structure) && hasNesting(original.Structure) {
		return NotReconstructable
	}

	// Partial structural overlap
	cmp := CompareStructures(derived.Structure, target)
	if cmp.Structural == StructPartial {
		return ReconIncomplete
	}
	if cmp.Structural == StructIdentical || cmp.Structural == StructEquivalent {
		return Recontructable
	}

	if !allowedExternal {
		return ExternalKnowledgeRequired
	}
	return ExternalKnowledgeRequired
}

func subsetPairs(want, have []string) bool {
	set := map[string]bool{}
	for _, h := range have {
		set[h] = true
	}
	for _, w := range want {
		if !set[w] {
			return false
		}
	}
	return true
}

func isFlatBag(n StructureNode) bool {
	return n.Kind == "FLAT_BAG"
}

func hasNesting(n StructureNode) bool {
	if len(n.Children) > 0 {
		for _, c := range n.Children {
			if len(c.Children) > 0 || c.Kind != "" {
				return true
			}
		}
		return len(n.Children) > 0
	}
	return false
}
