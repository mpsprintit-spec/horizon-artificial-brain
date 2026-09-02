package hfcc

// Ablate returns a derived candidate snapshot and a report. Original is never mutated.
func Ablate(original Candidate, spec AblationSpec) (Candidate, AblationReport) {
	derived := CloneCandidate(original)
	derived.Version = original.Version + "+ablate"
	report := AblationReport{}

	beforePart := CollectParticipationPairs(original.Structure)
	beforeFP := StructuralFingerprint(original.Structure)

	if spec.RemoveProvenance {
		if len(derived.Provenance.Records) > 0 {
			report.Lost = append(report.Lost, "provenance")
		}
		derived.Provenance = Provenance{}
	} else if len(original.Provenance.Records) > 0 {
		report.Retained = append(report.Retained, "provenance")
	}

	if spec.RemoveParticipation {
		removeParticipation(&derived.Structure)
		after := CollectParticipationPairs(derived.Structure)
		if len(beforePart) > 0 && len(after) == 0 {
			report.Lost = append(report.Lost, "semantic_participation")
		}
		// Relocation: participation still encoded only if something else holds it — we do not auto-claim redundancy.
	} else if HasParticipation(original.Structure) {
		report.Retained = append(report.Retained, "semantic_participation")
	}

	if spec.RemoveDirection {
		if stripDirection(&derived.Structure) {
			report.Lost = append(report.Lost, "direction")
		}
	}

	if spec.RemoveFunction {
		if stripFunction(&derived.Structure) {
			report.Lost = append(report.Lost, "function")
		}
	}

	if spec.RemoveParticipantID {
		stripParticipantIDs(&derived.Structure)
		report.Lost = append(report.Lost, "participant_identity")
	}

	if spec.RemoveNesting {
		derived.Structure = flattenNesting(derived.Structure)
		report.Lost = append(report.Lost, "structural_nesting")
		report.AmbiguityIntroduced = true
	}

	afterFP := StructuralFingerprint(derived.Structure)
	if beforeFP != afterFP {
		report.Notes = append(report.Notes, "structure fingerprint changed after ablation")
	}

	// Detect relocation of participation: if removed from participants but still present as Kind labels encoding roles
	if spec.RemoveParticipation && len(beforePart) > 0 {
		// If pairs still recoverable from structure kinds that embed role names — treated as relocated channel
		still := false
		for _, pair := range beforePart {
			if structureMentions(derived.Structure, pair) {
				still = true
			}
		}
		if still {
			report.Relocated = append(report.Relocated, "semantic_participation")
			report.Notes = append(report.Notes, "participation information still present via alternate structural channel")
		}
	}

	return derived, report
}

func removeParticipation(n *StructureNode) {
	for i := range n.Participants {
		n.Participants[i].Participation = nil
	}
	for i := range n.Children {
		removeParticipation(&n.Children[i])
	}
}

func stripDirection(n *StructureNode) bool {
	lost := n.Direction != nil
	n.Direction = nil
	for i := range n.Children {
		if stripDirection(&n.Children[i]) {
			lost = true
		}
	}
	return lost
}

func stripFunction(n *StructureNode) bool {
	lost := n.Function != nil
	n.Function = nil
	for i := range n.Children {
		if stripFunction(&n.Children[i]) {
			lost = true
		}
	}
	return lost
}

func stripParticipantIDs(n *StructureNode) {
	for i := range n.Participants {
		n.Participants[i].ID = ""
		n.Participants[i].Ref = ""
	}
	for i := range n.Children {
		stripParticipantIDs(&n.Children[i])
	}
}

// flattenNesting loses operator nesting by collecting leaf kinds under a bag root.
func flattenNesting(n StructureNode) StructureNode {
	var leaves []StructureNode
	var walk func(StructureNode)
	walk = func(x StructureNode) {
		if len(x.Children) == 0 {
			y := CloneStructure(x)
			y.Children = nil
			leaves = append(leaves, y)
			return
		}
		for _, c := range x.Children {
			walk(c)
		}
	}
	walk(n)
	return StructureNode{Kind: "FLAT_BAG", Children: leaves}
}

func structureMentions(n StructureNode, pair string) bool {
	role := participationFromPair(pair)
	ref := refFromPair(pair)
	if role != "" && containsKind(n, role) {
		return true
	}
	if ref != "" && containsKind(n, ref) {
		return true
	}
	if pair != "" && containsKind(n, pair) {
		return true
	}
	return false
}

func participationFromPair(pair string) string {
	const arrow = "→"
	for i := 0; i+len(arrow) <= len(pair); i++ {
		if pair[i:i+len(arrow)] == arrow {
			return pair[i+len(arrow):]
		}
	}
	for i := 0; i < len(pair); i++ {
		if pair[i] == '>' && i > 0 && pair[i-1] == '-' {
			return pair[i+1:]
		}
	}
	return ""
}

func refFromPair(pair string) string {
	const arrow = "→"
	for i := 0; i+len(arrow) <= len(pair); i++ {
		if pair[i:i+len(arrow)] == arrow {
			return pair[:i]
		}
	}
	return ""
}

func containsKind(n StructureNode, sub string) bool {
	if sub != "" && len(n.Kind) >= len(sub) {
		for i := 0; i+len(sub) <= len(n.Kind); i++ {
			if n.Kind[i:i+len(sub)] == sub {
				return true
			}
		}
	}
	for _, c := range n.Children {
		if containsKind(c, sub) {
			return true
		}
	}
	return false
}
