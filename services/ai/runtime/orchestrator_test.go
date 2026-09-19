package runtime

import (
	"testing"
	"time"
)

func TestCognitiveOrchestratorUsesSingleRuntimeBrain(t *testing.T) {
	brainRuntime := NewBrainRuntime(nil)
	orchestrator := NewCognitiveOrchestrator(brainRuntime)
	if orchestrator == nil || orchestrator.Runtime != brainRuntime { t.Fatal("orchestrator must reference the supplied brain runtime") }
	event := Event{ID: "test-observation-1", Stimulus: []string{"gelas", "air"}, Source: "test", Modality: "text", Cycles: 2, Timestamp: time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC)}
	observation := ObservationInput{Source: "test", Modality: "text", Tokens: []string{"gelas", "air"}}
	cognitive, learned, err := orchestrator.ProcessObservation(event, observation, nil)
	if err != nil { t.Fatalf("ProcessObservation failed: %v", err) }
	if learned { t.Fatal("no experience was supplied; learned must be false") }
	if cognitive.Recommendation != nil { t.Fatal("neural interpretation must not create an action recommendation") }
	if cognitive.Observation.Source != "test" || cognitive.Observation.Modality != "text" { t.Fatal("observation metadata was not preserved") }
	if cognitive.State.BrainIdentity != BrainIdentity { t.Fatalf("unexpected brain identity: %q", cognitive.State.BrainIdentity) }
}

func TestCognitiveOrchestratorGroundsContextAndDataIntoSharedSubstrate(t *testing.T) {
	orchestrator := NewCognitiveOrchestrator(NewBrainRuntime(nil))
	observation := ObservationInput{Source: "vision-sensor", Modality: "vision", Tokens: []string{"gelas"}, ContextTokens: []string{"meja"}, DataTokens: []string{"jarak:0.8"}}
	cognitive, _, err := orchestrator.ProcessObservation(Event{ID: "grounding-1", Stimulus: []string{"gelas"}, Cycles: 1, Timestamp: time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC)}, observation, nil)
	if err != nil { t.Fatalf("ProcessObservation grounding failed: %v", err) }
	if len(cognitive.GroundedRepresentations) != 2 { t.Fatalf("expected 2 grounded representations, got %d", len(cognitive.GroundedRepresentations)) }
	if cognitive.GroundedRepresentations[0].Status != "candidate" { t.Fatalf("first unseen representation should be candidate, got %q", cognitive.GroundedRepresentations[0].Status) }
	if cognitive.GroundedRepresentations[0].Source != "vision-sensor" || cognitive.GroundedRepresentations[0].Modality != "vision" { t.Fatal("grounding provenance was not preserved") }
	if cognitive.GroundedRepresentations[0].NodeID == 0 || cognitive.GroundedRepresentations[1].NodeID == 0 { t.Fatal("grounding must produce valid substrate node IDs") }
}

func TestCognitiveOrchestratorRepeatedGroundingReusesRepresentation(t *testing.T) {
	orchestrator := NewCognitiveOrchestrator(NewBrainRuntime(nil))
	observation := ObservationInput{Source: "sensor", Modality: "vision", ContextTokens: []string{"meja"}}
	first, _, err := orchestrator.ProcessObservation(Event{ID: "grounding-a", Cycles: 1}, observation, nil)
	if err != nil { t.Fatalf("first observation failed: %v", err) }
	second, _, err := orchestrator.ProcessObservation(Event{ID: "grounding-b", Cycles: 1}, observation, nil)
	if err != nil { t.Fatalf("second observation failed: %v", err) }
	if len(first.GroundedRepresentations) != 1 || len(second.GroundedRepresentations) != 1 { t.Fatal("expected one grounding record per observation") }
	if first.GroundedRepresentations[0].NodeID != second.GroundedRepresentations[0].NodeID { t.Fatal("identical grounding input must reuse the same neural representation") }
	if second.GroundedRepresentations[0].Status != "existing" { t.Fatalf("reused representation should be existing, got %q", second.GroundedRepresentations[0].Status) }
}

func TestCognitiveOrchestratorOutcomeRequiresRequestIdentity(t *testing.T) {
	orchestrator := NewCognitiveOrchestrator(NewBrainRuntime(nil))
	_, err := orchestrator.LearnFromOutcome(OutcomeEvent{BrainIdentity: BrainIdentity, Observation: []string{"success"}})
	if err == nil { t.Fatal("outcome without RequestID must be rejected") }
}

func TestNewExperienceFromObservationCopiesSequence(t *testing.T) {
	event := Event{ID: "experience-1", Timestamp: time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC)}
	observation := ObservationInput{Source: "sensor", Modality: "vision"}
	sequence := []string{"shape", "motion"}
	experience := NewExperienceFromObservation(event, observation, sequence)
	sequence[0] = "mutated"
	if experience.Sequence[0] != "shape" { t.Fatal("experience must own a copy of the sequence") }
	if experience.ExperienceID != event.ID || experience.Source != "sensor" || experience.Modality != "vision" { t.Fatal("experience metadata was not propagated") }
}

func TestCognitiveOrchestratorDataGroundingUsesEventTime(t *testing.T) {
	orchestrator := NewCognitiveOrchestrator(NewBrainRuntime(nil))
	at := time.Date(2026, 9, 16, 1, 2, 3, 0, time.UTC)
	observation := ObservationInput{Source: "sensor", Modality: "vision", DataTokens: []string{"jarak:0.8"}}
	cognitive, _, err := orchestrator.ProcessObservation(Event{ID: "grounding-time", Cycles: 1, Timestamp: at}, observation, nil)
	if err != nil { t.Fatalf("ProcessObservation failed: %v", err) }
	if len(cognitive.GroundedRepresentations) != 1 { t.Fatalf("expected one grounding, got %d", len(cognitive.GroundedRepresentations)) }
	node := orchestrator.Runtime.brain.Registry.GetByID(cognitive.GroundedRepresentations[0].NodeID)
	if node == nil { t.Fatal("grounded node is missing") }
	if !node.LastActivation.Equal(at) { t.Fatalf("data grounding timestamp = %v, want %v", node.LastActivation, at) }
}
