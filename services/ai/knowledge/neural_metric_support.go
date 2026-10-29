package knowledge

import (
	"math"
	"slices"
)

// Raw values are recovered from the same canonical feature graph, including
// both kinds of product; precision refinement is not limited to projections.
func nMetricFeatureRaw(m nModel, index int, rows []NWitness) ([]float64, float64) {
	c := m.Circuits[index]
	values := make([]float64, len(rows))
	scale := 0.
	parent := func(id NodeID) int {
		for i, f := range m.Circuits {
			if f.Meta.Node == id {
				return i
			}
		}
		return -1
	}
	for i, r := range rows {
		if c.Meta.Kind == 1 {
			values[i] = nDot(c.Axis, r.X)
		} else {
			e := nMetricEvaluate(m, r.X, nil)
			v := e.Activation
			if c.Meta.Kind == 3 {
				v = e.Metric
			}
			values[i] = v[parent(c.Meta.Parents[0])] * v[parent(c.Meta.Parents[1])]
		}
		scale = math.Max(scale, math.Abs(values[i]))
	}
	return values, math.Max(math.SmallestNonzeroFloat64, nRoundoff(scale))
}

// Witness selection covers the active feature dependencies of each query.
// It uses finite experience diversity for retention, never to assert identity
// or create a neuron. All selected labels remain ordinary learning evidence.
func nMetricWitnesses(rows []NWitness, heads []NHeadTopology, budget int, m nModel) []NWitness {
	if len(heads) == 0 {
		return nil
	}
	out := []NWitness{}
	quota := budget / len(heads)
	for _, h := range heads {
		group := []NWitness{}
		for _, r := range rows {
			if slices.Equal(r.Query, h.Query) && !slices.ContainsFunc(group, func(w NWitness) bool { return w.Outcome == r.Outcome && slices.Equal(w.X, r.X) }) {
				group = append(group, r)
			}
		}
		if len(group) <= quota {
			out = append(out, group...)
			continue
		}
		hi := nHeadIndex(m, h.Query)
		relevant := map[NodeID]bool{}
		var include func(NodeID)
		include = func(id NodeID) {
			if relevant[id] {
				return
			}
			relevant[id] = true
			for _, c := range m.Circuits {
				if c.Meta.Node == id {
					for _, p := range c.Meta.Parents {
						include(p)
					}
				}
			}
		}
		if hi >= 0 {
			for i, w := range m.Heads[hi].W {
				if w != 0 {
					include(m.Circuits[i].Meta.Node)
				}
			}
		}
		vectors := make([][]float64, len(group))
		scales := make([]float64, len(m.Circuits)+1)
		for i, r := range group {
			e := nMetricEvaluate(m, r.X, nil)
			v := make([]float64, len(scales))
			v[0] = r.Outcome
			for j, c := range m.Circuits {
				if relevant[c.Meta.Node] {
					v[j+1] = e.Metric[j]
				}
			}
			vectors[i] = v
			for j, x := range v {
				scales[j] = math.Max(scales[j], math.Abs(x))
			}
		}
		selected := make([]bool, len(group))
		distance := make([]float64, len(group))
		for i := range distance {
			distance[i] = math.Inf(1)
		}
		pick := 0
		for i := range group {
			if group[i].Outcome < group[pick].Outcome {
				pick = i
			}
		}
		for count := 0; count < quota; count++ {
			selected[pick] = true
			out = append(out, group[pick])
			next := -1
			for i := range group {
				if selected[i] {
					continue
				}
				d := 0.
				for j, x := range vectors[i] {
					if scales[j] > 0 {
						v := (x - vectors[pick][j]) / scales[j]
						d += v * v
					}
				}
				distance[i] = math.Min(distance[i], d)
				if next < 0 || distance[i] > distance[next] {
					next = i
				}
			}
			if next < 0 {
				break
			}
			pick = next
		}
	}
	return out
}
