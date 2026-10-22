package runtime

import (
	"errors"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// WithBrainView permits a synchronous examination of the currently owned
// canonical neural substrate. The callback must not keep node/brain pointers
// or launch asynchronous work. The runtime lock prevents normal runtime
// mutations during the visit; callers needing handover safety must also
// acquire the RuntimeSupervisor lease before entering this method.
func (r *BrainRuntime) WithBrainView(fn func(*knowledge.Brain) error) error {
	if r == nil || r.brain == nil { return errors.New("brain runtime is not initialized") }
	if fn == nil { return errors.New("brain view callback is nil") }
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ensureWritableLocked(); err != nil { return err }
	return fn(r.brain)
}
