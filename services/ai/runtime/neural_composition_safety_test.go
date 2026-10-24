package runtime

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

func TestNativeCompositionAmbiguousStaleAndUnknownAreReadOnly(t *testing.T) {
	r, _, _ := newCompositionTestRuntime(t, 7, "separated", false)
	at := compositionTrain(t, r, 7, 1, false)
	input := nativeCompositionInput(bindWorldNew(7, 0, 0).input([]string{"nyalakan", "lampu"}, -1), at)
	ready, err := r.ReadComposition(input, at)
	compositionCheck(t, err)
	if !ready.Accepted {
		t.Fatal("known command did not establish a valid reference")
	}
	before := continuationBytes(t, r)
	for _, kind := range []string{"duplicate-object", "stale", "unknown-word", "empty-word", "no-patches", "too-many-words"} {
		t.Run(kind, func(t *testing.T) {
			bad, _ := freezeComposition(input)
			now := at
			switch kind {
			case "duplicate-object":
				var selected []float64
				for _, p := range bad.Scene.Patches {
					if slices.Equal(p.XY, ready.Motor[:2]) {
						selected = p.Values
					}
				}
				for i := range bad.Scene.Patches {
					if !slices.Equal(bad.Scene.Patches[i].XY, ready.Motor[:2]) {
						bad.Scene.Patches[i].Values = slices.Clone(selected)
						break
					}
				}
			case "stale":
				now = at.Add(3 * time.Second)
			case "unknown-word":
				bad.Words[0] = knowledge.EncodeObservation("never-taught-form", "language").Values
			case "empty-word":
				bad.Words = nil
			case "no-patches":
				bad.Scene.Patches = nil
			case "too-many-words":
				for len(bad.Words) <= knowledge.MaxCompositionWords {
					bad.Words = append(bad.Words, bad.Words[0])
				}
			}
			out, err := r.ReadComposition(bad, now)
			if err == nil && out.Accepted {
				t.Fatal("unsupported or ambiguous input was accepted")
			}
			if !bytes.Equal(before, continuationBytes(t, r)) {
				t.Fatal("read preview mutated a live state")
			}
		})
	}
}

func TestNativeCompositionOrdinaryObservationsAndThoughtRetainSkill(t *testing.T) {
	r, _, _ := newCompositionTestRuntime(t, 7, "separated", false)
	at := compositionTrain(t, r, 7, 6, false)
	scene := nativeCompositionInput(bindWorldNew(7, 2, 4).input(nil, -1), at).Scene
	_, err := r.ObserveCompositionScene("scene", "simulator", scene)
	compositionCheck(t, err)
	for i := 0; i < 24; i++ {
		out, _, err := NewCognitiveOrchestrator(r).ProcessObservation(Event{ID: fmt.Sprintf("pulse-%d", i), Timestamp: at.Add(time.Duration(i+1) * time.Millisecond), Cycles: 1}, ObservationInput{Source: "user", Modality: "language", Tokens: []string{"nyalakan", "televisi"}}, nil)
		compositionCheck(t, err)
		if out.Composition == nil || !out.Composition.Accepted || out.Composition.Status != "proposal" {
			t.Fatalf("native observation lost learned composition on step %d: %+v", i, out.Composition)
		}
		if out.Recommendation != nil {
			t.Fatal("composition preview authorized an action")
		}
		r.SetClock(NewFixedClock(at.Add(time.Duration(i+1) * time.Millisecond)))
		thought, err := r.CognitiveThink(1)
		compositionCheck(t, err)
		if thought.Composition == nil {
			t.Fatal("internal thought lost numerical continuation")
		}
	}
}

func TestNativeCompositionPreservesEarlierOrderedMemoryAndSynapticHistory(t *testing.T) {
	r, _, _ := newCompositionTestRuntime(t, 7, "separated", false)
	for i := 0; i < 8; i++ {
		_, err := r.LearnExperience(learning.Experience{Sequence: []string{"prior-cue", "prior-outcome"}, Weight: .9, Confidence: .9, ExperienceID: fmt.Sprintf("old-%d", i)}, compositionTestBase)
		compositionCheck(t, err)
	}
	cue, err := r.brain.GroundObservationWithEncodingAt("prior-cue", "test", "language", GroundingThreshold, compositionTestBase, knowledge.NeuralWrittenEncoding)
	compositionCheck(t, err)
	trace := &knowledge.ObservationTrace{Stimulus: [][]knowledge.NodeID{cue.Population}}
	prior := r.brain.Patterns.PredictOrdered(trace)
	if len(prior.Support) == 0 {
		t.Fatal("prior native memory was not established")
	}
	_ = compositionTrain(t, r, 7, 2, false)
	after := r.brain.Patterns.PredictOrdered(trace)
	for id, value := range prior.Support {
		if after.Support[id] < .9*value {
			t.Fatal("composition practice erased earlier native ordered knowledge")
		}
	}
	for _, node := range r.brain.Registry.Nodes() {
		for _, edge := range node.OutboundAll() {
			if edge.Dynamic.Confidence >= 1 {
				t.Fatal("numerical practice silently promoted confidence to certainty")
			}
		}
	}
}

func TestNativeCompositionDurableLostAcknowledgmentAndInterruptedApply(t *testing.T) {
	for _, phase := range []string{"lost-ack", "panic-after-learning"} {
		t.Run(phase, func(t *testing.T) {
			r, log, initial := newCompositionTestRuntime(t, 7, "separated", true)
			cases, _ := bindCases()
			at := compositionTestBase.Add(time.Second)
			e := nativeCompositionExposure(bindEpisodeFor(cases[0], 7, 0, cases[0].Words), "interrupted", at)
			beforeRevision := r.LastSequence()
			if phase == "lost-ack" {
				log.beforeAppend = func(record LoggedEvent) error {
					wire, err := json.Marshal(record)
					if err != nil {
						return err
					}
					if _, err = log.file.Write(append(wire, '\n')); err != nil {
						return err
					}
					if err = log.file.Sync(); err != nil {
						return err
					}
					return errors.New("durable append acknowledgment lost")
				}
				if _, err := r.LearnComposition(e, e.After.ObservedAt); !errors.Is(err, ErrRuntimeRecoveryRequired) {
					t.Fatal("uncertain durable command did not retire runtime")
				}
				log.beforeAppend = nil
			} else {
				// Fault injection at the real redo/apply boundary after canonical
				// updates but before dedup/revision publication. Recovery uses the
				// ordinary public teaching method without this injected panic.
				func() {
					defer func() {
						if recover() == nil {
							t.Error("missing injected panic")
						}
					}()
					r.mu.Lock()
					defer r.mu.Unlock()
					_ = r.runJournalCommandLocked(LoggedEvent{Type: EventTypeCompositionExposure, Timestamp: e.After.ObservedAt, CompositionExposure: &e}, func() error {
						compositionCheck(t, r.brain.LearnCompositionAt(r.composition.Config, e, e.After.ObservedAt))
						panic("interrupted compound update")
					})
				}()
			}
			if !r.RecoveryRequired() {
				t.Fatal("interrupted runtime stayed available")
			}
			rebuilt, err := RebuildRuntimeFromCheckpointAndJournal(log, initial)
			compositionCheck(t, err)
			if rebuilt.LastSequence() != beforeRevision+1 {
				t.Fatal("durable command did not recover exactly one revision")
			}
			before := continuationBytes(t, rebuilt)
			if _, err := rebuilt.LearnComposition(e, e.After.ObservedAt); err == nil {
				t.Fatal("acknowledgment retry applied twice")
			}
			if !bytes.Equal(before, continuationBytes(t, rebuilt)) {
				t.Fatal("duplicate retry altered state")
			}
		})
	}
}

func TestNativeCompositionBudgetAndMigrationBackups(t *testing.T) {
	r, log, initial := newCompositionTestRuntime(t, 7, "separated", true)
	// The actual backup pair is readable by the old schema laws and remains
	// anchored to exactly the prefix before composition's first command.
	backup := filepath.Join(filepath.Dir(initial), "old-composition.json")
	old := NewBrainRuntime(knowledge.NewBrain())
	compositionCheck(t, old.RestoreCheckpoint(backup))
	if old.composition != nil || !old.neuralWrittenGrounding {
		t.Fatal("rollback copy changed the old serving contract")
	}
	before := continuationBytes(t, r)
	c := r.composition.Config
	c.MaxLearnedWords = 1
	r.composition.Config = c
	cases, _ := bindCases()
	e := nativeCompositionExposure(bindEpisodeFor(cases[0], 7, 0, cases[0].Words), "over-budget", compositionTestBase.Add(time.Second))
	budgetBefore := continuationBytes(t, r)
	if _, err := r.LearnComposition(e, e.After.ObservedAt); err == nil {
		t.Fatal("new word recruitment exceeded its cap")
	}
	if !bytes.Equal(budgetBefore, continuationBytes(t, r)) {
		t.Fatal("budget refusal mutated state")
	}
	r.composition.Config.MaxLearnedWords = 256
	if !bytes.Equal(before, continuationBytes(t, r)) {
		t.Fatal("budget test did not restore its fixture")
	}
	if _, err := log.RepairConflictingSequences(); err == nil {
		t.Fatal("legacy renumbering accepted versioned composition commands")
	}
	// NaN is rejected before a durable payload or canonical update.
	e.Input.Words[0][0] = math.NaN()
	if _, err := r.LearnComposition(e, e.After.ObservedAt); err == nil {
		t.Fatal("non-finite teaching input accepted")
	}
}

func TestNativeCompositionRepeatedWordSharesGrowthAndHistory(t *testing.T) {
	r, _, _ := newCompositionTestRuntime(t, 7, "separated", false)
	r.composition.Config.MaxLearnedWords = 2
	world := bindWorldNew(7, 0, 0)
	for i := 0; i < 32; i++ {
		motor := world.motor(0, i%2 == 0, false)
		after, _ := world.apply(motor)
		e := nativeCompositionExposure(bindEpisode{Input: world.input(nil, -1), Motor: motor, After: after.patches()}, fmt.Sprintf("calibration-%d", i), compositionTestBase.Add(time.Duration(i+1)*time.Microsecond))
		_, err := r.LearnComposition(e, e.After.ObservedAt)
		compositionCheck(t, err)
	}
	motor := world.motor(0, true, false)
	after, _ := world.apply(motor)
	for i := 0; i < 40; i++ {
		episode := bindEpisode{Input: world.input([]string{"nyalakan", "lampu", "lampu"}, -1), Motor: motor, After: after.patches()}
		e := nativeCompositionExposure(episode, fmt.Sprintf("repeated-%d", i), compositionTestBase.Add(time.Duration(i+1)*time.Millisecond))
		_, err := r.LearnComposition(e, e.After.ObservedAt)
		compositionCheck(t, err)
	}
	count := 0
	eligible := 0
	for _, p := range r.brain.ProjectionPopulations {
		if p.Scope != nil && p.Scope.Field == "written-neural-v2" {
			count++
			for _, member := range p.Units {
				for _, edge := range r.brain.Registry.GetByID(member.NodeID).OutboundAll() {
					if edge.Dynamic.Frequency != 40 || edge.Dynamic.Confidence >= 1 {
						t.Fatal("repeated positions duplicated updates or erased synaptic history")
					}
					if edge.Dynamic.Eligibility > 0 {
						eligible++
					}
				}
			}
		}
	}
	if eligible == 0 {
		t.Fatal("all historical eligibility was erased")
	}
	if count != 2 {
		t.Fatalf("repeated written input grew independent memories: %d", count)
	}
	at := compositionTestBase.Add(time.Second)
	out, err := r.ReadComposition(nativeCompositionInput(world.input([]string{"nyalakan", "lampu", "lampu"}, -1), at), at)
	compositionCheck(t, err)
	if !out.Accepted || !slices.Equal(out.Motor, motor) {
		t.Fatal("repeated-position composition did not learn the demonstrated contact")
	}
}

func BenchmarkNativeCompositionReadout(b *testing.B) {
	r, _, _ := newCompositionTestRuntime(b, 7, "separated", false)
	at := compositionTrain(b, r, 7, 1, false)
	input := nativeCompositionInput(bindWorldNew(7, 2, 3).input([]string{"nyalakan", "lampu"}, -1), at)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := r.ReadComposition(input, at); err != nil {
			b.Fatal(err)
		}
	}
}
