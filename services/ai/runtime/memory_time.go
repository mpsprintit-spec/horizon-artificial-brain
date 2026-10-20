package runtime

import (
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// Settle passive retention after durable intent but before any learning at
// that command time. Reinforcement must see the retained, not stale, weight.
// Legacy encodings keep their historical placement and repeated-age law.
func (r *BrainRuntime) settleMemoryTimeLocked(at time.Time) {
	if r.dynamicsEncoding == knowledge.PhysicalMemoryDynamics {
		r.brain.ApplyMemoryDynamicsWithEncoding(at, r.dynamicsEncoding)
	}
}
