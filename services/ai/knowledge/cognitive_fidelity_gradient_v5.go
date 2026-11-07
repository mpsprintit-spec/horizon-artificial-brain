package knowledge

// Reverse the deterministic MUSE/BMPS feature path of the executing actor.
// Outcomes, costs, masks, returns, and advantages remain observed/detached.
// No evaluator distribution or counterfactual reward is accepted here.
func fidelityFeaturesBackwardV5(s *CognitiveModelState, w, temps, h, x []float64, input CognitiveInput, prediction CoordinatedPredictionV5, df, g, dh []float64) error {
	n := *s.Config.NeuralMeta
	joint := prediction.PredictedJoint
	classes := metaClassBelief(n, joint)
	utilities, _ := metaUtilities(s, w, h, x, classes, temps, true)
	du := make([][]float64, n.Classes)
	for y := range utilities {
		utilities[y][n.Classes] = .35
		du[y] = make([]float64, n.Classes+1)
	}
	dj := make([]float64, len(joint))
	for a := 0; a < n.Classes; a++ {
		d := df[32+a] + df[37+a]
		for i, p := range joint {
			y := metaTarget(n, i)
			dj[i] += d * utilities[y][a]
			du[y][a] += d * p
		}
	}
	// Probability normalizers cancel in the expected maximum:
	// P(o) max_a E[U_a | o] = max_a sum_i P(i) P(o|i) U(i,a).
	// Differentiate the selected piece of each max; first argmax matches forward.
	addMax := func(coefficient float64, selectState func(int) float64) {
		if coefficient == 0 {
			return
		}
		values := make([]float64, n.Classes+1)
		for i, p := range joint {
			for a, u := range utilities[metaTarget(n, i)] {
				values[a] += p * selectState(i) * u
			}
		}
		best := -1
		for a, v := range values {
			if input.Mask[n.Computations+a] && (best < 0 || v > values[best]) {
				best = a
			}
		}
		for i, p := range joint {
			y, d := metaTarget(n, i), coefficient*selectState(i)
			dj[i] += d * utilities[y][best]
			du[y][best] += d * p
		}
	}
	all := func(int) float64 { return 1 }
	if input.NeuralMeta.Remaining > 0 {
		bi := 0
		perfect := 0.
		for op := 0; op < n.Computations; op++ {
			if !input.Mask[op] {
				continue
			}
			f := prediction.BMPS.Features[bi]
			bi++
			weights := prediction.BMPS.Weights
			myopic := df[49+op] + df[42+op]*weights[0]
			subset := df[55+op] + df[42+op]*weights[2]
			if f.Perfect > 0 {
				perfect += df[52+op] + df[42+op]*weights[1]
			}
			if f.Myopic > 0 {
				for o := 0; o < n.Outcomes; o++ {
					addMax(myopic, func(i int) float64 {
						if input.NeuralMeta.UniformLikelihood {
							return 1 / float64(n.Outcomes)
						}
						if metaOutcome(n, i, op) == o {
							return 1
						}
						return 0
					})
				}
				addMax(-myopic, all)
			}
			if f.Subset > 0 {
				for o := 0; o < n.Outcomes; o++ {
					addMax(subset, func(i int) float64 {
						if metaOutcome(n, i, op) == o {
							return 1
						}
						return 0
					})
				}
				addMax(-subset, all)
			}
		}
		for tuple := 0; tuple < len(joint)/n.Classes; tuple++ {
			addMax(perfect, func(i int) float64 {
				if i/n.Classes == tuple {
					return 1
				}
				return 0
			})
		}
		addMax(-perfect, all)
	}
	// Learned competence depends on the same recurrent weights as observed
	// history. Reverse its hypothetical GRU, then the observed GRU in the caller.
	for y := 0; y < n.Classes; y++ {
		for a := 0; a < n.Classes; a++ {
			plan := cognitiveGRU(s.Config, s.Layout, w, metaPlanFeatures(s.Config, x, y, n.Computations+a), h)
			q := utilities[y][a]
			ds, dp := make([]float64, 5), make([]float64, s.Config.Hidden)
			ds[4] = du[y][a] * q * (1 - q) / temps[5]
			cognitiveLinearBackward(g, dp, w, plan.h, ds, s.Layout.Assessment, s.Layout.AssessmentBias)
			back := cognitiveGRUBackward(s.Config, s.Layout, w, g, plan, dp)
			for i, d := range back {
				dh[i] += d
			}
		}
	}
	// The observed-support softmax is equivalent to conditioning and renormalizing
	// the positive native world distribution. Zero support gets exactly zero credit.
	mean := 0.
	for i, p := range joint {
		mean += p * dj[i]
	}
	for i, p := range joint {
		dj[i] = p * (dj[i] - mean) / temps[0]
	}
	j, bias := phase2TensorsV5(s)
	dz := dj
	if s.Phase2V5 != nil && s.Phase2V5.Spec.Representation != "categorical" {
		structure, err := phase2StructureForV5(n, s.Phase2V5.Spec)
		if err != nil {
			return err
		}
		dz = make([]float64, j.Rows)
		for state, row := range structure.Dense {
			for f, v := range row {
				dz[f] += v * dj[state]
			}
		}
	}
	cognitiveLinearBackward(g, dh, w, h, dz, j, bias)
	// The existing ownership mask discards world/assessment parameter gradients;
	// their fixed transforms still pass their input derivative to recurrent N.
	return nil
}
