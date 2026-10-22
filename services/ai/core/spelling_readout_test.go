package core

import (
	"bytes"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
	"github.com/project-horizon/horizon-core/services/ai/perception"
	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

func TestManagedSpellingUsesSelectedForecastWithoutLabelsAndAfterRecovery(t *testing.T) {
	at := time.Date(2026, 10, 21, 6, 0, 0, 0, time.UTC)
	brain := knowledge.NewBrain()
	r := runtime.NewBrainRuntime(brain)
	r.SetClock(runtime.NewFixedClock(at))
	owner, err := runtime.NewRuntimeSupervisor(r)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	log, err := runtime.OpenEventLog(filepath.Join(t.TempDir(), "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	if err := owner.AttachEventLog(log); err != nil {
		t.Fatal(err)
	}
	var target knowledge.NodeID
	err = owner.WithMutation(func(active *runtime.BrainRuntime) error {
		plan, err := active.PlanNeuralWrittenGrounding()
		if err != nil {
			return err
		}
		if err := active.ApplyNeuralWrittenGrounding(plan, filepath.Join(t.TempDir(), "backup.json"), at); err != nil {
			return err
		}
		for i := 0; i < 4; i++ {
			if _, err := active.LearnExperience(learning.Experience{Sequence: []string{"naku", "melo", "tiva"}, Weight: .8, Confidence: .8}, at); err != nil {
				return err
			}
		}
		inspection, err := active.InspectBrain()
		if err != nil {
			return err
		}
		var found bool
		target, found = inspection.Graph.NodeForSurface("tiva")
		if !found {
			t.Fatal("trained target missing")
		}
		// A deliberately different Unicode realization proves that the surface
		// annotation "tiva" cannot itself supply the decoded output.
		e, err := knowledge.NewSpellingExposure([]knowledge.NodeID{target}, "Tíva🌌", .8, .8)
		if err != nil {
			return err
		}
		for i := 0; i < 4; i++ {
			if _, err := active.LearnSpelling(e, at); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	cp := filepath.Join(t.TempDir(), "trained-codec.json")
	if err := owner.WithMutation(func(active *runtime.BrainRuntime) error { return active.Checkpoint(cp) }); err != nil {
		t.Fatal(err)
	}
	var reference runtime.CognitiveInterpretation
	var referenceSpelling knowledge.SpellingReadout
	var referenceCanonical []byte
	for _, mode := range []string{"normal", "hidden", "forged"} {
		t.Run(mode, func(t *testing.T) {
			copyBrain := knowledge.NewBrain()
			copyRuntime := runtime.NewBrainRuntime(copyBrain)
			if err := copyRuntime.RestoreCheckpoint(cp); err != nil {
				t.Fatal(err)
			}
			annotations := copyBrain.SurfaceAnnotationsSnapshot()
			if mode == "hidden" {
				annotations = nil
			} else if mode == "forged" {
				for i := range annotations {
					annotations[i].Surface = "forged"
				}
			}
			if err := copyBrain.RestoreSurfaceAnnotations(annotations); err != nil {
				t.Fatal(err)
			}
			copyOwner, err := runtime.NewRuntimeSupervisor(copyRuntime)
			if err != nil {
				t.Fatal(err)
			}
			defer copyOwner.Close()
			h := &HorizonEngine{supervisor: copyOwner, Perception: fixedPerception{signal: perception.PerceptionSignal{Source: "test", Modality: "text", Tokens: []string{"naku", "melo"}, ObservedAt: at.Add(time.Second)}}}
			result, spelling, err := h.PulseWithSpelling("naku melo")
			if err != nil || spelling.Status != knowledge.SpellingDecoded || spelling.Text != "Tíva🌌" || result.Answer.Status != runtime.AnswerNeuralForecast || !slices.Equal(result.Answer.NodeIDs, []knowledge.NodeID{target}) {
				t.Fatalf("facade did not decode its native selected forecast: %+v %+v %v", result.Answer, spelling, err)
			}
			canonical := inspectForTest(t, h).Graph.CanonicalBytes()
			if mode == "normal" {
				reference, referenceSpelling, referenceCanonical = result, spelling, canonical
			} else if !reflect.DeepEqual(result.State, reference.State) || !reflect.DeepEqual(result.Answer, reference.Answer) || !reflect.DeepEqual(spelling, referenceSpelling) || !bytes.Equal(canonical, referenceCanonical) {
				t.Fatal("adapter labels changed the forecast, spelling or neural continuation")
			}
		})
	}
	h := &HorizonEngine{supervisor: owner, Perception: fixedPerception{signal: perception.PerceptionSignal{Source: "test", Modality: "text", Tokens: []string{"naku", "melo"}, ObservedAt: at.Add(time.Second)}}}
	_, spelling, err := h.PulseWithSpelling("naku melo")
	if err != nil {
		t.Fatal(err)
	}
	wantCanonical := inspectForTest(t, h).Graph.CanonicalBytes()
	if err := owner.RecoverFromCheckpoint(log, cp); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(wantCanonical, inspectForTest(t, h).Graph.CanonicalBytes()) {
		t.Fatal("owner recovery changed the codec/cognitive state")
	}
	if err := owner.With(func(active *runtime.BrainRuntime) error {
		got, err := active.ReadLearnedSpelling([]knowledge.NodeID{target})
		if err == nil && !reflect.DeepEqual(got, spelling) {
			t.Fatal("owner recovery lost learned spelling weights")
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
}

func TestManagedSpellingDoesNotUseUntrainedActivityAsAnswer(t *testing.T) {
	h := NewHorizonEngine()
	defer h.Supervisor().Close()
	result, spelling, err := h.PulseWithSpelling("naku melo")
	if err != nil || len(result.State.ActiveNodeIDs) == 0 || result.Answer.Status != runtime.AnswerUninterpreted || spelling.Status != knowledge.SpellingUntrained || spelling.Text != "" {
		t.Fatal("untrained activity was expressed as an answer")
	}
}
