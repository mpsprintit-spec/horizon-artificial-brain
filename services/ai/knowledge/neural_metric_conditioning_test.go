package knowledge

import (
	"math"
	"reflect"
	"testing"
)

func TestMetricReadoutLearnsNewVariableWithTinyConstantOldFeature(t *testing.T) {
	axisOld, axisNew := make([]float64, nDimensions), make([]float64, nDimensions)
	axisOld[0], axisNew[1] = 1, 1
	q := []NodeID{10}
	m := nModel{MetricEnabled: true, Circuits: []nCircuit{
		{Meta: NFeature{Node: 1, Kind: 1}, Axis: axisOld, Center: 1e-15, Delta: .25},
		{Meta: NFeature{Node: 2, Kind: 1}, Axis: axisNew, Delta: 1e-4},
	}, Heads: []nHead{{Metric: true, Query: q, W: []float64{0, 0}}, {Metric: true, Query: []NodeID{11}, W: []float64{.7, .2}, B: -.1, Known: true}}}
	retained := m.Heads[1]
	rows := []NWitness{}
	for _, v := range []float64{-.8, -.4, 0, .4, .8} {
		x := make([]float64, nDimensions)
		x[1] = v
		rows = append(rows, NWitness{X: x, Query: q, Outcome: .3 + .4*v})
	}
	if !nFitMetric(&m, 0, rows) {
		t.Fatal("uninformative constant column blocked a learnable new relation")
	}
	for _, v := range []float64{-.71, -.13, .19, .63} {
		x := make([]float64, nDimensions)
		x[1] = v
		if got := nPredict(m, x, q); math.Abs(got-(.3+.4*v)) > 3e-5 {
			t.Fatalf("new input %g: got %g", v, got)
		}
	}
	if !reflect.DeepEqual(retained, m.Heads[1]) {
		t.Fatal("new readout changed the retained task")
	}
}
