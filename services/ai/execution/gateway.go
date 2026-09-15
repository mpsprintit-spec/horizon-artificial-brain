package execution

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

// ExecutionGateway is the only execution boundary exposed to policy-approved
// execution requests. Cognition never receives a reference to this gateway.
type ExecutionGateway struct {
	Core *ExecutionCore
	Now  func() time.Time

	mu          sync.Mutex
	executedIDs map[string]struct{}
}

func NewExecutionGateway(core *ExecutionCore) *ExecutionGateway {
	if core == nil {
		core = NewExecutionCore()
	}
	return &ExecutionGateway{
		Core:        core,
		executedIDs: make(map[string]struct{}),
	}
}

// Execute validates the complete execution contract before touching a plugin.
// It fails closed on authorization, identity, expiry, policy and idempotency
// violations.
func (g *ExecutionGateway) Execute(request runtime.ExecutionRequest) error {
	if g == nil || g.Core == nil {
		return errors.New("execution gateway is not initialized")
	}
	if !request.Authorized() {
		return errors.New("execution request has no valid safety authorization")
	}
	if request.RequestID == "" || request.IdempotencyKey == "" {
		return errors.New("execution request identity is required")
	}
	if request.BrainIdentity != runtime.BrainIdentity {
		return fmt.Errorf("execution request belongs to unknown brain %q", request.BrainIdentity)
	}
	if request.Intent == "" {
		return errors.New("execution request intent is required")
	}
	if request.Expiry.IsZero() || !request.Expiry.After(g.now()) {
		return errors.New("execution request is expired")
	}
	if request.RequiredApproval && request.RiskLevel == runtime.RiskLow {
		return errors.New("low-risk execution cannot require approval")
	}
	if !request.RequiredApproval && request.RiskLevel != runtime.RiskLow {
		return errors.New("non-low-risk execution requires explicit approval")
	}

	g.mu.Lock()
	defer g.mu.Unlock()
	if _, exists := g.executedIDs[request.IdempotencyKey]; exists {
		return errors.New("execution request already executed")
	}

	g.Core.dispatchAuthorized(request.Intent, request.Intent)
	g.executedIDs[request.IdempotencyKey] = struct{}{}
	return nil
}

func (g *ExecutionGateway) now() time.Time {
	if g.Now != nil {
		return g.Now().UTC()
	}
	return time.Now().UTC()
}
