package runtime

import (
	"testing"
	"time"
)

func TestCognitiveOrchestratorUsesSingleRuntimeBrain(t *testing.T) {
	brainRuntime := NewBrainRuntime(nil)
	orchestrator := NewCognitiveOrchestrator(brainRuntime)
	if orchestrator == nil || orchestrator.Runtime != brainRuntime {
		t.Fatal("orchestrator must reference the supplied brain runtime")
	}

	event := Event{
		ID:        "test-observation-1",
		Stimulus:  []string{"gelas", "air"},
		Source:    "test",
		Modality:  "text",
		Cycles:    2,
		Timestamp: time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC),
	}
	observation := ObservationInput{
		Source:   "test",
		Modality: "text",
		Tokens:   []string{"gelas", "air"},
	}

	cognitive, learned, err := orchestrator.ProcessObservation(event, observation, nil)
	if err != nil {
		t.Fatalf("ProcessObservation failed: %v", err)
	}
	if learned {
		t.Fatal("no experience was supplied; learned must be false")
	}
	if cognitive.Recommendation != nil {
		t.Fatal("neural interpretation must not create an action recommendation")
	}
	if cognitive.Observation.Source != "test" || cognitive.Observation.Modality != "text" {
		t.Fatal("observation metadata was not preserved")
	}
	if cognitive.State.BrainIdentity != BrainIdentity {
		t.Fatalf("unexpected brain identity: %q", cognitive.State.BrainIdentity)
	}
}

func TestCognitiveOrchestratorOutcomeRequiresRequestIdentity(t *testing.T) {
	orchestrator := NewCognitiveOrchestrator(NewBrainRuntime(nil))
	_, err := orchestrator.LearnFromOutcome(OutcomeEvent{
		BrainIdentity: BrainIdentity,
		Observation:   []string{"success"},
	})
	if err == nil {
		t.Fatal("outcome without RequestID must be rejected")
	}
}

func TestNewExperienceFromObservationCopiesSequence(t *testing.T) {
	event := Event{ID: "experience-1", Timestamp: time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC)}
	observation := ObservationInput{Source: "sensor", Modality: "vision"}
	sequence := []string{"shape", "motion"}
	experience := NewExperienceFromObservation(event, observation, sequence)

	sequence[0] = "mutated"
	if experience.Sequence[0] != "shape" {
		t.Fatal("experience must own a copy of the sequence")
	}
	if experience.ExperienceID != event.ID || experience.Source != "sensor" || experience.Modality != "vision" {
		t.Fatal("experience metadata was not propagated")
	}
}
