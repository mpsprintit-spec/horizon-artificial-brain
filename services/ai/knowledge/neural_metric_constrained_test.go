package knowledge

import (
	"math"
	"testing"
)

func constrainedFixture() (nModel, []NodeID) {
	q := []NodeID{11, 12, 13, 14}
	a, b := make([]float64, nDimensions), make([]float64, nDimensions)
	a[0] = 1
	b[1] = 1
	return nModel{MetricEnabled: true, Circuits: []nCircuit{{Meta: NFeature{Node: 1, Kind: 1}, Axis: a, Delta: 1e-4}, {Meta: NFeature{Node: 2, Kind: 1}, Axis: b, Delta: 1e-4}}, Heads: []nHead{{Query: q, Metric: true, Known: true, W: []float64{.2, 0}}}}, q
}
func constraintRow(q []NodeID, a, b, y float64) NWitness {
	x := make([]float64, nDimensions)
	x[0] = a
	x[1] = b
	return NWitness{X: x, Query: q, Outcome: y}
}
func TestConstrainedSearchFindsSafeNonzeroDirection(t *testing.T) {
	m, q := constrainedFixture()
	old := []NWitness{constraintRow(q, -1, 0, -.2), constraintRow(q, 1, 0, .2)}
	fit := []NWitness{}
	for _, a := range []float64{-.5, .5} {
		for _, b := range []float64{-.5, .5} {
			fit = append(fit, constraintRow(q, a, b, .2*a+.5*b))
		}
	}
	steps := nMetricConstrainedProposals(m, m, q, fit, old, nil, &NMetricProposalConfig{Update: "readout"})
	found := false
	for _, s := range steps {
		if s.Valid && nMetricRegression(m, s.Model, old) == 0 && nError(s.Model, fit) < 1e-6 && s.Constraint.Displacement > .1 {
			found = true
		}
	}
	if !found {
		t.Fatal("solver did not find available orthogonal feasible update")
	}
}
func TestConstrainedNoUsefulDirectionDoesNotFabricateLearning(t *testing.T) {
	m, q := constrainedFixture()
	m.Heads[0].W = []float64{0, 0}
	old := []NWitness{constraintRow(q, 0, 0, 0)}
	fit := []NWitness{constraintRow(q, 0, 0, .5)}
	for _, s := range nMetricConstrainedProposals(m, m, q, fit, old, nil, &NMetricProposalConfig{Update: "bias"}) {
		if nMetricRegression(m, s.Model, old) == 0 && nError(s.Model, fit) < .249999999 {
			t.Fatal("infeasible change called beneficial")
		}
	}
}
func TestRetentionContractIsGroupedNotPointwise(t *testing.T) {
	m, q := constrainedFixture()
	m.Heads[0].B = 0
	m.Heads[0].W = []float64{.5, 0}
	after := nClone(m)
	after.Heads[0].B = .2
	after.Heads[0].W = []float64{.6, 0}
	rows := []NWitness{constraintRow(q, 0, 0, 0), constraintRow(q, 1, 0, 1)}
	if nMetricRegression(m, after, rows) != 0 {
		t.Fatal("valid grouped error tradeoff rejected")
	}
	if math.Abs(nPredict(after, rows[0].X, q)) <= math.Abs(nPredict(m, rows[0].X, q)) {
		t.Fatal("fixture lacks pointwise worsening")
	}
}
func TestConstrainedCoefficientPopulationReconstruction(t *testing.T) {
	for _, v := range []float64{.1, -.03, 7.99, -7.99, 1e-30} {
		q := nConstraintCoefficient(v)
		for _, n := range []int{1, 4, 16} {
			sum := 0.
			for i := 0; i < n; i++ {
				sum += 8 * (math.Max(0, q)/8 - math.Max(0, -q)/8)
			}
			if sum/float64(n) != q {
				t.Fatal("not exact", v, q, n)
			}
		}
	}
}
