package runtime

import (
	"bytes"
	"fmt"
	"math"
	"path/filepath"
	"sync"
	"testing"

	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestMetricBoundedNonS2RecoveryAndIndependence(t *testing.T) {
	r, wal, dir, start := mNew(t, k.NeuralCompositionMetricV6)
	fit := nGrid([]float64{-.8, -.4, .4, .8}, []float64{-.7, .7})
	check := nGrid([]float64{-.9, -.2, .2, .9}, []float64{-.5, .5})
	fn := func(p nXY) float64 { return .3 * p.x }
	e := nLesson(0, 0, fit, check, fn)
	e.Resolution.Proposal = &k.NMetricProposalConfig{Protocol: k.NMetricBoundedV1, Strategy: "combined", Fractions: []float64{1. / 64, 1. / 16, .25, .5}}
	d := mTeach(t, r, e, 1e-5)
	if !d.Committed || d.ProposalProtocol != k.NMetricBoundedV1 || d.Evidence == nil {
		t.Fatal(d)
	}
	for _, p := range nGrid([]float64{-.6, -.1, .3, .6}, []float64{-.3, .3}) {
		if math.Abs(mRead(t, r, 0, p, 1e-5).NResolution.Value-fn(p)) > 1e-5 {
			t.Fatal("non-S2 precision")
		}
	}
	checkpoint := filepath.Join(dir, "bounded.json")
	compositionCheck(t, r.Checkpoint(checkpoint))
	expected := continuationBytes(t, r)
	for _, cp := range []string{start, checkpoint} {
		rr, err := RebuildRuntimeFromCheckpointAndJournal(wal, cp)
		compositionCheck(t, err)
		if !bytes.Equal(expected, continuationBytes(t, rr)) {
			t.Fatal("bounded checkpoint/WAL replay")
		}
	}
	for _, w := range r.brain.BrainState.NResolution.Witnesses {
		if w.Role != "evaluation" {
			t.Fatal("check witness lost role")
		}
	}
	// The same measurements with different IDs must fail before a WAL append.
	bad := nLesson(0, 1, fit, fit, fn)
	bad.Resolution.Proposal = e.Resolution.Proposal
	before := continuationBytes(t, r)
	seq := r.LastSequence()
	_, _, err := r.LearnCompositionWithDiagnosis(bad, bad.After.ObservedAt)
	if err == nil || seq != r.LastSequence() || !bytes.Equal(before, continuationBytes(t, r)) {
		t.Fatal("encoded overlap admitted or mutated state")
	}
}

func TestMetricBoundedConcurrentReceiptOwnership(t *testing.T) {
	r, _, _, _ := mNew(t, k.NeuralCompositionMetricV6)
	e := nLesson(0, 0, nGrid([]float64{-.8, -.4, .4, .8}, []float64{-.7, .7}), nGrid([]float64{-.9, -.2, .2, .9}, []float64{-.5, .5}), func(p nXY) float64 { return .3 * p.x })
	e.Motor = []float64{0, 0, 0, 0}
	e.After = nScene(e.After.ObservedAt)
	tol := 1e-5
	e.Input.RequiredPrecision = &tol
	e.Resolution.Proposal = &k.NMetricProposalConfig{Protocol: k.NMetricBoundedV1, Strategy: "combined", Fractions: []float64{1. / 64, 1. / 16, .25, .5}}
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 8; i++ {
			lesson := e
			lesson.ID = fmt.Sprintf("bounded-concurrent-%d", i)
			_, rec, err := r.LearnCompositionWithDiagnosis(lesson, lesson.After.ObservedAt)
			if err != nil {
				errs <- err
				return
			}
			if rec == nil || rec.Diagnosis.Evidence == nil {
				errs <- fmt.Errorf("missing detached evidence")
				return
			}
			rec.Diagnosis.Evidence.Fit = -1
			rec.Diagnosis.Trials = nil
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 40; i++ {
			in := e.Input
			if _, err := r.ReadComposition(in, e.After.ObservedAt); err != nil {
				errs <- err
				return
			}
		}
	}()
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	if r.brain.LastNDiagnosis().Evidence.Fit < 4 {
		t.Fatal("receipt aliases canonical evidence")
	}
}
