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

func TestManagedContinuationPreviewKeepsNativeAnswerAndCanonicalState(t *testing.T) {
	at := time.Unix(100, 0).UTC()
	r := runtime.NewBrainRuntime(knowledge.NewBrain())
	r.SetClock(runtime.NewFixedClock(at))
	plan, err := r.PlanNeuralWrittenGrounding()
	if err != nil {
		t.Fatal(err)
	}
	if err := r.ApplyNeuralWrittenGrounding(plan, filepath.Join(t.TempDir(), "backup.json"), at); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 6; i++ {
		if _, err := r.LearnExperience(learning.Experience{Sequence: []string{"naku", "melo", "tiva"}, Weight: .9, Confidence: .9}, at); err != nil {
			t.Fatal(err)
		}
	}
	inspection, err := r.InspectBrain()
	if err != nil {
		t.Fatal(err)
	}
	target, found := inspection.Graph.NodeForSurface("tiva")
	if !found {
		t.Fatal("learned forecast missing")
	}
	var pieces [][]knowledge.NodeID
	for i, text := range []string{"Tíva", " 🌌", ""} {
		vector := make([]float64, 32)
		vector[i] = 1
		p, err := r.GroundVectorAt(knowledge.NewNeuralVector(vector), at)
		if err != nil {
			t.Fatal(err)
		}
		pieces = append(pieces, p.Population)
		e, err := knowledge.NewSpellingExposure(p.Population, text, .9, .9)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := r.LearnSpelling(e, at); err != nil {
			t.Fatal(err)
		}
	}
	previous := []knowledge.NodeID{target}
	for _, next := range pieces {
		for i := 0; i < 6; i++ {
			if _, err := r.LearnExperience(learning.Experience{NeuralSequence: [][]knowledge.NodeID{previous, next}, Weight: .9, Confidence: .9}, at); err != nil {
				t.Fatal(err)
			}
		}
		previous = next
	}
	cp := filepath.Join(t.TempDir(), "learned.json")
	if err := r.Checkpoint(cp); err != nil {
		t.Fatal(err)
	}
	var baseline runtime.CognitiveInterpretation
	var canonical []byte
	var preview runtime.ContinuationReadout
	for _, mode := range []string{"pulse", "preview", "hidden", "forged"} {
		brain := knowledge.NewBrain()
		copy := runtime.NewBrainRuntime(brain)
		if err := copy.RestoreCheckpoint(cp); err != nil {
			t.Fatal(err)
		}
		annotations := brain.SurfaceAnnotationsSnapshot()
		if mode == "hidden" {
			annotations = nil
		} else if mode == "forged" {
			for i := range annotations {
				annotations[i].Surface = "forged"
			}
		}
		if err := brain.RestoreSurfaceAnnotations(annotations); err != nil {
			t.Fatal(err)
		}
		owner, err := runtime.NewRuntimeSupervisor(copy)
		if err != nil {
			t.Fatal(err)
		}
		defer owner.Close()
		h := &HorizonEngine{supervisor: owner, Perception: fixedPerception{signal: perception.PerceptionSignal{Source: "test", Modality: "text", Tokens: []string{"naku", "melo"}, ObservedAt: at.Add(time.Second)}}}
		var result runtime.CognitiveInterpretation
		var out runtime.ContinuationReadout
		if mode == "pulse" {
			result, err = h.Pulse("naku melo")
		} else {
			result, out, err = h.PulseWithContinuationPreview("naku melo", 8)
			if err == nil && (out.Status != runtime.ContinuationComplete || out.Text != "Tíva 🌌" || !out.Hypothetical || !slices.Equal(out.Steps[0].Forecast.NodeIDs, pieces[0])) {
				t.Fatalf("facade did not project learned pieces: %+v", out)
			}
		}
		if err != nil || result.Answer.Status != runtime.AnswerNeuralForecast || !slices.Equal(result.Answer.NodeIDs, []knowledge.NodeID{target}) {
			t.Fatalf("facade changed native selection: %+v, %v", result.Answer, err)
		}
		if mode == "pulse" {
			baseline, canonical = result, inspectForTest(t, h).Graph.CanonicalBytes()
		} else if !reflect.DeepEqual(result, baseline) || !bytes.Equal(canonical, inspectForTest(t, h).Graph.CanonicalBytes()) {
			t.Fatal("preview or metadata changed the real pulse/state")
		}
		if mode == "preview" {
			preview = out
		} else if mode != "pulse" && !reflect.DeepEqual(preview, out) {
			t.Fatal("labels changed the learned continuation")
		}
	}
}

func TestManagedContinuationPreviewRejectsBudgetBeforePulseAndKeepsUnknown(t *testing.T) {
	h := NewHorizonEngine()
	defer h.Supervisor().Close()
	before := inspectForTest(t, h).Graph.CanonicalBytes()
	if _, _, err := h.PulseWithContinuationPreview("naku melo", runtime.MaxContinuationSteps+1); err == nil {
		t.Fatal("invalid budget accepted")
	}
	if !bytes.Equal(before, inspectForTest(t, h).Graph.CanonicalBytes()) {
		t.Fatal("invalid budget processed a real input")
	}
	result, out, err := h.PulseWithContinuationPreview("naku melo", 8)
	if err != nil || result.Answer.Status != runtime.AnswerUninterpreted || out.Status != runtime.ContinuationUnknown || out.Text != "" || len(out.Steps) != 0 {
		t.Fatal("generic activity became a language hypothesis")
	}
}

func TestManagedContinuationPreviewKeepsWeakNativeForecastUncertain(t *testing.T) {
	at := time.Unix(100, 0).UTC()
	r := runtime.NewBrainRuntime(knowledge.NewBrain())
	if _, err := r.LearnExperience(learning.Experience{Sequence: []string{"naku", "melo", "tiva"}, Weight: .9, Confidence: .9}, at); err != nil {
		t.Fatal(err)
	}
	owner, err := runtime.NewRuntimeSupervisor(r)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	h := &HorizonEngine{supervisor: owner, Perception: fixedPerception{signal: perception.PerceptionSignal{Source: "test", Modality: "text", Tokens: []string{"naku", "melo"}, ObservedAt: at.Add(time.Second)}}}
	result, out, err := h.PulseWithContinuationPreview("naku melo", 8)
	if err != nil || result.Answer.Status != runtime.AnswerNeuralForecast || result.Answer.Confidence > runtime.ContinuationConfidenceGate || out.Status != runtime.ContinuationUncertain || len(out.Steps) != 0 || out.Text != "" {
		t.Fatalf("weak native selection became fluent output: %+v %+v %v", result.Answer, out, err)
	}
}
