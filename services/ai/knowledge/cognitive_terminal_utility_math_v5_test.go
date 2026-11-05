package knowledge

import (
	"math"
	"testing"
)

func TestTerminalUtilityV5AllHeadGradientsAndPropensityExpectation(t *testing.T) {
	p := &Phase4StateV5{Spec: Phase4SpecV5{Version: Phase4VersionV5, LearningRate: .003, EntropyWeight: .02},
		BaseParameters: 7, WI: CognitiveTensor{7, 16, 58}, BI: CognitiveTensor{935, 16, 1},
		Actor: CognitiveTensor{951, 8, 16}, ActorBias: CognitiveTensor{1079, 8, 1}, Critic: CognitiveTensor{1087, 1, 16}, CriticBias: CognitiveTensor{1103, 1, 1}}
	w := make([]float64, 1104)
	for i := range w {
		w[i] = .03 * math.Sin(float64(i+1))
	}
	x := make([]float64, 58)
	for i := range x {
		x[i] = .1 * math.Cos(float64(i))
	}
	truth := [5]float64{.5, .125, .25, .125, .35}
	row := terminalUtilityRowV5{F: x, U: truth}
	for _, kind := range []string{"policy_value", "logscore"} {
		t.Run(kind, func(t *testing.T) {
			_, g := terminalUtilityObjectiveV5(p, w, []terminalUtilityRowV5{row}, kind)
			maxerr := 0.
			for i, v := range w {
				if (i < p.BaseParameters || i >= p.Critic.Offset) && g[i] != 0 {
					t.Fatal("gradient owner leak")
				}
				if i < p.BaseParameters {
					continue
				}
				const eps = 1e-5
				w[i] = v + eps
				pos, _ := terminalUtilityObjectiveV5(p, w, []terminalUtilityRowV5{row}, kind)
				w[i] = v - eps
				neg, _ := terminalUtilityObjectiveV5(p, w, []terminalUtilityRowV5{row}, kind)
				w[i] = v
				maxerr = math.Max(maxerr, math.Abs((pos-neg)/(2*eps)-g[i]))
			}
			if maxerr > 1e-7 {
				t.Fatalf("gradient error %g", maxerr)
			}
			expected := make([]float64, len(w))
			for a := 0; a < 4; a++ {
				for reward := 0; reward < 2; reward++ {
					prob := .25 * truth[a]
					if reward == 0 {
						prob = .25 * (1 - truth[a])
					}
					observed := terminalUtilityRowV5{F: x}
					observed.U[4] = .35
					observed.U[a] = float64(reward) / .25
					_, grad := terminalUtilityObjectiveV5(p, w, []terminalUtilityRowV5{observed}, kind)
					for i, v := range grad {
						expected[i] += prob * v
					}
				}
			}
			for i, v := range g {
				if math.Abs(v-expected[i]) > 1e-12 {
					t.Fatalf("propensity expectation error at %d", i)
				}
			}
			t.Logf("1097 coordinates max finite-difference error %g; logged-action estimator exact in expectation", maxerr)
		})
	}
}

func TestTerminalUtilityV5RareActionLogscoreGradient(t *testing.T) {
	p := &Phase4StateV5{Spec: Phase4SpecV5{EntropyWeight: .02},
		BaseParameters: 7, WI: CognitiveTensor{7, 16, 58}, BI: CognitiveTensor{935, 16, 1},
		Actor: CognitiveTensor{951, 8, 16}, ActorBias: CognitiveTensor{1079, 8, 1}, Critic: CognitiveTensor{1087, 1, 16}, CriticBias: CognitiveTensor{1103, 1, 1}}
	w := make([]float64, 1104)
	for i := 0; i < 16; i++ {
		w[p.BI.Offset+i] = 7
		w[p.Actor.Offset+3*16+i] = -7
		w[p.Actor.Offset+7*16+i] = 7
	}
	x := make([]float64, 58)
	_, probability, _ := phase4Forward(p, w, x, []bool{false, false, false, true, true, true, true, true})
	if probability[3] <= 0 || probability[3] >= 1e-15 {
		t.Fatal("test must exercise a representable very rare legal action")
	}
	rows := []terminalUtilityRowV5{{F: x, U: [5]float64{.5, .125, .25, .125, .35}}}
	loss, gradient := terminalUtilityObjectiveV5(p, w, rows, "logscore")
	if !finiteNumber(loss) {
		t.Fatal("rare-action loss is not finite")
	}
	maxError := 0.
	for i := p.BaseParameters; i < len(w); i++ {
		value := w[i]
		const epsilon = 1e-5
		w[i] = value + epsilon
		positive, _ := terminalUtilityObjectiveV5(p, w, rows, "logscore")
		w[i] = value - epsilon
		negative, _ := terminalUtilityObjectiveV5(p, w, rows, "logscore")
		w[i] = value
		maxError = math.Max(maxError, math.Abs((positive-negative)/(2*epsilon)-gradient[i]))
	}
	if maxError > 1e-7 {
		t.Fatalf("rare-action gradient error %g", maxError)
	}
	t.Logf("rare legal action probability %g; 1097-coordinate maximum error %g", probability[3], maxError)
}
