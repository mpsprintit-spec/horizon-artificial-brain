package runtime

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
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
	if len(interpreted.GroundedRepresentations) != 4 {
		t.Fatalf("grounded representations = %d, want 4", len(interpreted.GroundedRepresentations))
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

func TestP1AcceptedInquiryLearnsIntoCanonicalBrainAndIsRecoverable(t *testing.T) {
	brain := knowledge.NewBrain()
	runtime := NewBrainRuntime(brain)
	at := time.Date(2026, 9, 19, 13, 2, 0, 0, time.UTC)

	candidate, err := learning.BuildInquiryCandidate(
		[]string{"matahari", "terbit"},
		[]learning.InquirySource{
			{Source: "source-a", Reputation: 0.95, Confidence: 0.90},
			{Source: "source-b", Reputation: 0.90, Confidence: 0.95},
		},
		"web-inquiry",
		at,
	)
	if err != nil { t.Fatalf("BuildInquiryCandidate: %v", err) }
	decision := learning.ValidateInquiry(candidate, learning.DefaultLearningPolicy())
	if decision != learning.PromotionAccepted { t.Fatalf("inquiry decision = %v, want %v", decision, learning.PromotionAccepted) }
	if !candidate.Experience.Timestamp.Equal(at) { t.Fatalf("candidate timestamp = %v, want %v", candidate.Experience.Timestamp, at) }

	if _, err := runtime.LearnExperience(candidate.Experience, at); err != nil { t.Fatalf("LearnExperience: %v", err) }

	first := brain.Registry.Get("matahari")
	second := brain.Registry.Get("terbit")
	if first == nil || second == nil { t.Fatal("accepted inquiry did not create canonical Brain nodes") }
	if !first.LastActivation.Equal(at) || !second.LastActivation.Equal(at) { t.Fatalf("learned node timestamps = %v, %v; want %v", first.LastActivation, second.LastActivation, at) }

	patterns := brain.Patterns.MatchTrace([]knowledge.PatternStep{
		{NodeID: first.ID, Position: 0, Activation: 1},
		{NodeID: second.ID, Position: 1, Activation: 1},
	}, nil)
	if len(patterns) == 0 { t.Fatal("accepted inquiry did not form a recoverable canonical pattern") }
	pattern := patterns[0]
	if pattern.Result != second.ID { t.Fatalf("pattern result = %d, want %d", pattern.Result, second.ID) }
	if len(pattern.Evidence) != 1 { t.Fatalf("pattern evidence count = %d, want 1", len(pattern.Evidence)) }
	if !pattern.Evidence[0].Timestamp.Equal(at) { t.Fatalf("pattern evidence timestamp = %v, want %v", pattern.Evidence[0].Timestamp, at) }
	if pattern.Evidence[0].Source != "source-a" { t.Fatalf("pattern provenance source = %q, want source-a", pattern.Evidence[0].Source) }

	// Recover the learned inquiry through the same observation boundary used by
	// ordinary perception. Context/data grounding must resolve to the canonical
	// nodes created by the accepted inquiry, while the stimulus is processed by
	// the cognitive runtime.
	recoveryAt := at.Add(time.Second)
	observation := ObservationInput{
		Source: "follow-up-observation",
		Modality: "text",
		Tokens: []string{"matahari", "terbit"},
		ContextTokens: []string{"matahari", "terbit"},
	}
	interpreted, learned, err := NewCognitiveOrchestrator(runtime).ProcessObservation(Event{
		ID: "p1-inquiry-recovery",
		Stimulus: observation.Tokens,
		Timestamp: recoveryAt,
		Cycles: 1,
	}, observation, nil)
	if err != nil { t.Fatalf("ProcessObservation recovery: %v", err) }
	if learned { t.Fatal("recovery observation without an experience should not learn") }
	if len(interpreted.GroundedRepresentations) != 2 {
		t.Fatalf("recovery grounding count = %d, want 2", len(interpreted.GroundedRepresentations))
	}
	if interpreted.GroundedRepresentations[0].NodeID == first.ID ||
		interpreted.GroundedRepresentations[1].NodeID == second.ID {
		t.Fatalf("text surface grounding incorrectly collapsed into language-store identity: got %d,%d; language nodes=%d,%d",
			interpreted.GroundedRepresentations[0].NodeID,
			interpreted.GroundedRepresentations[1].NodeID, first.ID, second.ID)
	}
	for i, representation := range interpreted.GroundedRepresentations {
		if representation.Status != "candidate" {
			t.Fatalf("first text recovery[%d] status = %q, want candidate", i, representation.Status)
		}
		if !representation.Timestamp.Equal(recoveryAt) {
			t.Fatalf("first text recovery[%d] timestamp = %v, want %v", i, representation.Timestamp, recoveryAt)
		}
	}

	secondRecoveryAt := recoveryAt.Add(time.Second)
	recovered, _, err := NewCognitiveOrchestrator(runtime).ProcessObservation(Event{
		ID: "p1-inquiry-recovery-repeat",
		Stimulus: observation.Tokens,
		Timestamp: secondRecoveryAt,
		Cycles: 1,
	}, observation, nil)
	if err != nil { t.Fatalf("ProcessObservation repeated recovery: %v", err) }
	for i, representation := range recovered.GroundedRepresentations {
		if representation.Status != "existing" {
			t.Fatalf("repeated text recovery[%d] status = %q, want existing", i, representation.Status)
		}
		if !representation.Timestamp.Equal(secondRecoveryAt) {
			t.Fatalf("repeated text recovery[%d] timestamp = %v, want %v", i, representation.Timestamp, secondRecoveryAt)
		}
		if representation.NodeID != interpreted.GroundedRepresentations[i].NodeID {
			t.Fatalf("repeated text recovery[%d] changed anchor: got %d want %d", i, representation.NodeID, interpreted.GroundedRepresentations[i].NodeID)
		}
	}

	if interpreted.State.BrainIdentity != BrainIdentity {
		t.Fatalf("recovery brain identity = %q, want %q", interpreted.State.BrainIdentity, BrainIdentity)
	}
	if len(interpreted.State.ActiveNodeIDs) == 0 {
		t.Fatal("recovery interpretation produced no ranked neural nodes")
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
