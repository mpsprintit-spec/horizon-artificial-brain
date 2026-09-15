package execution

import (
	"sync"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

type gate8TestPlugin struct {
	mu       sync.Mutex
	triggers []string
}

func (p *gate8TestPlugin) Name() string { return "gate8-test" }
func (p *gate8TestPlugin) Trigger(data string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.triggers = append(p.triggers, data)
}
func (p *gate8TestPlugin) Count() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.triggers)
}

func buildLowRiskRequest(t *testing.T, now time.Time, id string) runtime.ExecutionRequest {
	t.Helper()
	boundary := runtime.SafetyBoundary{Now: func() time.Time { return now }}
	recommendation := runtime.Recommendation{
		RequestID: id, BrainIdentity: runtime.BrainIdentity, Intent: "action",
		RiskLevel: runtime.RiskLow, Confidence: 0.1, CreatedAt: now,
	}
	assessment, err := boundary.Assess(recommendation)
	if err != nil { t.Fatalf("assess: %v", err) }
	authorization, err := boundary.Authorize(recommendation, assessment, "")
	if err != nil { t.Fatalf("authorize: %v", err) }
	request, err := boundary.BuildExecutionRequest(recommendation, assessment, authorization, now.Add(time.Minute))
	if err != nil { t.Fatalf("build request: %v", err) }
	return request
}

func TestGate8BDirectDispatchFailsClosed(t *testing.T) {
	core := NewExecutionCore()
	p := &gate8TestPlugin{}
	core.RegisterPlugin("action", p)
	if err := core.Dispatch("action", "direct"); err == nil { t.Fatal("direct dispatch unexpectedly succeeded") }
	if got := p.Count(); got != 0 { t.Fatalf("direct dispatch triggered plugin %d times", got) }
}

func TestGate8BGatewayExecutesAuthorizedRequest(t *testing.T) {
	now := time.Date(2026, 9, 15, 20, 0, 0, 0, time.UTC)
	core := NewExecutionCore()
	p := &gate8TestPlugin{}
	core.RegisterPlugin("action", p)
	gateway := NewExecutionGateway(core)
	gateway.Now = func() time.Time { return now }
	request := buildLowRiskRequest(t, now, "gate8b-1")
	if err := gateway.Execute(request); err != nil { t.Fatalf("execute: %v", err) }
	if got := p.Count(); got != 1 { t.Fatalf("plugin trigger count: got %d want 1", got) }
}

func TestGate8BGatewayRejectsFabricatedRequest(t *testing.T) {
	now := time.Date(2026, 9, 15, 20, 0, 0, 0, time.UTC)
	core := NewExecutionCore()
	p := &gate8TestPlugin{}
	core.RegisterPlugin("action", p)
	gateway := NewExecutionGateway(core)
	gateway.Now = func() time.Time { return now }
	err := gateway.Execute(runtime.ExecutionRequest{
		RequestID: "gate8b-2", BrainIdentity: runtime.BrainIdentity, Intent: "action",
		RiskLevel: runtime.RiskLow, Expiry: now.Add(time.Minute), IdempotencyKey: "gate8b-2",
	})
	if err == nil { t.Fatal("fabricated request was accepted") }
	if p.Count() != 0 { t.Fatal("fabricated request reached plugin") }
}

func TestGate8BGatewayRejectsUnknownBrain(t *testing.T) {
	now := time.Date(2026, 9, 15, 20, 0, 0, 0, time.UTC)
	core := NewExecutionCore()
	p := &gate8TestPlugin{}
	core.RegisterPlugin("action", p)
	gateway := NewExecutionGateway(core)
	gateway.Now = func() time.Time { return now }
	err := gateway.Execute(runtime.ExecutionRequest{
		RequestID: "gate8b-3", BrainIdentity: "foreign-brain", Intent: "action",
		RiskLevel: runtime.RiskLow, Expiry: now.Add(time.Minute), IdempotencyKey: "gate8b-3",
	})
	if err == nil { t.Fatal("unknown brain was accepted") }
	if p.Count() != 0 { t.Fatal("unknown brain reached plugin") }
}

func TestGate8BGatewayRejectsExpiredAuthorizedRequest(t *testing.T) {
	created := time.Date(2026, 9, 15, 20, 0, 0, 0, time.UTC)
	now := created.Add(2 * time.Minute)
	core := NewExecutionCore()
	p := &gate8TestPlugin{}
	core.RegisterPlugin("action", p)
	gateway := NewExecutionGateway(core)
	gateway.Now = func() time.Time { return now }
	request := buildLowRiskRequest(t, created, "gate8b-4")
	if err := gateway.Execute(request); err == nil { t.Fatal("expired authorized request was accepted") }
	if p.Count() != 0 { t.Fatal("expired request reached plugin") }
}

func TestGate8BGatewayIdempotencyPreventsDuplicateExecution(t *testing.T) {
	now := time.Date(2026, 9, 15, 20, 0, 0, 0, time.UTC)
	core := NewExecutionCore()
	p := &gate8TestPlugin{}
	core.RegisterPlugin("action", p)
	gateway := NewExecutionGateway(core)
	gateway.Now = func() time.Time { return now }
	request := buildLowRiskRequest(t, now, "gate8b-5")
	if err := gateway.Execute(request); err != nil { t.Fatalf("first execute: %v", err) }
	if err := gateway.Execute(request); err == nil { t.Fatal("duplicate execution was accepted") }
	if got := p.Count(); got != 1 { t.Fatalf("plugin trigger count: got %d want 1", got) }
}

func TestGate8BGatewayRejectsFabricatedApprovalPolicyRequest(t *testing.T) {
	now := time.Date(2026, 9, 15, 20, 0, 0, 0, time.UTC)
	core := NewExecutionCore()
	p := &gate8TestPlugin{}
	core.RegisterPlugin("action", p)
	gateway := NewExecutionGateway(core)
	gateway.Now = func() time.Time { return now }
	if err := gateway.Execute(runtime.ExecutionRequest{
		RequestID: "gate8b-6", BrainIdentity: runtime.BrainIdentity, Intent: "action",
		RiskLevel: runtime.RiskLow, RequiredApproval: true, Expiry: now.Add(time.Minute),
		IdempotencyKey: "gate8b-6",
	}); err == nil { t.Fatal("fabricated approval policy request was accepted") }
	if p.Count() != 0 { t.Fatal("invalid approval request reached plugin") }
}
