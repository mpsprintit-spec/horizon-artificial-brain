package knowledge

import "slices"

func nMetricRefinableLeaf(m nModel, index int, query []NodeID) bool {
	c := m.Circuits[index]
	if c.Meta.Kind != 1 {
		return false
	}
	for _, child := range m.Circuits {
		if slices.Contains(child.Meta.Parents, c.Meta.Node) {
			return false
		}
	}
	for _, h := range m.Heads {
		if h.Known && !slices.Equal(h.Query, query) && h.W[index] != 0 {
			return false
		}
	}
	return true
}
