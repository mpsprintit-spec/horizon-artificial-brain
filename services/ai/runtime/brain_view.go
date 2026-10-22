package runtime

import (
	"errors"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// Deprecated: WithBrainView is restricted to standalone, unjournaled legacy
// experiments. Managed or journaled runtimes reject it without invoking fn.
// Use InspectBrain for detached values and typed commands for mutation.

func (r *BrainRuntime) WithBrainView(fn func(*knowledge.Brain) error) error {
	if r == nil || r.brain == nil {
		return errors.New("brain runtime is not initialized")
	}
	if fn == nil {
		return errors.New("brain view callback is nil")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ensureWritableLocked(); err != nil {
		return err
	}
	if r.supervised || r.eventLog != nil {
		return ErrManagedRuntimeAccess
	}
	return fn(r.brain)
}
