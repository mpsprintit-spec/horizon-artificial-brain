package hfcc

import "fmt"

// CloneStructure deep-copies a structure tree (snapshots / non-destructive ops).
func CloneStructure(n StructureNode) StructureNode {
	out := StructureNode{
		Kind: n.Kind,
	}
	if n.Direction != nil {
		d := *n.Direction
		out.Direction = &d
	}
	if n.Function != nil {
		f := *n.Function
		out.Function = &f
	}
	if len(n.Participants) > 0 {
		out.Participants = make([]Participant, len(n.Participants))
		for i, p := range n.Participants {
			out.Participants[i] = p
			if p.Participation != nil {
				s := *p.Participation
				out.Participants[i].Participation = &s
			}
		}
	}
	if len(n.Children) > 0 {
		out.Children = make([]StructureNode, len(n.Children))
		for i, c := range n.Children {
			out.Children[i] = CloneStructure(c)
		}
	}
	return out
}

// CloneCandidate returns an independent snapshot.
func CloneCandidate(c Candidate) Candidate {
	out := c
	out.Structure = CloneStructure(c.Structure)
	if c.Provenance.Records != nil {
		out.Provenance.Records = append([]ProvenanceRecord{}, c.Provenance.Records...)
		for i := range out.Provenance.Records {
			out.Provenance.Records[i].EvidenceRefs = append([]string{}, c.Provenance.Records[i].EvidenceRefs...)
			out.Provenance.Records[i].DependentOn = append([]string{}, c.Provenance.Records[i].DependentOn...)
		}
	}
	if c.Gold != nil {
		g := *c.Gold
		g.EvidenceSet = append([]string{}, c.Gold.EvidenceSet...)
		out.Gold = &g
	}
	return out
}

// StrPtr helper for optional string fields in fixtures.
func StrPtr(s string) *string { return &s }

// Node builds a leaf-ish structure node.
func Node(kind string, parts ...Participant) StructureNode {
	return StructureNode{Kind: kind, Participants: parts}
}

// Nest builds parent(kind, children...).
func Nest(kind string, children ...StructureNode) StructureNode {
	return StructureNode{Kind: kind, Children: children}
}

// WithFunction sets optional communicative function on a copy.
func WithFunction(n StructureNode, fn string) StructureNode {
	n = CloneStructure(n)
	n.Function = StrPtr(fn)
	return n
}

// WithDirection sets optional semantic direction on a copy.
func WithDirection(n StructureNode, dir string) StructureNode {
	n = CloneStructure(n)
	n.Direction = StrPtr(dir)
	return n
}

// StructuralFingerprint is a deterministic tree encoding for comparison (not human display).
func StructuralFingerprint(n StructureNode) string {
	return fp(n)
}

func fp(n StructureNode) string {
	s := n.Kind
	if n.Direction != nil {
		s += "@dir:" + *n.Direction
	}
	if n.Function != nil {
		s += "@fn:" + *n.Function
	}
	for _, p := range n.Participants {
		s += "|p:" + p.ID + ":" + p.Ref
		if p.Participation != nil {
			s += ":" + *p.Participation
		}
	}
	for _, c := range n.Children {
		s += "[" + fp(c) + "]"
	}
	return s
}

// HasParticipation reports whether any participant carries a participation label.
func HasParticipation(n StructureNode) bool {
	for _, p := range n.Participants {
		if p.Participation != nil && *p.Participation != "" {
			return true
		}
	}
	for _, c := range n.Children {
		if HasParticipation(c) {
			return true
		}
	}
	return false
}

// CollectParticipationPairs returns (ref, participation) pairs for relocation checks.
func CollectParticipationPairs(n StructureNode) []string {
	var out []string
	var walk func(StructureNode)
	walk = func(x StructureNode) {
		for _, p := range x.Participants {
			if p.Participation != nil && *p.Participation != "" {
				out = append(out, fmt.Sprintf("%s→%s", p.Ref, *p.Participation))
			}
		}
		for _, c := range x.Children {
			walk(c)
		}
	}
	walk(n)
	return out
}
