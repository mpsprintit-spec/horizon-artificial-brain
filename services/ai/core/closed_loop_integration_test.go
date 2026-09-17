package core

import (
	"sync"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/plugin"
	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

type closedLoopTestPlugin struct {
	mu       sync.Mutex
	triggers []string
}

func (p *closedLoopTestPlugin) Name() string { return "closed-loop-test" }
func (p *closedLoopTestPlugin) Trigger(data string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.triggers = append(p.triggers, data)
}
func (p *closedLoopTestPlugin) Count() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.triggers)
}

var _ plugin.Plugin = (*closedLoopTestPlugin)(nil)

func TestP15RecommendationAuthorizationActionOutcomePromotionLoop(t *testing.T) {
	now := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	h := NewHorizonEngine()
	// Keep the safety clock deterministic so the test validates the closed
	// loop rather than depending on wall-clock time. Production safety still
	// uses the real UTC clock when Safety.Now is nil.
	h.Safety.Now = func() time.Time { return now }

	p := &closedLoopTestPlugin{}
	h.Gateway.RegisterPlugin("complete-task", p)

	completed := h.Knowledge.Store("completed")
	beforeImportance := completed.Importance
	beforePlasticity := completed.Plasticity

	const requestID = "p1-5-closed-loop"
	if err := h.Runtime.RegisterActionBinding(runtime.ActionBinding{
		RequestID:     requestID,
		BrainIdentity: runtime.BrainIdentity,
		Intent:        "complete-task",
		TargetNodeIDs: []knowledge.NodeID{completed.ID},
	}); err != nil {
		t.Fatalf("register action binding: %v", err)
	}

	recommendation := runtime.Recommendation{
		RequestID:     requestID,
		BrainIdentity: runtime.BrainIdentity,
		Intent:        "complete-task",
		Confidence:    0.95,
		RiskLevel:     runtime.RiskLow,
		CreatedAt:     now,
	}

	if err := h.RequestAction(recommendation, "", now.Add(time.Minute)); err != nil {
		t.Fatalf("request action: %v", err)
	}
	if got := p.Count(); got != 1 {
		t.Fatalf("authorized action trigger count: got %d want 1", got)
	}

	firstOutcome := runtime.OutcomeEvent{
		RequestID:     requestID,
		BrainIdentity: runtime.BrainIdentity,
		Success:       true,
		Observation:   []string{"task completed"},
		Source:        "executor",
		Modality:      "execution-outcome",
		ObservedAt:    now.Add(2 * time.Minute),
		Reliability:   1,
	}
	if _, err := h.Runtime.ObserveOutcome(firstOutcome); err != nil {
		t.Fatalf("first outcome: %v", err)
	}
	if completed.Importance != beforeImportance {
		t.Fatalf("first independent outcome promoted node: got importance %v want %v", completed.Importance, beforeImportance)
	}

	secondOutcome := firstOutcome
	secondOutcome.Source = "sensor-verification"
	secondOutcome.ObservedAt = now.Add(3 * time.Minute)
	secondOutcome.Observation = []string{"sensor independently verified completion"}
	if _, err := h.Runtime.ObserveOutcome(secondOutcome); err != nil {
		t.Fatalf("second outcome: %v", err)
	}

	if completed.Importance <= beforeImportance {
		t.Fatalf("promotion did not change neural state: importance got %v want > %v", completed.Importance, beforeImportance)
	}
	if completed.Plasticity <= beforePlasticity {
		t.Fatalf("promotion did not change plasticity: got %v want > %v", completed.Plasticity, beforePlasticity)
	}

	cognitive, err := h.Runtime.CognitiveProcess(runtime.Event{
		Stimulus:  []string{"completed"},
		Source:    "closed-loop-test",
		Modality:  "symbolic",
		Cycles:    1,
		Timestamp: now.Add(4 * time.Minute),
	})
	if err != nil {
		t.Fatalf("next cognitive process: %v", err)
	}
	if cognitive.BrainIdentity != runtime.BrainIdentity {
		t.Fatalf("brain identity: got %q want %q", cognitive.BrainIdentity, runtime.BrainIdentity)
	}
	if activation := cognitive.Activations[completed.ID]; activation <= 0 {
		t.Fatalf("promoted target was not active in next cognitive process: got %v", activation)
	}
}
