package knowledge

// A generic, disposable, Fit-only numerical workspace. It changes neither
// the input codec nor the native prediction law. Accepted coefficients go
// through the existing shared protection, Check, retention and signed writer.
import (
	"math"
)

func nMetricRegularizedProposal(basis nModel, query []NodeID, rows []NWitness, ridge float64) nMetricProposal {
	m := nClone(basis)
	hi := nHeadIndex(m, query)
	h := &m.Heads[hi]
	n := len(m.Circuits)
	mean, sd := make([]float64, n), make([]float64, n)
	x := make([][]float64, len(rows))
	targetMean := 0.
	for i, r := range rows {
		x[i] = nMetricValues(m, r.X, h.Metric)
		targetMean += r.Outcome / float64(len(rows))
		for j, v := range x[i] {
			mean[j] += v / float64(len(rows))
		}
	}
	for _, v := range x {
		for j, a := range v {
			sd[j] += (a - mean[j]) * (a - mean[j]) / float64(len(rows))
		}
	}
	for j := range sd {
		sd[j] = math.Sqrt(sd[j])
	}
	a := make([][]float64, n)
	b, z := make([]float64, n), make([]float64, n)
	for j := range a {
		a[j] = make([]float64, n)
	}
	for i, v := range x {
		c := make([]float64, n)
		for j, w := range v {
			if sd[j] > 1e-12 {
				c[j] = (w - mean[j]) / sd[j]
			}
		}
		for j, w := range c {
			b[j] += w * (rows[i].Outcome - targetMean) / float64(len(rows))
			for k, u := range c {
				a[j][k] += w * u / float64(len(rows))
			}
		}
	}
	for it := 0; it < 256; it++ {
		change := 0.
		for j := range z {
			if sd[j] <= 1e-12 {
				continue
			}
			rhs := b[j]
			for k, w := range z {
				if j != k {
					rhs -= a[j][k] * w
				}
			}
			value := math.Max(-7.99*sd[j], math.Min(7.99*sd[j], rhs/(a[j][j]+ridge)))
			change = math.Max(change, math.Abs(value-z[j]))
			z[j] = value
		}
		if change < 1e-13 {
			break
		}
	}
	h.W = make([]float64, n)
	h.B = targetMean
	h.Known = true
	for j := range z {
		if sd[j] > 1e-12 {
			h.W[j] = nConstraintCoefficient(z[j] / sd[j])
			h.B -= h.W[j] * mean[j]
		}
	}
	h.B = nConstraintCoefficient(h.B)
	return nMetricProposal{Model: m, Name: "standardized-bounded-ridge", Fraction: 1, Valid: nMetricValidHead(m, hi)}
}
