package knowledge

import "strings"

// EvidenceCounts separates reported measurements and provenance. A source
// label is not a declaration of independence; request IDs and episode IDs
// never establish independence. LegacyObservations have no typed provenance.
type EvidenceCounts struct {
	Observations       int `json:"observations"`
	Episodes           int `json:"episodes"`
	Sources            int `json:"sources"`
	CausalGroups       int `json:"causal_groups"`
	IndependentGroups  int `json:"independent_groups"`
	LegacyObservations int `json:"legacy_observations,omitempty"`
}

// CountEvidenceProvenance counts connected provenance components containing
// an explicit independence declaration. Shared sources, declared groups or
// causes join components transitively, including later bridge observations.
// Unknown provenance contributes frequency but no independent confirmation.
// This is adapter metadata accounting, not a semantic inference mechanism.
func CountEvidenceProvenance(evidence []ExperienceEvidence) EvidenceCounts {
	type label struct {
		kind  int
		value string
	}
	parents := make(map[label]label)
	var root func(label) label
	root = func(k label) label {
		p, ok := parents[k]
		if !ok {
			parents[k] = k
			return k
		}
		if p != k {
			parents[k] = root(p)
		}
		return parents[k]
	}
	join := func(a, b label) {
		a, b = root(a), root(b)
		if a != b {
			parents[b] = a
		}
	}
	episodes, sources, causes := map[string]bool{}, map[string]bool{}, map[string]bool{}
	declared := make([]label, 0, len(evidence))
	for _, e := range evidence {
		source, group, cause, episode := strings.TrimSpace(e.Source), strings.TrimSpace(e.IndependenceGroup), strings.TrimSpace(e.CausalGroup), strings.TrimSpace(e.EpisodeID)
		if episode != "" {
			episodes[episode] = true
		}
		if source != "" {
			sources[source] = true
		}
		if cause != "" {
			causes[cause] = true
		}
		// Unattributed records cannot establish declared independent support.
		if source == "" {
			continue
		}
		s := label{0, source}
		root(s)
		if group != "" {
			g := label{1, group}
			join(s, g)
			declared = append(declared, g)
		}
		if cause != "" {
			join(s, label{2, cause})
		}
	}
	components := make(map[label]bool)
	for _, g := range declared {
		components[root(g)] = true
	}
	return EvidenceCounts{Observations: len(evidence), Episodes: len(episodes), Sources: len(sources), CausalGroups: len(causes), IndependentGroups: len(components)}
}
