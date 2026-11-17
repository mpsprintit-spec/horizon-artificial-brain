package runtime

import (
	"bytes"
	"path/filepath"
	"testing"

	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestMetricExperimentalCapacityCheckpointAndReplay(t *testing.T) {
	r, wal, dir, start := mNew(t, k.NeuralCompositionMetricV6)
	e := nLesson(0, 0, nGrid([]float64{-.8, -.4, .4, .8}, []float64{-.7, .7}), nGrid([]float64{-.9, -.2, .2, .9}, []float64{-.5, .5}), func(p nXY) float64 { return .3 * p.x })
	e.Resolution.Proposal = &k.NMetricProposalConfig{Protocol: k.NMetricBoundedV1, Strategy: "combined", Fractions: []float64{.25, .5}, PreserveShared: true, FeatureLimit: 64}
	if !mTeach(t, r, e, 1e-5).Committed || r.brain.BrainState.NResolution.ExperimentalFeatureLimit != 64 {
		t.Fatal("explicit capacity not admitted")
	}
	cp := filepath.Join(dir, "capacity64.json")
	compositionCheck(t, r.Checkpoint(cp))
	expected := continuationBytes(t, r)
	for _, source := range []string{cp, start} {
		rr, err := RebuildRuntimeFromCheckpointAndJournal(wal, source)
		compositionCheck(t, err)
		if !bytes.Equal(expected, continuationBytes(t, rr)) {
			t.Fatal("capacity metadata or signed parameters lost on recovery")
		}
	}
	// A malformed research bound is rejected before journaling or mutation.
	e.ID = "invalid-limit"
	e.Resolution.Proposal.FeatureLimit = 65
	seq := r.LastSequence()
	if _, _, err := r.LearnCompositionWithDiagnosis(e, e.After.ObservedAt); err == nil || seq != r.LastSequence() || !bytes.Equal(expected, continuationBytes(t, r)) {
		t.Fatal("invalid capacity mutated durable state")
	}
}
