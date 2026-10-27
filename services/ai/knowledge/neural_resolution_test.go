package knowledge

import "testing"

func TestNRetentionDoesNotFreezeWrongUncertainAnswers(t *testing.T) {
	axis := make([]float64, nDimensions)
	axis[0] = 1
	query := []NodeID{100}
	before := nModel{Circuits: []nCircuit{{Meta: NFeature{Node: 1, Kind: 1}, Axis: axis, Delta: 1}}, Heads: []nHead{{Query: query, W: []float64{.1}, Error: .95, Known: true}}}
	makeRow := func(x, y float64) NWitness {
		v := make([]float64, nDimensions)
		v[0] = x
		return NWitness{X: v, Query: query, Outcome: y}
	}
	rows := []NWitness{makeRow(1, 1), makeRow(-1, -1), makeRow(.5, -1)}
	after := nClone(before)
	after.Heads[0].W[0] = .3
	if nError(after, rows) >= nError(before, rows) {
		t.Fatal("diagnostic fixture did not improve aggregate error")
	}
	old := nPredict(before, rows[2].X, query) - rows[2].Outcome
	next := nPredict(after, rows[2].X, query) - rows[2].Outcome
	if next*next <= old*old {
		t.Fatal("fixture must worsen an already wrong, uncertain residual")
	}
	if nRegression(before, after, rows) != 0 {
		t.Fatal("an uncertain mistake was frozen as protected knowledge")
	}
	confident := nClone(before)
	confident.Heads[0].W[0] = 1
	confident.Heads[0].Error = .01
	wrong := nClone(confident)
	wrong.Heads[0].W[0] = -1
	if nRegression(confident, wrong, rows) == 0 {
		t.Fatal("accepted correct distinction was allowed to flip")
	}
}
