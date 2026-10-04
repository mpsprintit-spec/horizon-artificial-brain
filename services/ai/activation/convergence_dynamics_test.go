package activation

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// TestRecurrentDynamicsReachAStableState measures the recurrent substrate directly.
// It deliberately does not use Result.Converged because that field currently means
// "the activation pass completed", not numerical convergence.
func TestRecurrentDynamicsReachAStableState(t *testing.T) {
	kb := knowledge.NewKnowledgeBase()
	source := kb.Store("source")
	target := kb.Store("target")
	kb.Connect(source, target, 0.6, 0.8, false)

	engine := NewEngine(kb)
	state := map[knowledge.NodeID]float64{source.ID: 1}
	confidence := map[knowledge.NodeID]float64{source.ID: 1}
	now := time.Unix(1000, 0).UTC()

	var previous map[knowledge.NodeID]float64
	var lastDelta float64
	for cycle := 0; cycle < 32; cycle++ {
		state, confidence = engine.advance(state, confidence, now, 1)
		if previous != nil {
			lastDelta = stateDifference(previous, state)
		}
		previous = cloneState(state)
	}

	if len(state) == 0 {
		t.Fatal("expected recurrent dynamics to retain an active state")
	}
	if lastDelta > 1e-9 {
		t.Fatalf("expected this deterministic recurrent system to settle after repeated cycles: final delta=%g state=%v", lastDelta, state)
	}
}

// TestDifferentStimuliCanEscapeThePreviousRecurrentState guards against a
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
