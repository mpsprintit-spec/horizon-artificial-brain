package runtime

import "time"

// Settle passive retention after durable intent but before any learning at
// that command time. Reinforcement must see the retained, not stale, weight.
// Legacy encodings keep their historical placement and repeated-age law.
func (r *BrainRuntime) settleMemoryTimeLocked(at time.Time) {
	if r.dynamicsEncoding.HasPhysicalMemory() {
		r.brain.ApplyMemoryDynamicsWithEncoding(at, r.dynamicsEncoding)
	}
}
