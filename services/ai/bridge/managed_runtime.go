package bridge

import (
	"errors"

	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

// NewManaged binds the Bridge to the active Horizon brain through its
// publication supervisor. Unlike New, it does not retain a BrainRuntime
// pointer that could become stale after checkpoint recovery.
func NewManaged(supervisor *runtime.RuntimeSupervisor, cfg Config) (*Server, error) {
	if supervisor == nil { return nil, errors.New("brain runtime supervisor is required") }
	if err := supervisor.With(func(*runtime.BrainRuntime) error { return nil }); err != nil {
		return nil, err
	}
	if cfg.MaxPayloadBytes <= 0 { cfg.MaxPayloadBytes = 4 << 20 }
	if cfg.RateLimitPerMinute <= 0 { cfg.RateLimitPerMinute = 600 }
	return &Server{supervisor: supervisor, cfg: cfg, rate: make(map[string]*rateBucket)}, nil
}

func (s *Server) withRuntime(fn func(*runtime.BrainRuntime) error) error {
	if s == nil { return errors.New("brain bridge is not initialized") }
	if s.supervisor != nil { return s.supervisor.With(fn) }
	if s.runtime == nil { return errors.New("brain runtime is not initialized") }
	if s.runtime.RecoveryRequired() { return runtime.ErrRuntimeRecoveryRequired }
	return fn(s.runtime)
}

// withMutatingRuntime excludes concurrent managed observations, outcome
// updates, thinking cycles, and handover during a multi-step operation.
func (s *Server) withMutatingRuntime(fn func(*runtime.BrainRuntime) error) error {
	if s == nil { return errors.New("brain bridge is not initialized") }
	if s.supervisor != nil { return s.supervisor.WithMutation(fn) }
	return s.withRuntime(fn)
}

// withRuntimeEpoch obtains a lease and an epoch invalidation signal. It is
// used to initialize long-lived telemetry subscriptions without holding
// the supervisor read lock across a WebSocket session.
func (s *Server) withRuntimeEpoch(fn func(*runtime.BrainRuntime, <-chan struct{}) error) error {
	if s == nil { return errors.New("brain bridge is not initialized") }
	if s.supervisor != nil { return s.supervisor.WithEpoch(fn) }
	return s.withRuntime(func(rt *runtime.BrainRuntime) error { return fn(rt, nil) })
}

func epochExpired(epoch <-chan struct{}) bool {
	if epoch == nil { return false }
	select {
	case <-epoch:
		return true
	default:
		return false
	}
}
