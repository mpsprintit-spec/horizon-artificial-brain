package core

import (
	"context"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/learning"
	"github.com/project-horizon/horizon-core/services/ai/websearch"
)

func TestP16KnowledgeGapInquiryGroundingValidationLearningLoop(t *testing.T) {
	now := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	h := NewHorizonEngine()

	// 1. The unknown observation is a knowledge gap, not trusted knowledge.
	grounded, err := h.Runtime.GroundObservation("burung-unta", "sensor", "symbolic")
	if err != nil {
		t.Fatalf("ground unknown observation: %v", err)
	}
	if grounded.Status != "candidate" {
		t.Fatalf("expected candidate grounding status, got %q", grounded.Status)
	}
	if !h.WebSearch.ShouldSearch(0.20, 1, 0) {
		t.Fatal("knowledge gap should trigger inquiry")
	}

	// 2. Inquiry obtains attributable external observations.
	h.WebSearch = websearch.NewEngine(websearch.StaticSearcher{Results: []websearch.SourceResult{
		{Source: "source-a", Reputation: 0.90, Confidence: 0.80, Tokens: []string{"burung-unta", "tidak", "terbang"}},
		{Source: "source-b", Reputation: 0.85, Confidence: 0.75, Tokens: []string{"burung-unta", "tidak", "terbang"}},
	}})
	results, err := h.WebSearch.Perceive(context.Background(), "burung-unta terbang")
	if err != nil {
		t.Fatalf("inquiry: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected two independent inquiry results, got %d", len(results))
	}

	// 3. Ground the response into the same Brain substrate.
	responseGrounding, err := h.Runtime.GroundObservation("tidak", "source-a", "inquiry")
	if err != nil {
		t.Fatalf("ground inquiry response: %v", err)
	}
	if responseGrounding.NodeID == 0 {
		t.Fatal("expected grounded response node")
	}
	if responseGrounding.Source != "source-a" || responseGrounding.Modality != "inquiry" {
		t.Fatalf("grounding provenance lost: %+v", responseGrounding)
	}

	// 4. Build a candidate without learning it into Patterns yet.
	candidate, err := learning.BuildInquiryCandidate(
		[]string{"burung-unta", "tidak", "terbang"},
		results,
		"inquiry",
		now,
	)
	if err != nil {
		t.Fatalf("build inquiry candidate: %v", err)
	}
	if got := learning.ValidateInquiry(candidate, learning.DefaultLearningPolicy()); got != learning.PromotionAccepted {
		t.Fatalf("expected independent evidence to validate candidate, got %q", got)
	}

	beforePatterns := len(h.Knowledge.Patterns.All())

	// 5. A candidate is not learned until validation has accepted it.
	weakResults := []websearch.SourceResult{{Source: "single-source", Reputation: 0.90, Confidence: 0.80}}
	weakCandidate, err := learning.BuildInquiryCandidate([]string{"candidate-only"}, weakResults, "inquiry", now)
	if err != nil {
		t.Fatalf("build weak candidate: %v", err)
	}
	if got := learning.ValidateInquiry(weakCandidate, learning.DefaultLearningPolicy()); got != learning.PromotionCandidate {
		t.Fatalf("single-source candidate must remain candidate, got %q", got)
	}
	if got := len(h.Knowledge.Patterns.All()); got != beforePatterns {
		t.Fatalf("unvalidated candidate mutated patterns: before=%d after=%d", beforePatterns, got)
	}

	// 6. Only the validated candidate enters the existing learning path.
	candidate.Experience.ExperienceID = "p1-6-inquiry"
	candidate.Experience.IndependenceGroup = "external-inquiry"
	if _, err := h.Runtime.LearnExperience(candidate.Experience, now); err != nil {
		t.Fatalf("learn validated inquiry candidate: %v", err)
	}
	if got := len(h.Knowledge.Patterns.All()); got <= beforePatterns {
		t.Fatalf("validated candidate did not enter neural learning path: before=%d after=%d", beforePatterns, got)
	}

	// 7. The learned representation is available to subsequent cognition in
	// the same primary Brain rather than a separate inquiry memory.
	cognitive, err := h.Runtime.CognitiveProcess(runtimeEventForP16(now))
	if err != nil {
		t.Fatalf("subsequent cognition: %v", err)
	}
	if cognitive.BrainIdentity != "horizon-primary-brain" {
		t.Fatalf("brain identity changed across loop: %q", cognitive.BrainIdentity)
	}
	if len(cognitive.RankedNodeIDs) == 0 {
		t.Fatal("subsequent cognition did not activate learned substrate")
	}
}
