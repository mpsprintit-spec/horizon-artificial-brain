package knowledge

import (
	"math"
	"testing"
)

func TestPhase4All1097NativeGradientsAndStopGradient(t *testing.T) {
	testPhase4Gradient(t, nil)
}

func TestPhase4ExplorationAll1097NativeGradients(t *testing.T) {
	testPhase4Gradient(t, [][]float64{{.25}, {.25}})
}

func testPhase4Gradient(t *testing.T, exploration [][]float64) {
	t.Helper()
	p := &Phase4StateV5{Spec: Phase4SpecV5{Version: Phase4VersionV5, Seed: 101, LearningRate: .003, EntropyWeight: .02, ValueWeight: .2}, BaseParameters: 7, WI: CognitiveTensor{7, 16, 58}, BI: CognitiveTensor{935, 16, 1}, Actor: CognitiveTensor{951, 8, 16}, ActorBias: CognitiveTensor{1079, 8, 1}, Critic: CognitiveTensor{1087, 1, 16}, CriticBias: CognitiveTensor{1103, 1, 1}}
	w := make([]float64, 1104)
	for i := range w {
		w[i] = .03 * math.Sin(float64(i+1))
	}
	x := make([]float64, 58)
	for i := range x {
		x[i] = .1 * math.Cos(float64(i))
	}
	b := Phase4BatchV5{Episodes: []Phase4EpisodeV5{{Frames: []Phase4FrameV5{{Features: x, Mask: []bool{true, true, false, true, true, true, true, true}, Action: 3, Reward: 1}}}, {Frames: []Phase4FrameV5{{Features: x, Mask: []bool{false, true, true, true, true, true, true, true}, Action: 7, Reward: .35}}}}}
	a, r := phase4Targets(p, w, b)
	_, g := phase4ExplorationObjective(p, w, b, a, r, exploration)
	maxerr := 0.
	for i := range w {
		if i < 7 {
			if g[i] != 0 {
				t.Fatal("stop-gradient leaked")
			}
			continue
		}
		old := w[i]
		w[i] = old + 1e-5
		plus, _ := phase4ExplorationObjective(p, w, b, a, r, exploration)
		w[i] = old - 1e-5
		minus, _ := phase4ExplorationObjective(p, w, b, a, r, exploration)
		w[i] = old
		err := math.Abs((plus.Loss-minus.Loss)/2e-5 - g[i])
		maxerr = math.Max(maxerr, err)
		if err > 1e-7 {
			t.Fatalf("gradient%d err%g", i, err)
		}
	}
	t.Logf("all1097 native derivatives; maxerr=%g; frozen prefix zero; detached advantage", maxerr)
}
