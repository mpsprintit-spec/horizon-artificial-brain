package activation

import (
	"bytes"
	"fmt"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"math"
	"math/rand"
	"testing"
	"time"
)

// Differential oracle is the unchanged PR69 advance function below. Compare
// every canonical field and IEEE754 output bit; a speedup cannot change a law.
func TestGateGAdvanceMatchesFrozenClosure(t *testing.T) {
	at := time.Date(2026, 11, 6, 0, 0, 0, 123, time.UTC)
	for _, seed := range []int64{37, 83, 151} {
		for _, native := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d/native=%t", seed, native), func(t *testing.T) {
				fixture := func() *Engine {
					r := rand.New(rand.NewSource(seed))
					brain := knowledge.NewBrain()
					nodes := make([]*knowledge.ConceptNode, 32)
					for i := range nodes {
						nodes[i] = brain.StoreAt(fmt.Sprintf("world-%d", i), at)
						nodes[i].RestingActivation = r.Float64() * .1
					}
					for i, n := range nodes {
						for j := 1; j <= 4; j++ {
							brain.ConnectAt(n, nodes[(i+j)%len(nodes)], .1+.8*r.Float64(), .1+.8*r.Float64(), j%2 == 0, at)
						}
					}
					if native {
						cfg := knowledge.NativeControlConfig{Version: knowledge.NativeControlCausalVersion, Encoder: "local-relational-v2", Sources: []knowledge.NodeID{nodes[0].ID}, Operations: []knowledge.NativeOperation{{Kind: "propagate", Gain: .4, Cost: .01}, {Kind: "stop"}}, Seed: seed, LearningRate: .01, Gamma: 1, Lambda: 1, Temperature: 1, MaxSteps: 4, MaxActiveNodes: 512, MaxSynapseReads: 4096}
						if err := brain.InitializeNativeControl(cfg, at); err != nil {
							t.Fatal(err)
						}
					}
					return NewEngine(brain)
				}
				live, ref := fixture(), fixture()
				for step := 0; step < 36; step++ {
					x, c := map[knowledge.NodeID]float64{}, map[knowledge.NodeID]float64{}
					nodes := live.Memory.Registry.Nodes()
					for i, n := range nodes {
						if step%3 == 0 || i < 3 {
							x[n.ID] = []float64{0, math.Copysign(0, -1), .03, .8, 1.2}[i%5]
							c[n.ID] = .7
						}
					}
					if step%3 == 1 {
						x[knowledge.NodeID(999999)] = .1
					}
					beforeX, beforeC := cloneState(x), cloneState(c)
					now := at.Add(time.Duration(step%7-2) * time.Second)
					a, ac := live.advance(x, c, now, step%6-1)
					b, bc := ref.advanceFrozenClosure(x, c, now, step%6-1)
					equalBits := func(a, b map[knowledge.NodeID]float64) {
						t.Helper()
						if len(a) != len(b) {
							t.Fatal("map size changed")
						}
						for id, v := range a {
							w, ok := b[id]
							if !ok || math.Float64bits(v) != math.Float64bits(w) {
								t.Fatalf("output bits changed step%d id%d", step, id)
							}
						}
					}
					equalBits(a, b)
					equalBits(ac, bc)
					equalBits(x, beforeX)
					equalBits(c, beforeC)
					raw, e := live.Memory.CanonicalJSON()
					if e != nil {
						t.Fatal(e)
					}
					old, e := ref.Memory.CanonicalJSON()
					if e != nil {
						t.Fatal(e)
					}
					if !bytes.Equal(raw, old) {
						t.Fatalf("canonical state changed step%d", step)
					}
					a[999998] = 1
					ac[999998] = 1
					if _, ok := x[999998]; ok {
						t.Fatal("output aliases input state")
					}
					if _, ok := c[999998]; ok {
						t.Fatal("output aliases input confidence")
					}
				}
			})
		}
	}
}

func (e *Engine) advanceFrozenClosure(state, confidence map[knowledge.NodeID]float64, now time.Time, cycles int) (map[knowledge.NodeID]float64, map[knowledge.NodeID]float64) {
	if e == nil || e.Memory == nil {
		return cloneState(state), cloneState(confidence)
	}
	e.Memory.Lock()
	defer e.Memory.Unlock()
	native := e.Memory.NativeOwnershipLocked()
	state = cloneState(state)
	confidence = cloneState(confidence)
	for i := 0; i < cycles; i++ {
		next := map[knowledge.NodeID]float64{}
		nextConfidence := map[knowledge.NodeID]float64{}
		ids := sortedNodeIDs(state)
		for _, id := range ids {
			n := e.Memory.Registry.GetByID(id)
			if n == nil || (native.Enabled() && native.Has(id)) {
				continue
			}
			adaptive := n.Threshold - (n.Importance * .05) - (float64(n.Frequency) * .001)
			n.Threshold = clamp(adaptive, .12, .8)
			next[id] = n.RestingActivation
		}
		for _, id := range ids {
			level := state[id]
			n := e.Memory.Registry.GetByID(id)
			if n == nil || (native.Enabled() && native.Has(id)) {
				continue
			}
			next[id] += level * (1 - e.Decay)
			nextConfidence[id] = max(nextConfidence[id], confidence[id])
			synapses := n.OutboundAll()
			sortSynapsesByTarget(synapses)
			for _, s := range synapses {
				if native.Enabled() && native.Has(s.TargetID) {
					continue
				}
				age := temporalPenalty(now, s.LastActivation)
				pulse := level * s.Weight * s.Confidence * e.SpreadRate * age
				if s.Inhibitory {
					next[s.TargetID] -= pulse * e.Inhibition
				} else {
					next[s.TargetID] += pulse
				}
				nextConfidence[s.TargetID] = max(nextConfidence[s.TargetID], confidence[id]*s.Confidence*age)
				s.Activation = pulse
			}
		}
		state = normalize(next)
		confidence = normalize(nextConfidence)
	}
	return state, confidence
}
