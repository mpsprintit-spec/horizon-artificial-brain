package core

import "testing"

func TestPulseUsesCanonicalCognitivePipeline(t *testing.T) {
	horizon := NewHorizonEngine()
	result, err := horizon.Pulse("burung elang")
	if err != nil {
		t.Fatalf("Pulse() error = %v", err)
	}
	if result.State.BrainIdentity != "horizon-primary-brain" {
		t.Fatalf("brain identity = %q, want horizon-primary-brain", result.State.BrainIdentity)
	}
	if result.Interpretation.BrainIdentity != "horizon-primary-brain" {
		t.Fatalf("interpretation brain identity = %q, want horizon-primary-brain", result.Interpretation.BrainIdentity)
	}
	if len(result.Answer.NodeIDs) == 0 {
		t.Fatal("canonical Pulse produced no cognitive answer nodes")
	}
	if result.Recommendation != nil {
		t.Fatal("canonical Pulse must not bypass the action-safety boundary")
	}
}
