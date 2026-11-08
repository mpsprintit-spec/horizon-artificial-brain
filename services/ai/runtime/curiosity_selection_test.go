package runtime

import (
	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
	"math"
	"math/rand"
	"sort"
	"testing"
	"time"
)

// Frozen pre-optimization law, including its stable NaN ordering. This checks
// exact target/boost equivalence across ties, signs, clamps and registry growth.
func oldCuriosityWinner(brain *k.Brain) (k.NodeID, float64, bool) {
	type candidate struct {
		id    k.NodeID
		score float64
	}
	xs := []candidate{}
	p := brain.BrainState.LearningPolicyState
	pe := clamp01(brain.BrainState.PredictionState["last_error"])
	for _, n := range brain.Registry.Nodes() {
		if n == nil {
			continue
		}
		usage := 1.0 / (1.0 + float64(maxInt64(n.Frequency, 0)))
		under := clamp01(1.0 - n.Activation)
		novelty := clamp01(usage * clamp01(p.NoveltySensitivity))
		uncertainty := clamp01(.5*under+.5*pe) * clamp01(p.UncertaintySensitivity)
		score := clamp01(p.CuriosityPressure) * clamp01(novelty+uncertainty)
		if score <= 0 {
			continue
		}
		xs = append(xs, candidate{n.ID, score})
	}
	if len(xs) == 0 {
		return 0, 0, false
	}
	sort.SliceStable(xs, func(i, j int) bool {
		if xs[i].score == xs[j].score {
			return xs[i].id < xs[j].id
		}
		return xs[i].score > xs[j].score
	})
	boost := clamp01(xs[0].score * .30)
	return xs[0].id, boost, !(boost <= 0)
}
func TestCuriositySelectionMatchesHistoricalSort(t *testing.T) {
	brain := k.NewBrain()
	rt := NewBrainRuntime(brain)
	rng := rand.New(rand.NewSource(930031))
	for trial := 0; trial < 512; trial++ {
		if trial%4 == 0 {
			_, _, err := brain.Registry.GetOrCreateRepresentation(k.NewNeuralVector([]float64{math.Sin(float64(trial + 1)), math.Cos(float64(trial + 1)), float64(trial + 1)}), 1)
			if err != nil {
				t.Fatal(err)
			}
		}
		p := &brain.BrainState.LearningPolicyState
		p.CuriosityPressure = rng.Float64()*2 - .5
		p.NoveltySensitivity = rng.Float64()*2 - .5
		p.UncertaintySensitivity = rng.Float64()*2 - .5
		brain.BrainState.PredictionState["last_error"] = rng.Float64()*2 - .5
		for _, n := range brain.Registry.Nodes() {
			n.Frequency = int64(rng.Intn(100) - 10)
			n.Activation = rng.Float64()*2 - .5
			if trial%7 == 0 {
				n.Frequency = 0
				n.Activation = .5
			}
		}
		if trial%31 == 0 {
			nodes := brain.Registry.Nodes()
			nodes[len(nodes)/2].Activation = math.NaN()
		}
		id, boost, ok := oldCuriosityWinner(brain)
		at := time.Date(2026, 11, 8, 0, 0, 0, trial, time.UTC)
		got := rt.applyCuriosityDrive(at)
		if (len(got) > 0) != ok {
			t.Fatalf("candidate existence at %d", trial)
		}
		if ok {
			v, present := got[id]
			if !present || !(v == boost || math.IsNaN(v) && math.IsNaN(boost)) {
				t.Fatalf("winner changed at %d: want %d %g got %v", trial, id, boost, got)
			}
		}
		if brain.BrainState.CuriosityState["last_update_unix"] != float64(at.UnixNano()) {
			t.Fatal("heartbeat changed")
		}
	}
}
