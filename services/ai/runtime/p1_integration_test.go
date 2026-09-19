package runtime

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestP1CognitiveIntegrationGroundLearnThinkInterpret(t *testing.T) {
	brain := knowledge.NewBrain()
	runtime := NewBrainRuntime(brain)
	orchestrator := NewCognitiveOrchestrator(runtime)
	at := time.Date(2026, 9, 19, 13, 0, 0, 0, time.UTC)

	observation := ObservationInput{
		Source:        "vision-sensor",
		Modality:      "vision",
		Tokens:        []string{"gelas", "air"},
		ContextTokens: []string{"meja"},
		DataTokens:    []string{"jarak:0.8"},
	}
	event := Event{
		ID:        "p1-integration-1",
		Stimulus:  observation.Tokens,
		Cycles:    2,
		Timestamp: at,
	}
	experience := NewExperienceFromObservation(event, observation, []string{"gelas", "air"})

	interpreted, learned, err := orchestrator.ProcessObservation(event, observation, &experience)
	if err != nil {
		t.Fatalf("ProcessObservation: %v", err)
	}
	if !learned {
		t.Fatal("expected observation experience to be learned")
	}
	if interpreted.State.BrainIdentity != BrainIdentity {
		t.Fatalf("brain identity = %q, want %q", interpreted.State.BrainIdentity, BrainIdentity)
	}
	if len(interpreted.GroundedRepresentations) != 2 {
		t.Fatalf("grounded representations = %d, want 2", len(interpreted.GroundedRepresentations))
	}
	if interpreted.GroundedRepresentations[0].NodeID == 0 || interpreted.GroundedRepresentations[1].NodeID == 0 {
		t.Fatal("grounding returned invalid canonical node IDs")
	}
	if interpreted.GroundedRepresentations[0].Source != "vision-sensor" {
		t.Fatalf("grounding source = %q, want vision-sensor", interpreted.GroundedRepresentations[0].Source)
	}
	if interpreted.Recommendation != nil {
		t.Fatal("neural interpretation must not bypass the action-safety boundary")
	}

	before := runtime.LastSequence()
	thought, err := runtime.CognitiveThink(2)
	if err != nil {
		t.Fatalf("CognitiveThink: %v", err)
	}
	if thought.BrainIdentity != BrainIdentity {
		t.Fatalf("thought brain identity = %q, want %q", thought.BrainIdentity, BrainIdentity)
	}
	if thought.Sequence <= before {
		t.Fatalf("thought sequence = %d, want greater than %d", thought.Sequence, before)
	}
	if len(thought.RankedNodeIDs) == 0 {
		t.Fatal("thought produced no ranked neural nodes")
	}

	repeated, _, err := orchestrator.ProcessObservation(Event{
		ID: "p1-integration-2", Stimulus: observation.Tokens, Cycles: 1, Timestamp: at.Add(time.Second),
	}, observation, nil)
	if err != nil {
		t.Fatalf("repeated ProcessObservation: %v", err)
	}
	if repeated.GroundedRepresentations[0].NodeID != interpreted.GroundedRepresentations[0].NodeID {
		t.Fatal("repeated grounding did not reuse the canonical representation")
	}
	if repeated.GroundedRepresentations[0].Status != "existing" {
		t.Fatalf("repeated grounding status = %q, want existing", repeated.GroundedRepresentations[0].Status)
	}
}

func TestP1ActionOutcomeLearningRequiresSafetyAndIndependentEvidence(t *testing.T) {
	brain := knowledge.NewBrain()
	runtime := NewBrainRuntime(brain)

	source := brain.Store("action-source")
	target := brain.Store("action-target")
	brain.Connect(source, target, 0.50, 0.50, false)

	const requestID = "p1-action-1"
	if err := runtime.RegisterActionBinding(ActionBinding{
		RequestID:     requestID,
		BrainIdentity: BrainIdentity,
		Intent:        "observe-target",
		TargetNodeIDs: []knowledge.NodeID{target.ID},
		Synapses: []SynapseBinding{{
			SourceNodeID: source.ID,
			TargetNodeID: target.ID,
			Inhibitory:   false,
		}},
	}); err != nil {
		t.Fatalf("RegisterActionBinding: %v", err)
	}

	now := time.Date(2026, 9, 19, 13, 1, 0, 0, time.UTC)
	safety := SafetyBoundary{Now: func() time.Time { return now }}
	recommendation := Recommendation{
		RequestID:     requestID,
		BrainIdentity: BrainIdentity,
		Intent:        "observe-target",
		Confidence:    0.99,
		RiskLevel:     RiskLow,
		CreatedAt:     now,
		Reversibility: true,
	}
	assessment, err := safety.Assess(recommendation)
	if err != nil {
		t.Fatalf("Assess: %v", err)
	}
	authorization, err := safety.Authorize(recommendation, assessment, "")
	if err != nil {
		t.Fatalf("Authorize: %v", err)
	}
	if !authorization.Authorized {
		t.Fatal("low-risk action should be policy-authorized without human approval")
	}
	execution, err := safety.BuildExecutionRequest(recommendation, assessment, authorization, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("BuildExecutionRequest: %v", err)
	}
	if !execution.Authorized() {
		t.Fatal("execution request lacks safety capability marker")
	}

	before := brain.Registry.GetByID(target.ID)
	beforeImportance := before.Importance
	beforeSynapse := source.FindDynamicSynapse(target.ID, false)
	if beforeSynapse == nil {
		t.Fatal("dynamic action synapse is missing")
	}
	beforeWeight := beforeSynapse.Weight

	for _, sourceName := range []string{"executor-a", "executor-b"} {
		if _, err := runtime.ObserveOutcome(OutcomeEvent{
			RequestID:     execution.RequestID,
			BrainIdentity: BrainIdentity,
			Success:       true,
			Observation:   []string{"target-observed"},
			Source:        sourceName,
			Modality:      "execution-outcome",
			ObservedAt:    now.Add(10 * time.Second),
			Reliability:   1.0,
		}); err != nil {
			t.Fatalf("ObserveOutcome(%s): %v", sourceName, err)
		}
	}

	after := brain.Registry.GetByID(target.ID)
	afterSynapse := source.FindDynamicSynapse(target.ID, false)
	if after.Importance <= beforeImportance {
		t.Fatalf("target importance = %v, want > %v after accepted evidence", after.Importance, beforeImportance)
	}
	if afterSynapse.Weight <= beforeWeight {
		t.Fatalf("synapse weight = %v, want > %v after accepted evidence", afterSynapse.Weight, beforeWeight)
	}
}
