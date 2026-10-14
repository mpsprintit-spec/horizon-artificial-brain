package activation

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// TestRecurrentDynamicsEnterADiminishingChangeRegime measures the recurrent
// substrate directly. It deliberately does not use Result.Converged because
// that field means the activation pass completed, not numerical convergence.
//
// The current substrate is dissipative: activation decays by e.Decay while
// recurrent spread can temporarily redistribute activity into downstream
// units. Therefore finite-cycle exact equality is not the numerical contract.
// The contract is that the late tail of the trajectory is bounded and changing
// materially less than the earlier tail.
func TestRecurrentDynamicsEnterADiminishingChangeRegime(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	source := kb.Store("source")
	target := kb.Store("target")
	kb.Connect(source, target, 0.6, 0.8, false)

	engine := NewEngine(kb)
	state := map[knowledge.NodeID]float64{source.ID: 1}
	confidence := map[knowledge.NodeID]float64{source.ID: 1}
	now := time.Unix(1000, 0).UTC()

	const cycles = 32
	const tail = 8
	deltas := make([]float64, 0, cycles-1)
	var previous map[knowledge.NodeID]float64

	for cycle := 0; cycle < cycles; cycle++ {
		state, confidence = engine.advance(state, confidence, now, 1)
		if previous != nil {
			deltas = append(deltas, stateDifference(previous, state))
		}
		previous = cloneState(state)
	}

	if len(state) == 0 {
		t.Fatal("expected recurrent dynamics to retain an active state")
	}

	maxActivation := 0.0
	for _, level := range state {
		if level > maxActivation {
			maxActivation = level
		}
	}
	// squash() deliberately maps values above one into the finite interval
	// [1,2); the substrate's normalization contract is therefore bounded
	// rather than strictly limited to [0,1].
	if maxActivation >= 2 {
		t.Fatalf("recurrent state escaped normalized bound: max=%g state=%v", maxActivation, state)
	}

	if len(deltas) < tail*2 {
		t.Fatalf("not enough samples for convergence tail contract: got=%d", len(deltas))
	}

	previousTail := deltas[len(deltas)-tail*2 : len(deltas)-tail]
	finalTail := deltas[len(deltas)-tail:]
	mean := func(values []float64) float64 {
		var total float64
		for _, value := range values {
			total += value
		}
		return total / float64(len(values))
	}

	previousMean := mean(previousTail)
	finalMean := mean(finalTail)

	// Require a meaningful reduction in trajectory change, rather than exact
	// finite-cycle equality. This preserves the ability of new evidence to
	// perturb the state while asserting that the current trajectory is settling.
	if finalMean >= previousMean*0.75 {
		t.Fatalf(
			"expected recurrent trajectory to enter a diminishing-change regime: previous_tail_mean=%g final_tail_mean=%g state=%v",
			previousMean,
			finalMean,
			state,
		)
	}
}

// TestDifferentStimuliCanPerturbAStableRecurrentState guards against a
// permanently frozen recurrent attractor. A stable state is acceptable only
// when new evidence can still perturb the dynamics.
func TestDifferentStimuliCanPerturbAStableRecurrentState(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	a := kb.Store("a")
	b := kb.Store("b")
	c := kb.Store("c")
	kb.Connect(a, c, 0.7, 0.9, false)
	kb.Connect(b, c, 0.2, 0.9, false)

	engine := NewEngine(kb)
	now := time.Unix(2000, 0).UTC()

	stateA, confA := engine.advance(
		map[knowledge.NodeID]float64{a.ID: 1},
		map[knowledge.NodeID]float64{a.ID: 1},
		now,
		16,
	)
	stateB, _ := engine.advance(
		map[knowledge.NodeID]float64{b.ID: 1},
		map[knowledge.NodeID]float64{b.ID: 1},
		now,
		16,
	)

	if stateDifference(stateA, stateB) <= 1e-6 {
		t.Fatalf("different stimuli collapsed to the same recurrent state: A=%v B=%v", stateA, stateB)
	}

	_ = confA
}
