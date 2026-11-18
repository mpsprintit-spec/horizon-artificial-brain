package runtime

import (
	"bytes"
	"path/filepath"
	"reflect"
	"testing"

	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// A new readout must reuse a shared circuit, leave its original function
// exact, and survive a replay beginning before the new query existed.
func TestMetricFixedBasisNewQueryRecovery(t *testing.T) {
	for _, protocol := range []string{k.NMetricReuseV1, k.NMetricReuseV2, k.NMetricReuseV3} {
		t.Run(protocol, func(t *testing.T) { testMetricFixedBasisNewQueryRecovery(t, protocol) })
	}
}
func testMetricFixedBasisNewQueryRecovery(t *testing.T, protocol string) {
	r, wal, dir, _ := mNew(t, k.NeuralCompositionMetricV6)
	fit := nGrid([]float64{-.8, -.4, .4, .8}, []float64{-.7, .7})
	check := nGrid([]float64{-.9, -.2, .2, .9}, []float64{-.5, .5})
	metric := func(p nXY) float64 { return p.x }
	for i := 0; i < 4; i++ {
		mTeach(t, r, nLesson(0, i, fit, check, metric), 1e-5)
	}
	start := filepath.Join(dir, "before-reuse.json")
	compositionCheck(t, r.Checkpoint(start))
	features := append([]k.NFeature(nil), r.brain.BrainState.NResolution.Features...)
	before := mRead(t, r, 0, nXY{.371, -.29}, 1e-5).NResolution.Value
	e := nLesson(1, 10, fit, check, func(p nXY) float64 { return -.3 * p.x })
	e.Resolution.Proposal = &k.NMetricProposalConfig{Protocol: protocol, Strategy: "full-refit", FixedBasis: true, PreserveShared: true, FeatureLimit: 32}
	if protocol == k.NMetricReuseV3 {
		e.Resolution.Proposal.Ridge = .01
	}
	d := mTeach(t, r, e, 1e-5)
	if !d.Committed {
		t.Fatalf("new shared-basis query not learned: %+v", d)
	}
	if !reflect.DeepEqual(features, r.brain.BrainState.NResolution.Features) {
		t.Fatal("shared basis changed")
	}
	if got := mRead(t, r, 0, nXY{.371, -.29}, 1e-5).NResolution.Value; got != before {
		t.Fatal("old function changed")
	}
	if got := mRead(t, r, 1, nXY{.371, -.29}, 1e-5).NResolution.Value; got > -.10 || got < -.13 {
		t.Fatal("new function absent", got)
	}
	replay, err := RebuildRuntimeFromCheckpointAndJournal(wal, start)
	compositionCheck(t, err)
	if !bytes.Equal(continuationBytes(t, r), continuationBytes(t, replay)) {
		t.Fatal("new-query suffix replay differs")
	}
	final := filepath.Join(dir, "reuse-final.json")
	compositionCheck(t, r.Checkpoint(final))
	restarted, err := RebuildRuntimeFromCheckpointAndJournal(wal, final)
	compositionCheck(t, err)
	if !bytes.Equal(continuationBytes(t, r), continuationBytes(t, restarted)) {
		t.Fatal("new-query checkpoint restart differs")
	}
}
