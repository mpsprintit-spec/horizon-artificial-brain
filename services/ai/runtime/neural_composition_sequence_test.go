package runtime

import (
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestSequenceCompositionNativeJournalRestoreAndReadOnly(t *testing.T) {
	r := NewBrainRuntime(knowledge.NewBrain())
	r.SetClock(NewFixedClock(compositionTestBase))
	dir := t.TempDir()
	log, e := OpenEventLog(filepath.Join(dir, "events.jsonl"))
	compositionCheck(t, e)
	defer log.Close()
	r.SetEventLog(log)
	written, e := r.PlanNeuralWrittenGrounding()
	compositionCheck(t, e)
	compositionCheck(t, r.ApplyNeuralWrittenGrounding(written, filepath.Join(dir, "before-written.json"), compositionTestBase))
	cfg := knowledge.DefaultCompositionConfig()
	cfg.Version = knowledge.NeuralCompositionSequenceV4
	cfg.Initialization = "random"
	p, e := r.PlanNeuralComposition(cfg)
	compositionCheck(t, e)
	compositionCheck(t, r.ApplyNeuralComposition(p, filepath.Join(dir, "before-composition.json"), compositionTestBase))
	start := filepath.Join(dir, "start.json")
	compositionCheck(t, r.Checkpoint(start))
	scene := func(at time.Time) knowledge.CompositionScene {
		return knowledge.CompositionScene{Point: []float64{0, 0, 1}, ObservedAt: at, Patches: []knowledge.CompositionPatch{{Values: []float64{.4, -.2, .6, -1, 0}, XY: []float64{0, 0}}}}
	}
	input := func(sign float64) knowledge.CompositionInput {
		q := make([]float64, 16)
		q[14] = 1
		return knowledge.CompositionInput{Words: [][]float64{q}, Scene: scene(compositionTestBase.Add(time.Hour)), Sequence: [][]float64{{sign * .2, .1, .2, .1, .02, .2, 1}, {sign * .3, .2, .1, .1, 0, -.1, 1}}}
	}
	for i := 0; i < 128; i++ {
		at := compositionTestBase.Add(time.Duration(i+1) * time.Millisecond)
		positive := i%2 == 0
		side := -1.
		if positive {
			side = 1
		}
		in := input(side)
		if i < 64 {
			in = knowledge.CompositionInput{}
		}
		in.Scene = scene(at)
		after := scene(at.Add(time.Nanosecond))
		after.Patches[0].Values[3] = side
		_, e = r.LearnComposition(knowledge.CompositionExposure{ID: fmt.Sprintf("seq-%d", i), Source: "numeric-sequence-test", EpisodeID: fmt.Sprint(i), IndependenceGroup: "one-fixed-numeric-fixture", CausalGroup: fmt.Sprint(i), Reliability: .5, Input: in, Motor: []float64{0, 0, side, 1}, After: after}, after.ObservedAt)
		compositionCheck(t, e)
	}
	r.SetClock(NewFixedClock(compositionTestBase.Add(time.Second)))
	final := filepath.Join(dir, "final.json")
	compositionCheck(t, r.Checkpoint(final))
	before := continuationBytes(t, r)
	a, e := r.ReadComposition(input(1), compositionTestBase.Add(time.Hour))
	compositionCheck(t, e)
	b, e := r.ReadComposition(input(-1), compositionTestBase.Add(time.Hour))
	compositionCheck(t, e)
	if len(a.Coordinates) != 6 || len(b.Coordinates) != 6 || reflect.DeepEqual(a.Coordinates, b.Coordinates) {
		t.Fatal("continuous sequence did not affect numerical readout")
	}
	if !reflect.DeepEqual(before, continuationBytes(t, r)) {
		t.Fatal("inference changed Brain")
	}
	restored := NewBrainRuntime(knowledge.NewBrain())
	compositionCheck(t, restored.RestoreCheckpoint(final))
	replayed, e := RebuildRuntimeFromCheckpointAndJournal(log, start)
	compositionCheck(t, e)
	for _, candidate := range []*BrainRuntime{restored, replayed} {
		candidate.SetClock(NewFixedClock(compositionTestBase.Add(time.Second)))
		if !reflect.DeepEqual(before, continuationBytes(t, candidate)) {
			t.Fatal("full canonical restore/replay differs")
		}
		v, e := candidate.ReadComposition(input(1), compositionTestBase.Add(time.Hour))
		compositionCheck(t, e)
		if !reflect.DeepEqual(a, v) {
			t.Fatal("restored/replayed output differs")
		}
	}
}
