package runtime

import (
	"errors"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// Deprecated: WithBrainView exposes a LIVE mutable *knowledge.Brain and
// cannot enforce read-only access or durable journaling for callback changes.
// It is retained only for legacy API compatibility. Production services,
// inspection tools and CLI consumers MUST use InspectBrain instead, which
// returns detached numerical values. The runtime lock here prevents standard
// concurrent methods, but it does not make this mutable callback safe.

func (r *BrainRuntime) WithBrainView(fn func(*knowledge.Brain) error) error {
	if r == nil || r.brain == nil { return errors.New("brain runtime is not initialized") }
	if fn == nil { return errors.New("brain view callback is nil") }
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ensureWritableLocked(); err != nil { return err }
	return fn(r.brain)
}
