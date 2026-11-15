package knowledge

import "testing"

func TestNativeRuntimeViewMatchesDigestAndIsIsolated(t *testing.T) {
	b, c, at := nativeControlFixture(t, 1)
	b.BrainState.NativeControl.Config.Version = NativeControlCausalVersion
	check := func() {
		t.Helper()
		full := b.NativeControlSnapshot()
		view := b.NativeControlRuntimeView(true)
		if NativeDigest(view.Config) != NativeDigest(full.Config) || NativeDigest(view.Outputs) != NativeDigest(full.Outputs) {
			t.Fatal("metadata mismatch")
		}
		if full.Episode != nil {
			if view.Episode.TraceDigest != NativeDigest(full.Episode) || view.Episode.DecisionCount != len(full.Episode.Decisions) {
				t.Fatal("trajectory digest mismatch")
			}
		}
		if full.PendingContrast != nil && (view.PendingContrast.ID != full.PendingContrast.ID || view.PendingContrast.Episode.DecisionCount != len(full.PendingContrast.Episode.Decisions)) {
			t.Fatal("pending metadata mismatch")
		}
		view.Config.Sources[0] = 999
		view.Config.Operations[0].Gain = .123
		view.Outputs[0][0] = 999
		if NativeDigest(full) != NativeDigest(b.NativeControlSnapshot()) {
			t.Fatal("runtime view aliases canonical data")
		}
	}
	check()
	if err := b.BeginNativeTrial("first", true, at, true, "matched"); err != nil {
		t.Fatal(err)
	}
	state := map[NodeID]float64{c.Sources[0]: .8}
	p, err := b.NativePolicy(state, false, .1)
	if err != nil {
		t.Fatal(err)
	}
	stop := len(c.Operations) - 1
	if err = b.AppendNativeDecision(NativeDecision{State: state, Values: p.Values, Probabilities: p.Probabilities, Action: stop, Cost: c.Operations[stop].Cost, Before: NativeStateDigest(state), After: NativeStateDigest(state), At: at}); err != nil {
		t.Fatal(err)
	}
	check()
	if _, err = b.LearnNativeObservedOutcome(.4, true, "pair", "environment", "first", at, "action"); err != nil {
		t.Fatal(err)
	}
	check()
}
