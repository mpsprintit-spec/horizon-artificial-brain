package runtime

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

var compositionTestBase = time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)

func compositionCheck(t testing.TB, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func newCompositionTestRuntime(t testing.TB, seed uint64, initialization string, withJournal bool) (*BrainRuntime, *EventLog, string) {
	t.Helper()
	r := NewBrainRuntime(knowledge.NewBrain())
	r.SetClock(NewFixedClock(compositionTestBase))
	dir := t.TempDir()
	var log *EventLog
	if withJournal {
		var err error
		log, err = OpenEventLog(filepath.Join(dir, "events.jsonl"))
		compositionCheck(t, err)
		t.Cleanup(func() { _ = log.Close() })
		r.SetEventLog(log)
	}
	initial := filepath.Join(dir, "initial.json")
	compositionCheck(t, r.Checkpoint(initial))
	written, err := r.PlanNeuralWrittenGrounding()
	compositionCheck(t, err)
	compositionCheck(t, r.ApplyNeuralWrittenGrounding(written, filepath.Join(dir, "old-written.json"), compositionTestBase))
	c := knowledge.DefaultCompositionConfig()
	c.Seed = seed
	c.Initialization = initialization
	plan, err := r.PlanNeuralComposition(c)
	compositionCheck(t, err)
	compositionCheck(t, r.ApplyNeuralComposition(plan, filepath.Join(dir, "old-composition.json"), compositionTestBase))
	return r, log, initial
}

func nativeCompositionInput(input bindInput, at time.Time) knowledge.CompositionInput {
	scene := knowledge.CompositionScene{Point: slices.Clone(input.Point[:]), ObservedAt: at}
	for _, p := range input.Patches {
		scene.Patches = append(scene.Patches, knowledge.CompositionPatch{Values: slices.Clone(p.Values), XY: slices.Clone(p.XY[:])})
	}
	return knowledge.CompositionInput{Words: input.Words, Scene: scene}
}

func nativeCompositionExposure(e bindEpisode, id string, at time.Time) knowledge.CompositionExposure {
	before := nativeCompositionInput(e.Input, at)
	after := knowledge.CompositionScene{Point: slices.Clone(before.Scene.Point), ObservedAt: at.Add(time.Nanosecond)}
	for _, p := range e.After {
		after.Patches = append(after.Patches, knowledge.CompositionPatch{Values: slices.Clone(p.Values), XY: slices.Clone(p.XY[:])})
	}
	return knowledge.CompositionExposure{ID: id, Source: "numeric-simulator-teacher", EpisodeID: id, IndependenceGroup: "one-controlled-curriculum", CausalGroup: id, Reliability: 1, Input: before, Motor: e.Motor, After: after}
}

func compositionTrain(t testing.TB, r *BrainRuntime, seed int64, rounds int, shuffled bool) time.Time {
	t.Helper()
	step := 0
	learn := func(e bindEpisode) {
		step++
		at := compositionTestBase.Add(time.Duration(step) * time.Millisecond)
		exposure := nativeCompositionExposure(e, fmt.Sprintf("lesson-%d", step), at)
		_, err := r.LearnComposition(exposure, exposure.After.ObservedAt)
		compositionCheck(t, err)
	}
	for bg := 0; bg < 4; bg++ {
		for target := 0; target < 7; target++ {
			for _, set := range []bool{false, true} {
				w := bindWorldNew(seed, bg, 0)
				motor := w.motor(target, set, false)
				after, _ := w.apply(motor)
				learn(bindEpisode{Input: w.input(nil, -1), Motor: motor, After: after.patches()})
			}
		}
	}
	train, _ := bindCases()
	assignment := make([][]string, len(train))
	for i, c := range train {
		assignment[i] = slices.Clone(c.Words)
	}
	if shuffled {
		rng := rand.New(rand.NewSource(seed + 20261024))
		rng.Shuffle(len(assignment), func(i, j int) { assignment[i], assignment[j] = assignment[j], assignment[i] })
	}
	order := make([]int, len(train))
	for i := range order {
		order[i] = i
	}
	rng := rand.New(rand.NewSource(seed))
	for round := 0; round < rounds; round++ {
		rng.Shuffle(len(order), func(i, j int) { order[i], order[j] = order[j], order[i] })
		for _, i := range order {
			learn(bindEpisodeFor(train[i], seed, step, assignment[i]))
		}
	}
	return compositionTestBase.Add(time.Duration(step+1) * time.Millisecond)
}

func TestNativeCompositionSealedCurriculum(t *testing.T) {
	train, held := bindCases()
	if len(train) != 82 || len(held) != 236 {
		t.Fatalf("sealed protocol changed: %d/%d", len(train), len(held))
	}
	// Check split separation before training. Inputs include actual context,
	// positions and pointing; words alone are not a complete test identity.
	seen := map[string]bool{}
	key := func(c bindCase) string {
		b, _ := json.Marshal([]any{c.Words, c.Pointer, c.Background, c.Layout})
		return string(b)
	}
	for _, c := range train {
		seen[key(c)] = true
	}
	for _, c := range held {
		if seen[key(c)] {
			t.Fatal("training/heldout overlap")
		}
	}
	for _, initialization := range []string{"separated", "zero", "random"} {
		for _, seed := range []int64{7, 17, 47} {
			// Statistical replication runs in the normal suite. Race covers
			// every initialization at seed 7 plus the dedicated concurrent tests.
			if compositionRaceDetector && seed != 7 {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", initialization, seed), func(t *testing.T) {
				r, _, _ := newCompositionTestRuntime(t, uint64(seed), initialization, false)
				at := compositionTrain(t, r, seed, 24, false)
				before := continuationBytes(t, r)
				correctSeen, correctHeld, unknownAccepted := 0, 0, 0
				for group, cases := range [][]bindCase{train, held} {
					for _, c := range cases {
						world := bindWorldNew(seed, c.Background, c.Layout)
						input := nativeCompositionInput(world.input(c.Words, c.Pointer), at)
						out, err := r.ReadComposition(input, at)
						compositionCheck(t, err)
						// Teacher labels are first accessed after inference completed.
						gold := world.motor(c.Target, c.Set, c.Noop)
						correct := out.Accepted && slices.Equal(out.Motor, gold)
						if correct {
							actual, valid := world.apply(out.Motor)
							expected, _ := world.apply(gold)
							if !valid || actual != expected {
								t.Fatal("motor did not produce the actual intended consequence")
							}
							if gold[3] != 0 && math.Abs(out.LocalEffect-out.Goal[3]) > .02 {
								t.Fatal("local consequence prediction disagrees with simulator")
							}
							if group == 0 {
								correctSeen++
							} else {
								correctHeld++
							}
						}
						if c.Split == "heldout_new_paraphrase" && out.Accepted {
							unknownAccepted++
						}
					}
				}
				ambiguousAccepted := 0
				for _, verb := range []string{"nyalakan", "matikan", "buka", "tutup"} {
					input := nativeCompositionInput(bindWorldNew(seed, 0, 0).input([]string{verb, "itu"}, -1), at)
					out, err := r.ReadComposition(input, at)
					compositionCheck(t, err)
					if out.Accepted {
						ambiguousAccepted++
					}
				}
				if !bytes.Equal(before, continuationBytes(t, r)) {
					t.Fatal("preview mutated complete neural/runtime continuation")
				}
				t.Logf("seen=%d/82 heldout_known=%d/226 unknown_accepted=%d ambiguous_accepted=%d", correctSeen, correctHeld, unknownAccepted, ambiguousAccepted)
				if correctSeen != 82 || unknownAccepted != 0 || ambiguousAccepted != 0 {
					t.Errorf("seen performance or refusal gate regressed")
				}
				if initialization == "separated" && correctHeld != 226 {
					t.Errorf("structured composition transfer regressed: %d/226", correctHeld)
				}
			})
		}
	}
}

func TestNativeCompositionReplayRestoreAndNextThought(t *testing.T) {
	r, log, initial := newCompositionTestRuntime(t, 7, "separated", true)
	at := compositionTrain(t, r, 7, 2, false)
	// A real numeric sensor command and ordinary written observation use the
	// same owner/runtime/journal contract after supervised learning.
	scene := nativeCompositionInput(bindWorldNew(7, 2, 3).input(nil, -1), at).Scene
	_, err := r.ObserveCompositionScene("scene", "simulator", scene)
	compositionCheck(t, err)
	_, _, err = NewCognitiveOrchestrator(r).ProcessObservation(Event{ID: "ordinary-written", Timestamp: at.Add(time.Millisecond), Cycles: 1}, ObservationInput{Modality: "language", Source: "user", Tokens: []string{"nyalakan", "televisi"}}, nil)
	compositionCheck(t, err)
	r.SetClock(NewFixedClock(at.Add(2 * time.Millisecond)))
	checkpoint := filepath.Join(t.TempDir(), "composition.json")
	compositionCheck(t, r.Checkpoint(checkpoint))
	data, err := os.ReadFile(checkpoint)
	compositionCheck(t, err)
	var snapshot BrainSnapshot
	compositionCheck(t, json.Unmarshal(data, &snapshot))
	if snapshot.SchemaVersion != 14 || snapshot.Composition == nil {
		t.Fatal("composition continuation was not authenticated in schema 14")
	}
	restored := NewBrainRuntime(knowledge.NewBrain())
	compositionCheck(t, restored.RestoreCheckpoint(checkpoint))
	restored.SetClock(NewFixedClock(at.Add(2 * time.Millisecond)))
	if !bytes.Equal(continuationBytes(t, r), continuationBytes(t, restored)) {
		t.Fatal("full restore lost continuation")
	}
	genesis, err := RebuildRuntimeFromCheckpointAndJournal(log, initial)
	compositionCheck(t, err)
	genesis.SetClock(NewFixedClock(at.Add(2 * time.Millisecond)))
	suffix, err := RebuildRuntimeFromCheckpointAndJournal(log, checkpoint)
	compositionCheck(t, err)
	suffix.SetClock(NewFixedClock(at.Add(2 * time.Millisecond)))
	for _, candidate := range []*BrainRuntime{genesis, suffix} {
		if !bytes.Equal(continuationBytes(t, r), continuationBytes(t, candidate)) {
			t.Fatal("native committed replay diverged")
		}
	}
	expected, err := r.CognitiveThink(1)
	compositionCheck(t, err)
	for _, candidate := range []*BrainRuntime{restored, genesis, suffix} {
		got, err := candidate.CognitiveThink(1)
		compositionCheck(t, err)
		if !reflect.DeepEqual(expected, got) || !bytes.Equal(continuationBytes(t, r), continuationBytes(t, candidate)) {
			t.Fatal("next thought or continuation differed after recovery")
		}
	}
	loaded := NewBrainRuntime(knowledge.NewBrain())
	compositionCheck(t, loaded.LoadBrain(checkpoint))
	if loaded.composition == nil {
		t.Fatal("startup lost the complete serving contract")
	}
}

func TestNativeCompositionInvalidDuplicateAndUncertainWrites(t *testing.T) {
	r, log, initial := newCompositionTestRuntime(t, 7, "separated", true)
	train, _ := bindCases()
	e := nativeCompositionExposure(bindEpisodeFor(train[0], 7, 0, train[0].Words), "same", compositionTestBase.Add(time.Second))
	before := continuationBytes(t, r)
	revision := r.LastSequence()
	for _, mutate := range []func(*knowledge.CompositionExposure){func(e *knowledge.CompositionExposure) { e.After.Patches[0].Values[4] = math.NaN() }, func(e *knowledge.CompositionExposure) {
		e.Input.Words = append(e.Input.Words, e.Input.Words...)
		e.Input.Words = append(e.Input.Words, e.Input.Words...)
	}, func(e *knowledge.CompositionExposure) { e.CausalGroup = "" }, func(e *knowledge.CompositionExposure) { e.Motor = []float64{1, 1, 1, 1} }} {
		bad, _ := freezeComposition(e)
		mutate(&bad)
		if _, err := r.LearnComposition(bad, e.After.ObservedAt); err == nil {
			t.Fatal("invalid exposure accepted")
		}
		if !bytes.Equal(before, continuationBytes(t, r)) {
			t.Fatal("invalid late field mutated state")
		}
	}
	var wg sync.WaitGroup
	success := 0
	var mu sync.Mutex
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := r.LearnComposition(e, e.After.ObservedAt); err == nil {
				mu.Lock()
				success++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if success != 1 || r.LastSequence() != revision+1 {
		t.Fatal("duplicate teaching applied more than once")
	}
	before = continuationBytes(t, r)
	e.ID = "uncertain"
	log.beforeAppend = func(LoggedEvent) error { return errors.New("lost acknowledgment") }
	if _, err := r.LearnComposition(e, e.After.ObservedAt); !errors.Is(err, ErrRuntimeRecoveryRequired) {
		t.Fatal("uncertain append did not retire runtime")
	}
	if !bytes.Equal(before, continuationBytes(t, r)) {
		t.Fatal("append failure mutated neural state")
	}
	log.beforeAppend = nil
	rebuilt, err := RebuildRuntimeFromCheckpointAndJournal(log, initial)
	compositionCheck(t, err)
	if !bytes.Equal(before, continuationBytes(t, rebuilt)) {
		t.Fatal("recovery after uncertain append diverged")
	}
	if _, err := r.ReadComposition(e.Input, e.After.ObservedAt); !errors.Is(err, ErrRuntimeRecoveryRequired) {
		t.Fatal("retired preview was allowed")
	}
}

func TestNativeCompositionCheckpointTamperingAndStalePlan(t *testing.T) {
	r, _, _ := newCompositionTestRuntime(t, 7, "separated", false)
	path := filepath.Join(t.TempDir(), "state.json")
	compositionCheck(t, r.Checkpoint(path))
	data, err := os.ReadFile(path)
	compositionCheck(t, err)
	var s BrainSnapshot
	compositionCheck(t, json.Unmarshal(data, &s))
	s.Composition.Config.Seed++
	bad, err := json.Marshal(s)
	compositionCheck(t, err)
	compositionCheck(t, os.WriteFile(path, bad, 0600))
	fresh := NewBrainRuntime(knowledge.NewBrain())
	before := continuationBytes(t, fresh)
	if err := fresh.RestoreCheckpoint(path); err == nil {
		t.Fatal("tampered configuration accepted")
	}
	if !bytes.Equal(before, continuationBytes(t, fresh)) {
		t.Fatal("invalid checkpoint mutated restore target")
	}
	s.SchemaVersion = 13
	s.Checksum, err = checkpointChecksumForSchema(s)
	compositionCheck(t, err)
	bad, _ = json.Marshal(s)
	compositionCheck(t, os.WriteFile(path, bad, 0600))
	if err := fresh.RestoreCheckpoint(path); err == nil {
		t.Fatal("composition smuggled into old schema")
	}
	legacy := NewBrainRuntime(knowledge.NewBrain())
	if _, err := legacy.PlanNeuralComposition(knowledge.DefaultCompositionConfig()); !errors.Is(err, ErrCompositionRequiresWrittenMigration) {
		t.Fatal("numeric written migration was bypassed")
	}
	status, err := r.CompositionStatus()
	compositionCheck(t, err)
	status.Config.Seed++
	status.Config.Version = "edited"
	original, err := r.CompositionStatus()
	compositionCheck(t, err)
	if original.Config.Version != knowledge.NeuralCompositionV1 {
		t.Fatal("status exposed a mutable live alias")
	}
	if strings.Contains(string(data), "televisi") {
		t.Fatal("untrained checkpoint contains semantic labels")
	}
	preflight := NewBrainRuntime(knowledge.NewBrain())
	preflight.SetClock(NewFixedClock(compositionTestBase))
	written, err := preflight.PlanNeuralWrittenGrounding()
	compositionCheck(t, err)
	dir := t.TempDir()
	compositionCheck(t, preflight.ApplyNeuralWrittenGrounding(written, filepath.Join(dir, "written.json"), compositionTestBase))
	plan, err := preflight.PlanNeuralComposition(knowledge.DefaultCompositionConfig())
	compositionCheck(t, err)
	_, err = preflight.CognitiveThink(1)
	compositionCheck(t, err)
	beforePlan := continuationBytes(t, preflight)
	backup := filepath.Join(dir, "must-not-exist.json")
	if err := preflight.ApplyNeuralComposition(plan, backup, compositionTestBase); err == nil {
		t.Fatal("stale composition plan accepted")
	}
	if !bytes.Equal(beforePlan, continuationBytes(t, preflight)) {
		t.Fatal("stale migration changed the Brain")
	}
	if _, err := os.Stat(backup); !os.IsNotExist(err) {
		t.Fatal("stale plan created a backup before validation")
	}
}
