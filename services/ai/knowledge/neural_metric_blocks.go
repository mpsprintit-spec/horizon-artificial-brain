package knowledge

import (
	"math"
	"slices"
	"sort"
)

// Coordinate blocks express input structure, never task names or desired
// answers. Directions and ranking use only Fit residuals; Check admits them.
func nMetricBlockAxes(m nModel, query []NodeID, fit []NWitness, blocks [][]int) [][]float64 {
	if len(blocks) == 0 {
		for offset := 0; offset < nDimensions; offset += 7 {
			blocks = append(blocks, []int{offset, offset + 1, offset + 2}, []int{offset + 3, offset + 4, offset + 5})
		}
	}
	type scored struct {
		axis  []float64
		score float64
	}
	ranked := []scored{}
	mean := make([]float64, nDimensions)
	for _, r := range fit {
		for j, v := range r.X {
			mean[j] += v / float64(len(fit))
		}
	}
	for _, block := range blocks {
		axis := make([]float64, nDimensions)
		for _, j := range block {
			cov, variance := 0., 0.
			for _, r := range fit {
				d := r.X[j] - mean[j]
				cov += d * (r.Outcome - nPredict(m, r.X, query))
				variance += d * d
			}
			if variance > nRoundoff(variance) {
				axis[j] = cov / math.Sqrt(variance)
			}
		}
		norm := math.Sqrt(nDot(axis, axis))
		if !nFinite(norm) || norm <= math.SmallestNonzeroFloat64 {
			continue
		}
		for j := range axis {
			axis[j] /= norm
		}
		covariance, variance := 0., 0.
		center := nCenter(axis, fit)
		for _, r := range fit {
			v := nDot(axis, r.X) - center
			covariance += v * (r.Outcome - nPredict(m, r.X, query))
			variance += v * v
		}
		if variance > 0 {
			ranked = append(ranked, scored{axis, math.Abs(covariance) / math.Sqrt(variance)})
		}
	}
	sort.SliceStable(ranked, func(i, j int) bool { return ranked[i].score > ranked[j].score })
	out := [][]float64{}
	for _, v := range ranked[:min(4, len(ranked))] {
		out = append(out, slices.Clone(v.axis))
	}
	return out
}
