package runtime

import (
	"encoding/json"
	"fmt"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestQueryCompositionRuntimeContractAndIndependentRoles(t *testing.T) {
	r := NewBrainRuntime(knowledge.NewBrain())
	r.SetClock(NewFixedClock(compositionTestBase))
	dir := t.TempDir()
	log, err := OpenEventLog(filepath.Join(dir, "events.jsonl"))
	compositionCheck(t, err)
	defer log.Close()
	r.SetEventLog(log)
	compositionCheck(t, r.Checkpoint(filepath.Join(dir, "initial.json")))
	written, err := r.PlanNeuralWrittenGrounding()
	compositionCheck(t, err)
	compositionCheck(t, r.ApplyNeuralWrittenGrounding(written, filepath.Join(dir, "before-written.json"), compositionTestBase))
	cfg := knowledge.DefaultCompositionConfig()
	cfg.Version = knowledge.NeuralCompositionQueryV2
	cfg.Initialization = "random"
	plan, err := r.PlanNeuralComposition(cfg)
	compositionCheck(t, err)
	compositionCheck(t, r.ApplyNeuralComposition(plan, filepath.Join(dir, "before-composition.json"), compositionTestBase))
	start := filepath.Join(dir, "start.json")
	compositionCheck(t, r.Checkpoint(start))
	scene := func(at time.Time) knowledge.CompositionScene {
		return knowledge.CompositionScene{Point: []float64{0, 0, 1}, ObservedAt: at, Patches: []knowledge.CompositionPatch{{Values: []float64{.4, -.2, .6, -1, 0}, XY: []float64{0, 0}}}}
	}
	input := func(state, query int) knowledge.CompositionInput {
		words := [][]float64{}
		for f := 0; f < 3; f++ {
			v := make([]float64, 16)
			v[f] = 1
			v[8] = -1
			if state&(1<<f) != 0 {
				v[8] = 1
			}
			words = append(words, v)
		}
		v := make([]float64, 16)
		v[3] = 1
		v[9+query] = 1
		words = append(words, v)
		return knowledge.CompositionInput{Words: words, Scene: scene(compositionTestBase.Add(time.Hour))}
	}
	step := 0
	teach := func(in knowledge.CompositionInput, positive bool) {
		step++
		at := compositionTestBase.Add(time.Duration(step) * time.Millisecond)
		r.SetClock(NewFixedClock(at))
		in.Scene = scene(at)
		after := scene(at.Add(time.Nanosecond))
		side := -1.
		if positive {
			side = 1
		}
		after.Patches[0].Values[3] = side
		e := knowledge.CompositionExposure{ID: fmt.Sprintf("query-%d", step), Source: "numeric-role-test", EpisodeID: fmt.Sprintf("episode-%d", step), IndependenceGroup: "one-controlled-role-curriculum", CausalGroup: fmt.Sprintf("group-%d", step), Reliability: 1, Input: in, Motor: []float64{0, 0, side, 1}, After: after}
		_, err := r.LearnComposition(e, after.ObservedAt)
		compositionCheck(t, err)
	}
	for i := 0; i < 64; i++ {
		teach(knowledge.CompositionInput{}, i%2 == 0)
	}
	for epoch := 0; epoch < 80; epoch++ {
		for _, state := range []int{0, 1, 2, 3, 4, 5, 6, 7} {
			for q := 0; q < 3; q++ {
				teach(input(state, q), state&(1<<q) != 0)
			}
		}
	}
	final := filepath.Join(dir, "final.json")
	r.SetClock(NewFixedClock(compositionTestBase.Add(time.Duration(step+1) * time.Millisecond)))
	compositionCheck(t, r.Checkpoint(final))
	read := func(rt *BrainRuntime) []knowledge.CompositionReadout {
		out := []knowledge.CompositionReadout{}
		for state := 0; state < 8; state++ {
			for q := 0; q < 3; q++ {
				in := input(state, q)
				v, e := rt.ReadComposition(in, in.Scene.ObservedAt)
				compositionCheck(t, e)
				if !v.Accepted || len(v.Goal) < 4 || (v.Goal[3] > 0) != (state&(1<<q) != 0) {
					t.Fatalf("state%d q%d rejected/wrong: %+v", state, q, v)
				}
				out = append(out, v)
			}
		}
		return out
	}
	reference := read(r)
	restored := NewBrainRuntime(knowledge.NewBrain())
	compositionCheck(t, restored.RestoreCheckpoint(final))
	replayed, err := RebuildRuntimeFromCheckpointAndJournal(log, start)
	compositionCheck(t, err)
	if !reflect.DeepEqual(reference, read(restored)) || !reflect.DeepEqual(reference, read(replayed)) {
		t.Fatal("v2 readout changed after restore or replay")
	}
	bytes, err := os.ReadFile(final)
	compositionCheck(t, err)
	var snap BrainSnapshot
	compositionCheck(t, json.Unmarshal(bytes, &snap))
	if snap.Composition.Config.Version != knowledge.NeuralCompositionQueryV2 {
		t.Fatal("version lost in checkpoint")
	}
	t.Log("24/24 independently queried role answers correct on the taught8-combination curriculum; real v2 journal replay and restore equal. Held-out market tests are separate.")
}
