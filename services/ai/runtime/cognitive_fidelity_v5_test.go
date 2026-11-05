package runtime

import (
	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
	"testing"
	"time"
)

func TestFidelityV5CanonicalBMPSSearchAfterHeadInitialization(t *testing.T) {
	rt, _, _ := phase4Fixture(t)
	at := time.Unix(2400000000, 0)
	spec := k.Phase4SpecV5{Version: k.Phase4VersionV5, Seed: 101, LearningRate: .003, EntropyWeight: .02, ValueWeight: .2}
	if _, err := rt.ApplyCognitiveModel(CognitiveModelCommand{ID: "fidelity-init", Phase4Init: &spec}, at); err != nil {
		t.Fatal(err)
	}
	b := coordinatedRuntimeBatchWithMode(t, rt, k.CoordinatedFidelityV5)
	if _, err := rt.ApplyCognitiveModel(CognitiveModelCommand{ID: "fidelity-learn", Coordinated: &b}, at.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	cfg, err := rt.brain.CognitiveConfig()
	if err != nil {
		t.Fatal(err)
	}
	point, err := k.ProposeBMPS(nil, cfg.Seed)
	if err != nil {
		t.Fatal(err)
	}
	history := []k.BMPSEvaluation{{Point: point, Return: .35, Episodes: 16}}
	evidence := k.CognitiveEvidence{ID: "bmps-unit", Source: "unit-measured-return", Verifier: "fixture", Digest: k.CognitiveDigest(history)}
	if _, err = rt.ApplyCognitiveModel(CognitiveModelCommand{ID: "fidelity-search", Search: history, Evidence: &evidence}, at.Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	trace := b.Episodes[0].Traces[0]
	p, err := rt.PreviewCoordinatedV5(trace, k.Phase4DropV5{})
	if err != nil {
		t.Fatal(err)
	}
	w, err := rt.brain.CognitiveBMPSWeights()
	if err != nil {
		t.Fatal(err)
	}
	if p.BMPS == nil || p.BMPS.Weights != w {
		t.Fatal("BMPS search not connected to canonical inference")
	}
}
