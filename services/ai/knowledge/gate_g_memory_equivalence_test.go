package knowledge

import (
	"bytes"
	"fmt"
	"math"
	"math/rand"
	"testing"
	"time"
)

func TestGateGPhysicalTimeMatchesFrozenClosure(t *testing.T) {
	at := time.Date(2026, 11, 6, 0, 0, 0, 123, time.UTC)
	for _, seed := range []int64{37, 83, 151} {
		for _, native := range []bool{false, true} {
			for _, retention := range []float64{0, .2, .6, 1, 1.2} {
				t.Run(fmt.Sprintf("%d/native=%t/r=%g", seed, native, retention), func(t *testing.T) {
					fixture := func() *Brain {
						r := rand.New(rand.NewSource(seed))
						b := NewBrain()
						nodes := make([]*ConceptNode, 16)
						for i := range nodes {
							nodes[i] = b.StoreAt(fmt.Sprintf("world-%d", i), at)
						}
						for i, n := range nodes {
							for j := 1; j <= 4; j++ {
								target := nodes[(i+j)%len(nodes)]
								inhibitory := j%2 == 0
								b.ConnectAt(n, target, .1+.8*r.Float64(), .1+.8*r.Float64(), inhibitory, at)
								s := n.FindDynamicSynapse(target.ID, inhibitory)
								s.Dynamic.LastModification = []time.Time{{}, at.Add(-time.Hour), at, at.Add(3 * time.Hour)}[(i+j)%4]
								s.Dynamic.Eligibility = []float64{0, math.Copysign(0, -1), .3, .9}[(i+j)%4]
								s.Weight = .123
								s.Confidence = .456
							}
						}
						b.BrainState.MemoryState["retention"] = retention
						b.BrainState.ExperienceTraces = []ExperienceTrace{{CurrentAt: at.Add(-time.Hour), MemoryPriority: .7}, {CurrentAt: at.Add(3 * time.Hour), MemoryPriority: .5}, {MemoryPriority: .2}}
						if native {
							c := NativeControlConfig{Version: NativeControlCausalVersion, Encoder: NativeRelationalEncoderV2, Sources: []NodeID{nodes[0].ID}, Operations: []NativeOperation{{Kind: "propagate", Gain: .4, Cost: .01}, {Kind: "stop"}}, Seed: seed, LearningRate: .01, Gamma: 1, Lambda: 1, Temperature: 1, MaxSteps: 4, MaxActiveNodes: 512, MaxSynapseReads: 4096}
							if err := b.InitializeNativeControl(c, at); err != nil {
								t.Fatal(err)
							}
						}
						return b
					}
					live, reference := fixture(), fixture()
					for step := 0; step < 24; step++ {
						now := at.Add(time.Duration(step/3)*time.Hour + time.Duration(step%3-1)*time.Nanosecond)
						if step%3 == 1 {
							now = now.Add(-2 * time.Hour)
						}
						if step == 13 {
							live.BrainState.MemoryState["retention"] = .9
							reference.BrainState.MemoryState["retention"] = .9
						}
						live.ApplyMemoryDynamicsWithEncoding(now, PhysicalMemoryDynamics)
						reference.applyMemoryDynamicsFrozenClosure(now, PhysicalMemoryDynamics)
						a, e := live.CanonicalJSON()
						if e != nil {
							t.Fatal(e)
						}
						b, e := reference.CanonicalJSON()
						if e != nil {
							t.Fatal(e)
						}
						if !bytes.Equal(a, b) {
							t.Fatalf("physical-time canonical mismatch step%d", step)
						}
					}
				})
			}
		}
	}
}

func (k *KnowledgeBase) applyMemoryDynamicsFrozenClosure(now time.Time, dynamics DynamicsEncoding) {
	if !dynamics.HasPhysicalMemory() {
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
	protectedN := k.protectedNParameter()
	native := k.NativeOwnershipLocked()
	k.Registry.VisitNodes(func(source *ConceptNode) {
		if source == nil || (native.Enabled() && native.Has(source.ID)) {
			return
		}
		for _, list := range source.Synapses {
			for _, synapse := range list {
				if synapse == nil || (native.Enabled() && native.Has(synapse.TargetID)) {
					continue
				}
				d := &synapse.Dynamic
				factor := factorSince(d.LastModification, 1)
				if factor == 1 {
					continue
				}
				if !protectedN(source.ID, synapse.TargetID) {
					d.Weight = clamp01(d.Weight * factor)
				}
				// Half the weight rate preserves slower confidence forgetting
				// while obeying the same interval-composition law.
				d.Confidence = clamp01(d.Confidence * factorSince(d.LastModification, .5))
				syncSynapseLegacyState(synapse)
			}
		}
	})
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
