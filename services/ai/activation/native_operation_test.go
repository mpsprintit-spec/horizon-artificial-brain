package activation

import (
	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
	"math"
	"testing"
	"time"
)

func TestNativeOperationContinuityAndEvidence(t *testing.T) {
	e, s, from, to, at := localActivityFixture(t, false)
	e.RestoreState(StateSnapshot{InternalState: map[k.NodeID]float64{from: 1}, InternalConfidence: map[k.NodeID]float64{from: .5}, ObservationPrediction: &ObservationPrediction{At: at, Source: map[k.NodeID]float64{from: 1}}})
	before := s.Dynamic
	observation := k.NativeDigest(e.SnapshotState().ObservationPrediction)
	out, err := e.ExecuteNativeOperation(k.NativeOperation{Kind: "propagate", Gain: 1}, at)
	if err != nil || out.ReadSynapses != 1 || e.CurrentState()[to] == 0 {
		t.Fatal(out, err)
	}
	first := e.CurrentState()
	_, err = e.ExecuteNativeOperation(k.NativeOperation{Kind: "relax", Gain: .5}, at)
	if err != nil || e.CurrentState()[to] != first[to]*.5 {
		t.Fatal("composition lost prior state")
	}
	if s.Dynamic != before || k.NativeDigest(e.SnapshotState().ObservationPrediction) != observation {
		t.Fatal("thought generated empirical learning")
	}
	pre := k.NativeDigest(e.SnapshotState())
	_, _ = e.ExecuteNativeOperation(k.NativeOperation{Kind: "stop"}, at)
	if k.NativeDigest(e.SnapshotState()) != pre {
		t.Fatal("stop mutated activity")
	}
}
func TestSparseObservedEligibilityMatchesEagerLaw(t *testing.T) {
	for _, inh := range []bool{false, true} {
		e, s, from, to, at := localActivityFixture(t, inh)
		lazy, ls, _, _, _ := localActivityFixture(t, inh)
		for i := 1; i <= 100; i++ {
			now := at.Add(time.Duration(i) * 37 * time.Millisecond)
			var pre, post map[k.NodeID]float64
			if i%11 == 0 {
				pre = map[k.NodeID]float64{from: .7}
				post = map[k.NodeID]float64{to: .8}
			}
			e.ApplyActivityPlasticityWithDynamics(pre, post, now, k.LocalActivityDynamics)
			lazy.ApplySparseObservedPlasticity(pre, post, now)
			if math.Abs(s.Dynamic.Eligibility-decayEligibility(ls.Dynamic.Eligibility, ls.Dynamic.LastEligibilityUpdate, now)) > 2e-15 || math.Abs(s.Weight-ls.Weight) > 2e-15 || math.Abs(s.Confidence-ls.Confidence) > 2e-15 || s.Dynamic.Frequency != ls.Dynamic.Frequency {
				t.Fatal("lazy effective law mismatch", i)
			}
		}
	}
}

func TestNativeTransitionRejectsStaleForeignAndReusedPlans(t *testing.T) {
	e, _, from, _, at := localActivityFixture(t, false)
	e.RestoreState(StateSnapshot{InternalState: map[k.NodeID]float64{from: 1}, InternalConfidence: map[k.NodeID]float64{from: .5}})
	op := k.NativeOperation{Kind: "relax", Gain: .5}
	plan, err := e.PrepareNativeOperation(op, at, 100, 100)
	if err != nil {
		t.Fatal(err)
	}
	other, _, _, _, _ := localActivityFixture(t, false)
	other.RestoreState(e.SnapshotState())
	if _, err = other.CommitNativeOperation(plan); err == nil {
		t.Fatal("foreign plan accepted")
	}
	snapshot := e.SnapshotState()
	snapshot.InternalConfidence[from] = .7
	e.RestoreState(snapshot)
	if _, err = e.CommitNativeOperation(plan); err == nil {
		t.Fatal("stale confidence accepted")
	}
	plan, err = e.PrepareNativeOperation(op, at, 100, 100)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = e.CommitNativeOperation(plan); err != nil {
		t.Fatal(err)
	}
	if _, err = e.CommitNativeOperation(plan); err == nil {
		t.Fatal("plan consumed twice")
	}
}
