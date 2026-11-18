package runtime

import (
	"bytes"
	"encoding/json"
	"errors"
	"math"
	"path/filepath"
	"testing"

	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestMetricAdaptiveNovelDirectionAndRestart(t *testing.T) {
	r, wal, dir, start := mNew(t, k.NeuralCompositionMetricV6)
	fit := nGrid([]float64{-.8, -.4, .4, .8}, []float64{-.7, .7})
	check := nGrid([]float64{-.9, -.2, .2, .9}, []float64{-.5, .5})
	p := &k.NMetricProposalConfig{Protocol: k.NMetricAdaptiveV1, Strategy: "combined", Fractions: []float64{.25, .5}, PreserveShared: true, Ridge: .001, FeatureLimit: 97}
	for i := 0; i < 4; i++ {
		e := nLesson(0, i, fit, check, func(v nXY) float64 { return .3 * v.x })
		e.Resolution.Proposal = p
		mTeach(t, r, e, .003)
	}
	old := mRead(t, r, 0, nXY{.3, -.3}, .003).NResolution.Value
	features := len(r.brain.BrainState.NResolution.Features)
	for i := 4; i < 8; i++ {
		e := nLesson(1, i, fit, check, func(v nXY) float64 { return .3 * v.y })
		e.Resolution.Proposal = p
		mTeach(t, r, e, .003)
	}
	if len(r.brain.BrainState.NResolution.Features) <= features {
		t.Fatal("new independent direction not recruited")
	}
	if got := mRead(t, r, 0, nXY{.3, -.3}, .003).NResolution.Value; got != old {
		t.Fatal("shared function changed", old, got)
	}
	if got := mRead(t, r, 1, nXY{.3, -.3}, .003).NResolution.Value; math.Abs(got+.09) > .01 {
		t.Fatal("new function not learned", got)
	}
	final := filepath.Join(dir, "adaptive.json")
	compositionCheck(t, r.Checkpoint(final))
	for _, cp := range []string{start, final} {
		rr, err := RebuildRuntimeFromCheckpointAndJournal(wal, cp)
		compositionCheck(t, err)
		if !bytes.Equal(continuationBytes(t, r), continuationBytes(t, rr)) {
			t.Fatal("adaptive replay differs")
		}
	}
	before := len(r.brain.BrainState.NResolution.Features)
	e := nLesson(1, 8, fit, check, func(v nXY) float64 { return .3 * v.y })
	e.Resolution.Proposal = p
	mTeach(t, r, e, .02)
	if len(r.brain.BrainState.NResolution.Features) != before {
		t.Fatal("gratuitous recruitment on adequate representation")
	}
}

func TestMetricAdaptiveLostAckRecovery(t *testing.T) {
	r, wal, _, start := mNew(t, k.NeuralCompositionMetricV6)
	fit := nGrid([]float64{-.8, -.4, .4, .8}, []float64{-.7, .7})
	check := nGrid([]float64{-.9, -.2, .2, .9}, []float64{-.5, .5})
	e := nLesson(0, 0, fit, check, func(v nXY) float64 { return .3 * v.x })
	e.Motor = []float64{0, 0, 0, 0}
	e.After = nScene(e.After.ObservedAt)
	e.Resolution.Proposal = &k.NMetricProposalConfig{Protocol: k.NMetricAdaptiveV1, Strategy: "combined", Fractions: []float64{.25, .5}, PreserveShared: true, Ridge: .001}
	wal.beforeAppend = func(record LoggedEvent) error {
		b, err := json.Marshal(record)
		if err != nil {
			return err
		}
		if _, err = wal.file.Write(append(b, '\n')); err != nil {
			return err
		}
		if err = wal.file.Sync(); err != nil {
			return err
		}
		return errors.New("lost acknowledgment after durable adaptive command")
	}
	if _, _, err := r.LearnCompositionWithDiagnosis(e, e.After.ObservedAt); !errors.Is(err, ErrRuntimeRecoveryRequired) {
		t.Fatal("uncertain adaptive write stayed writable", err)
	}
	wal.beforeAppend = nil
	rr, err := RebuildRuntimeFromCheckpointAndJournal(wal, start)
	compositionCheck(t, err)
	if !rr.brain.LastNDiagnosis().Committed || len(rr.brain.BrainState.NResolution.Features) == 0 {
		t.Fatal("adaptive recovery lost admitted structure")
	}
	before := continuationBytes(t, rr)
	rr.SetEventLog(wal)
	if _, _, err = rr.LearnCompositionWithDiagnosis(e, e.After.ObservedAt); err == nil || !bytes.Equal(before, continuationBytes(t, rr)) {
		t.Fatal("duplicate adaptive write changed recovered state")
	}
}
