package knowledge

import (
	"encoding/json"
	"errors"
	"math"
	"time"
)

// MemoryDecayState is an exact, monotonic physical-time watermark on the
// existing substrate. LastModification still means learning, never decay.
type MemoryDecayState struct {
	Encoding  DynamicsEncoding `json:"encoding"`
	AppliedAt time.Time        `json:"applied_at"`
}

// ApplyMemoryDynamicsWithEncoding keeps old journals on their original law.
// Passive retention does not count experiences or consume eligibility; those
// learning signals retain their separate activity/prediction/consequence clocks.
func (k *KnowledgeBase) ApplyMemoryDynamicsWithEncoding(now time.Time, dynamics DynamicsEncoding) {
	if dynamics != PhysicalMemoryDynamics {
		k.ApplyMemoryDynamics(now)
		return
	}
	if k == nil {
		return
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()
	k.mu.Lock()
	defer k.mu.Unlock()

	state := &k.BrainState
	var previous time.Time
	if state.MemoryDecay != nil {
		previous = state.MemoryDecay.AppliedAt
	} else if old, exists := state.MemoryState["last_decay_unix"]; exists && old >= float64(math.MinInt64) && old < float64(math.MaxInt64) {
		// Older snapshots already paid for decay through this timestamp. The
		// legacy float had sub-microsecond precision loss; do not charge the
		// full age again when adopting the new exact clock.
		previous = time.Unix(0, int64(old)).UTC()
	}
	if !previous.IsZero() && !now.After(previous) {
		return // Duplicate timestamp or rollback cannot rewind the watermark.
	}
	retention := state.MemoryState["retention"]
	if retention <= 0 {
		retention = state.LearningPolicyState.MemoryRetention
	}
	if retention <= 0 {
		retention = .6
	}
	retention = clamp01(retention)
	rate := (1 - retention) / 24.0 // Hours; e-fold time, not an asserted biological half-life.
	factorSince := func(created time.Time, scale float64) float64 {
		if created.IsZero() {
			return 1
		}
		if previous.After(created) {
			created = previous
		}
		if !now.After(created) {
			return 1
		}
		return expDecay(now.Sub(created).Hours() * rate * scale)
	}
	for _, source := range k.Registry.Nodes() {
		if source == nil {
			continue
		}
		for _, list := range source.Synapses {
			for _, synapse := range list {
				if synapse == nil {
					continue
				}
				d := &synapse.Dynamic
				factor := factorSince(d.LastModification, 1)
				if factor == 1 {
					continue
				}
				d.Weight = clamp01(d.Weight * factor)
				// Half the weight rate preserves slower confidence forgetting
				// while obeying the same interval-composition law.
				d.Confidence = clamp01(d.Confidence * factorSince(d.LastModification, .5))
				syncSynapseLegacyState(synapse)
			}
		}
	}
	for i := range state.ExperienceTraces {
		trace := &state.ExperienceTraces[i]
		trace.MemoryPriority = clamp01(trace.MemoryPriority * factorSince(trace.CurrentAt, 1))
	}
	state.MemoryDecay = &MemoryDecayState{Encoding: PhysicalMemoryDynamics, AppliedAt: now}
	if state.MemoryState == nil {
		state.MemoryState = map[string]float64{}
	}
	state.MemoryState["last_decay_unix"] = float64(now.UnixNano()) // Legacy telemetry only.
	state.MemoryState["effective_retention"] = retention
}

// ValidateMemoryDecayState runs before a restore mutates its destination.
// Canonical v3/checkpoint v10 authenticate the new clock; old state stays v2.
func ValidateMemoryDecayState(data []byte, allowPhysical bool) error {
	var raw struct {
		SchemaVersion int               `json:"schema_version"`
		MemoryDecay   *MemoryDecayState `json:"memory_decay,omitempty"`
		BrainState    struct {
			MemoryDecay *MemoryDecayState `json:"memory_decay,omitempty"`
		} `json:"brain_state"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if raw.SchemaVersion > 3 {
		return errors.New("unsupported canonical brain version")
	}
	if raw.BrainState.MemoryDecay != nil {
		return errors.New("legacy graph cannot carry physical memory state")
	}
	if raw.MemoryDecay == nil {
		if raw.SchemaVersion == 3 {
			return errors.New("canonical v3 is missing the memory clock")
		}
		return nil
	}
	if !allowPhysical || raw.SchemaVersion != 3 {
		return errors.New("legacy checkpoint/brain cannot carry physical memory state")
	}
	if raw.MemoryDecay.Encoding != PhysicalMemoryDynamics || raw.MemoryDecay.AppliedAt.IsZero() {
		return errors.New("invalid physical memory clock")
	}
	return nil
}
