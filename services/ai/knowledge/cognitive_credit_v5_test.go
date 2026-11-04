package knowledge

import (
	"math"
	"testing"
)

// The behavior and target use different features. Enumerate every action path
// in a two-decision environment, and compare the actual detached estimator
// against a finite difference of the exact target return.
func TestPhase6ImportanceMatchesExactMultistepGradient(t *testing.T) {
	p := &Phase4StateV5{BaseParameters: 7, WI: CognitiveTensor{7, 16, 58}, BI: CognitiveTensor{935, 16, 1}, Actor: CognitiveTensor{951, 8, 16}, ActorBias: CognitiveTensor{1079, 8, 1}, Critic: CognitiveTensor{1087, 1, 16}, CriticBias: CognitiveTensor{1103, 1, 1}}
	w := make([]float64, 1104)
	for i := range w {
		w[i] = .03 * math.Sin(float64(i+1))
	}
	root, child := make([]float64, 58), make([]float64, 58)
	for j := range root {
		root[j] = .2 * math.Cos(float64(j))
		child[j] = .2 * math.Sin(float64(j))
	}
	rootMask := []bool{true, false, false, false, false, false, false, true}
	childMask := []bool{false, false, false, true, false, false, false, true}
	bx := append([]float64(nil), root...)
	bx[42] += .7
	by := append([]float64(nil), child...)
	by[32] -= .6
	_, bp, _ := phase4Forward(p, w, bx, rootMask)
	_, bc, _ := phase4Forward(p, w, by, childMask)
	qr := phase4BehaviorPolicy(bp, rootMask, .25)
	qc := phase4BehaviorPolicy(bc, childMask, .25)

	stop := Phase4FrameV5{Features: root, Mask: rootMask, Action: 7, Reward: .35, BehaviorProbability: qr[7]}
	op := Phase4FrameV5{Features: root, Mask: rootMask, Action: 0, Reward: -.05, BehaviorProbability: qr[0]}
	win := Phase4FrameV5{Features: child, Mask: childMask, Action: 3, Reward: 1, BehaviorProbability: qc[3]}
	wait := Phase4FrameV5{Features: child, Mask: childMask, Action: 7, Reward: .35, BehaviorProbability: qc[7]}
	paths := [][]Phase4FrameV5{{stop}, {op, win}, {op, wait}}
	masses := []float64{qr[7], qr[0] * qc[3], qr[0] * qc[7]}
	got, naive := make([]float64, len(w)), make([]float64, len(w))
	for i, frames := range paths {
		b := Phase4BatchV5{Episodes: []Phase4EpisodeV5{{Frames: frames}}}
		a, r, weights := phase4ImportanceTargets(p, w, b)
		_, g := phase4WeightedObjective(p, w, b, a, r, nil, &weights)
		a0, r0 := phase4Targets(p, w, b)
		for j := range frames {
			weights.through[0][j] /= weights.before[0][j]
		}
		_, g0 := phase4WeightedObjective(p, w, b, a0, r0, nil, &weights)
		for j := range got {
			got[j] -= masses[i] * g[j]
			naive[j] -= masses[i] * g0[j]
		}
	}
	exact := func() float64 {
		_, pi, _ := phase4Forward(p, w, root, rootMask)
		_, after, _ := phase4Forward(p, w, child, childMask)
		return pi[7]*.35 + pi[0]*(-.05+after[3]+after[7]*.35)
	}
	maxErr, naiveErr := 0., 0.
	for i, x := range w {
		w[i] = x + 1e-5
		hi := exact()
		w[i] = x - 1e-5
		lo := exact()
		w[i] = x
		want := (hi - lo) / 2e-5
		maxErr = math.Max(maxErr, math.Abs(got[i]-want))
		naiveErr = math.Max(naiveErr, math.Abs(naive[i]-want))
		if i < p.BaseParameters && got[i] != 0 {
			t.Fatalf("gradient escaped head at %d", i)
		}
	}
	if maxErr > 1e-8 || naiveErr < 1e-5 {
		t.Fatalf("exact error=%g, naive error=%g", maxErr, naiveErr)
	}
	t.Logf("1097 head coordinates, exact multi-step gradient max error %.3g; one-step-only correction error %.3g", maxErr, naiveErr)
}

func TestPhase6ObservedOutcomeControlsOperationCredit(t *testing.T) {
	p := &Phase4StateV5{WI: CognitiveTensor{0, 16, 58}, BI: CognitiveTensor{928, 16, 1}, Actor: CognitiveTensor{944, 8, 16}, ActorBias: CognitiveTensor{1072, 8, 1}, Critic: CognitiveTensor{1080, 1, 16}, CriticBias: CognitiveTensor{1096, 1, 1}}
	rootMask := []bool{true, false, false, false, false, false, false, true}
	childMask := []bool{false, false, false, true, false, false, false, true}
	for _, trial := range []struct {
		name   string
		cost   float64
		reward float64
		up     bool
	}{
		{"useful outcome", .05, 1, true},
		{"unchanged stop", .05, .35, false},
		{"incorrect outcome", .05, 0, false},
	} {
		t.Run(trial.name, func(t *testing.T) {
			w, x := make([]float64, 1097), make([]float64, 58)
			w[p.CriticBias.Offset] = .35
			stop := 3
			if trial.reward == .35 {
				stop = 7
			}
			b := Phase4BatchV5{Episodes: []Phase4EpisodeV5{{Frames: []Phase4FrameV5{
				{Features: x, Mask: rootMask, Action: 0, Reward: -trial.cost, BehaviorProbability: .5},
				{Features: x, Mask: childMask, Action: stop, Reward: trial.reward, BehaviorProbability: .5},
			}}}}
			a, r, weights := phase4ImportanceTargets(p, w, b)
			_, g := phase4WeightedObjective(p, w, b, a, r, nil, &weights)
			// Isolate the operation readout gradient. All other coordinates
			// are held fixed for the local preference intervention.
			w[p.ActorBias.Offset] -= .01 * g[p.ActorBias.Offset]
			_, pi, _ := phase4Forward(p, w, x, rootMask)
			if (pi[0] > .5) != trial.up || pi[0] == .5 {
				t.Fatalf("reward credit direction was wrong: %v", pi)
			}
			t.Logf("observed net %.4f, advantage %.4f, operation gradient %.4f, preference %.6f -> %.6f", r[0][0], a[0][0], g[p.ActorBias.Offset], .5, pi[0])
		})
	}
}
