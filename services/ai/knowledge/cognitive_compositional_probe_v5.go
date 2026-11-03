package knowledge

// V5 research kernel only. No runtime selector, checkpoint schema, optimizer or
// default world decoder calls this code. Full V4 checkpoints are unavailable in
// this recovery, so enabling or training this candidate in the runtime is deferred.
// The alternative decoder reuses rows of the existing canonical Joint tensor;
// it neither allocates a Brain nor supplies an external probability table.

import "errors"

// Only the mathematical incidence structure is compiled. No weights, recurrent
// state, beliefs, operation results or future data are retained here.
type pairwiseStructureV5 struct {
	classes, operations, outcomes, factors int
	active                                 [][]int
}

func compilePairwiseV5(n NeuralMetaConfig) (*pairwiseStructureV5, error) {
	design, err := pairwiseDesignV5(n)
	if err != nil {
		return nil, err
	}
	p := &pairwiseStructureV5{classes: n.Classes, operations: n.Computations, outcomes: n.Outcomes, factors: len(design[0]), active: make([][]int, len(design))}
	for state, row := range design {
		for f, value := range row {
			if value != 0 {
				p.active[state] = append(p.active[state], f)
			}
		}
	}
	return p, nil
}

// Reference-coded unary and pairwise sufficient statistics for (Y,O0,...).
// This is a log-linear family, NOT an independence factorization. Its support
// is the same joint state enumeration as V2. Higher-order interactions are an
// explicit limitation, tested separately from the rank/identifiability result.
func pairwiseDesignV5(n NeuralMetaConfig) ([][]float64, error) {
	if n.Classes < 2 || n.Classes > 16 || n.Computations < 1 || n.Computations > 8 || n.Outcomes < 2 || n.Outcomes > 8 || n.States() > 4096 {
		return nil, errors.New("invalid V5 research dimensions")
	}
	cards := make([]int, n.Computations+1)
	cards[0] = n.Classes
	for i := 1; i < len(cards); i++ {
		cards[i] = n.Outcomes
	}
	rows := make([][]float64, n.States())
	for state := range rows {
		values := make([]int, len(cards))
		values[0] = state % n.Classes
		for op := 0; op < n.Computations; op++ {
			values[op+1] = metaOutcome(n, state, op)
		}
		for i, card := range cards {
			for a := 0; a < card-1; a++ {
				v := 0.
				if values[i] == a {
					v = 1
				}
				rows[state] = append(rows[state], v)
			}
		}
		for i := range cards {
			for j := i + 1; j < len(cards); j++ {
				for a := 0; a < cards[i]-1; a++ {
					for b := 0; b < cards[j]-1; b++ {
						v := 0.
						if values[i] == a && values[j] == b {
							v = 1
						}
						rows[state] = append(rows[state], v)
					}
				}
			}
		}
	}
	return rows, nil
}

// Energy e(s,h)=sum_f psi_f(s)*(W_f h+b_f). The live N tensor is decoded by
// CognitiveParameters before this research function is called. The first F rows
// are a PROPOSED alternative meaning; old checkpoints are never reinterpreted.
func pairwiseEnergiesV5(s *CognitiveModelState, w, h []float64) ([]float64, [][]float64, error) {
	if s == nil || s.Config.NeuralMeta == nil || s.NeuralMeta == nil || len(h) != s.Config.Hidden || len(w) != len(s.Layout.Edges) {
		return nil, nil, errors.New("invalid V5 research state")
	}
	for _, v := range w {
		if !finiteNumber(v) {
			return nil, nil, errors.New("nonfinite N weights")
		}
	}
	for _, v := range h {
		if !finiteNumber(v) {
			return nil, nil, errors.New("nonfinite recurrent state")
		}
	}
	design, err := pairwiseDesignV5(*s.Config.NeuralMeta)
	if err != nil {
		return nil, nil, err
	}
	factors := len(design[0])
	j, b := s.NeuralMeta.Joint, s.NeuralMeta.JointBias
	if j.Rows < factors || j.Cols != len(h) || j.Offset < 0 || b.Offset < 0 || b.Rows < factors || j.Offset+factors*j.Cols > len(w) || b.Offset+factors > len(w) {
		return nil, nil, errors.New("insufficient existing N tensor rows")
	}
	joint := make([]float64, len(design))
	for f := 0; f < factors; f++ {
		v := w[b.Offset+f]
		for k, x := range h {
			v += w[j.Offset+f*j.Cols+k] * x
		}
		for state := range joint {
			joint[state] += design[state][f] * v
		}
	}
	return joint, design, nil
}

// Explicitly compiled equivalent for repeated probes. Shape checks prevent reuse
// across different variable cardinalities. Every call reads current N weights/h.
func pairwiseCompiledEnergiesV5(p *pairwiseStructureV5, s *CognitiveModelState, w, h []float64) ([]float64, error) {
	if p == nil || s == nil || s.Config.NeuralMeta == nil || s.NeuralMeta == nil || len(h) != s.Config.Hidden || len(w) != len(s.Layout.Edges) {
		return nil, errors.New("invalid compiled V5 probe")
	}
	n := s.Config.NeuralMeta
	if n.Classes != p.classes || n.Computations != p.operations || n.Outcomes != p.outcomes {
		return nil, errors.New("compiled incidence shape mismatch")
	}
	j, b := s.NeuralMeta.Joint, s.NeuralMeta.JointBias
	if j.Rows < p.factors || j.Cols != len(h) || j.Offset < 0 || b.Offset < 0 || b.Rows < p.factors || j.Offset+p.factors*j.Cols > len(w) || b.Offset+p.factors > len(w) {
		return nil, errors.New("compiled N tensor shape mismatch")
	}
	for _, v := range w {
		if !finiteNumber(v) {
			return nil, errors.New("nonfinite N weights")
		}
	}
	for _, v := range h {
		if !finiteNumber(v) {
			return nil, errors.New("nonfinite recurrent state")
		}
	}
	coefficients := make([]float64, p.factors)
	for f := range coefficients {
		v := w[b.Offset+f]
		for k, x := range h {
			v += w[j.Offset+f*j.Cols+k] * x
		}
		coefficients[f] = v
	}
	z := make([]float64, len(p.active))
	for state, active := range p.active {
		for _, f := range active {
			z[state] += coefficients[f]
		}
	}
	return z, nil
}

// Exact partial categorical NLL and gradients for the candidate family. The
// existing V4 stable likelihood and native linear/GRU backprop are reused.
// Labels are supplied only to this learning-side kernel, never an inference input.
func pairwiseLossV5(s *CognitiveModelState, w, h []float64, allowed []bool) (float64, []float64, []float64, error) {
	z, design, err := pairwiseEnergiesV5(s, w, h)
	if err != nil {
		return 0, nil, nil, err
	}
	if len(allowed) != len(z) {
		return 0, nil, nil, errors.New("partial observation shape")
	}
	any := false
	for _, yes := range allowed {
		any = any || yes
	}
	if !any {
		return 0, nil, nil, errors.New("impossible supervised event")
	}
	loss, dz := partialCategoricalV4(z, allowed)
	df := make([]float64, len(design[0]))
	for state, d := range dz {
		for f, v := range design[state] {
			df[f] += d * v
		}
	}
	g, dh := make([]float64, len(w)), make([]float64, len(h))
	j, b := s.NeuralMeta.Joint, s.NeuralMeta.JointBias
	j.Rows, b.Rows = len(df), len(df)
	cognitiveLinearBackward(g, dh, w, h, df, j, b)
	return loss, g, dh, nil
}
