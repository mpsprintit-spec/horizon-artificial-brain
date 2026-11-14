package activation

// Frozen general activity law from a9e4ed7, compared against the specialized
// production path over complete canonical state, not a reimplementation of it.
import (
	"bytes"
	"fmt"
	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
	"math"
	"math/rand"
	"reflect"
	"sort"
	"testing"
	"time"
)

func TestGateGEmptyLocalActivityMatchesFrozenLaw(t *testing.T) {
	at := time.Date(2026, 11, 6, 0, 0, 0, 123, time.UTC)
	for _, seed := range []int64{37, 83, 151} {
		for _, version := range []string{"none", k.NativeControlVersion, k.NativeControlRelationalVersion, k.NativeControlCausalVersion} {
			t.Run(fmt.Sprintf("%d/%s", seed, version), func(t *testing.T) {
				fixture := func() *Engine {
					r := rand.New(rand.NewSource(seed))
					brain := k.NewBrain()
					nodes := []*k.ConceptNode{}
					for i := 0; i < 24; i++ {
						nodes = append(nodes, brain.StoreAt(fmt.Sprintf("world-%d", i), at))
					}
					for i, n := range nodes {
						for j := 1; j <= 4; j++ {
							target := nodes[(i+j)%len(nodes)]
							inhibitory := r.Intn(2) == 0
							brain.ConnectAt(n, target, .1+.8*r.Float64(), .1+.8*r.Float64(), inhibitory, at)
							s := n.FindDynamicSynapse(target.ID, inhibitory)
							s.Dynamic.Eligibility = []float64{0, math.Copysign(0, -1), .02, .25, .999999, 1}[r.Intn(6)]
							s.Dynamic.LastEligibilityUpdate = []time.Time{{}, at.Add(-time.Second), at.Add(5 * time.Second)}[r.Intn(3)]
							// Deliberately stale legacy mirrors must also be settled identically.
							s.Weight = .123
							s.Confidence = .456
						}
					}
					if version != "none" {
						cfg := k.NativeControlConfig{Version: version, Sources: []k.NodeID{nodes[0].ID, nodes[1].ID}, Operations: []k.NativeOperation{{Kind: "propagate", Gain: .4, Cost: .01}, {Kind: "stop"}}, Seed: seed, LearningRate: .01, Gamma: 1, Lambda: 1, Temperature: 1, MaxSteps: 4, MaxActiveNodes: 512, MaxSynapseReads: 4096}
						if version != k.NativeControlVersion {
							cfg.Encoder = "local-relational-v1"
						}
						if e := brain.InitializeNativeControl(cfg, at); e != nil {
							t.Fatal(e)
						}
					}
					return NewEngine(brain)
				}
				live, reference := fixture(), fixture()
				equal := func() {
					a, e := live.Memory.CanonicalJSON()
					if e != nil {
						t.Fatal(e)
					}
					b, e := reference.Memory.CanonicalJSON()
					if e != nil {
						t.Fatal(e)
					}
					if !bytes.Equal(a, b) {
						t.Fatal("complete canonical state differs from frozen activity law")
					}
				}
				equal()
				for i := 0; i < 32; i++ {
					now := at.Add(time.Duration((i%8)-2) * time.Second)
					live.ApplyActivityPlasticityWithDynamics(nil, nil, now, k.LocalActivityDynamics)
					reference.applyActivityPlasticityFrozenGateG(nil, nil, now, true)
					equal()
				}
			})
		}
	}
}

func TestGateGTypedSynapseSortPreservesFrozenOrder(t *testing.T) {
	for _, seed := range []int64{37, 83, 151} {
		r := rand.New(rand.NewSource(seed))
		for _, n := range []int{0, 1, 4, 8, 12, 13, 64, 257, 1000} {
			original := make([]*k.Synapse, n)
			for i := range original {
				original[i] = &k.Synapse{TargetID: k.NodeID(1 + r.Intn(31)), Weight: float64(i)}
			}
			actual, reference := append([]*k.Synapse(nil), original...), append([]*k.Synapse(nil), original...)
			sortSynapsesByTarget(actual)
			sort.Slice(reference, func(i, j int) bool { return reference[i].TargetID < reference[j].TargetID })
			if !reflect.DeepEqual(actual, reference) {
				t.Fatalf("tie/accumulation ordering changed seed%d size%d", seed, n)
			}
		}
	}
}

func (e *Engine) applyActivityPlasticityFrozenGateG(preState, postState map[k.NodeID]float64, now time.Time, local bool) {
	if e == nil || e.Memory == nil {
		return
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	e.Memory.Lock()
	defer e.Memory.Unlock()
	native := e.Memory.NativeOwnershipLocked()

	e.Memory.Registry.VisitNodes(func(source *k.ConceptNode) {
		if native.Enabled() && native.Has(source.ID) {
			return
		}
		pre := clamp01(preState[source.ID])
		for _, list := range source.Synapses {
			for _, synapse := range list {
				if synapse == nil || (native.Enabled() && native.Has(synapse.TargetID)) {
					continue
				}

				target := clamp01(postState[synapse.TargetID])
				coactivity := pre * target
				eligibility := decayEligibility(synapse.Dynamic.Eligibility, synapse.Dynamic.LastEligibilityUpdate, now)

				if local {
					// Only elapsed time forgets eligibility. Actual co-activity
					// adds bounded evidence; empty calls cannot multiply it away.
					eligibility = clamp01(eligibility + .15*coactivity*(1-eligibility))
				} else {
					eligibility = clamp01(eligibility*0.85 + coactivity*0.15)
				}
				synapse.Dynamic.Eligibility = eligibility
				if !local || synapse.Dynamic.LastEligibilityUpdate.IsZero() || now.After(synapse.Dynamic.LastEligibilityUpdate) {
					synapse.Dynamic.LastEligibilityUpdate = now
				}

				if synapse.Inhibitory {
					if coactivity >= activityPlasticityMinActive {
						delta := activityPlasticityLearningRate * coactivity * eligibility
						synapse.Dynamic.Weight = clamp(synapse.Dynamic.Weight+delta, activityPlasticityMinWeight, activityPlasticityMaxWeight)
						synapse.Dynamic.LastModification = now
					}
					synapse.Weight = synapse.Dynamic.Weight
					continue
				}

				if coactivity >= activityPlasticityMinActive {
					delta := activityPlasticityLearningRate * coactivity * eligibility
					synapse.Dynamic.Weight = clamp(synapse.Dynamic.Weight+delta, activityPlasticityMinWeight, activityPlasticityMaxWeight)
					synapse.Dynamic.Confidence = clamp01(synapse.Dynamic.Confidence + delta*0.5)
					synapse.Dynamic.Frequency++
					synapse.Dynamic.LastActivation = now
					synapse.Dynamic.LastModification = now
				} else if !local {
					decay := activityPlasticityDecayRate * (1 - eligibility)
					synapse.Dynamic.Weight = clamp(synapse.Dynamic.Weight-decay, activityPlasticityMinWeight, activityPlasticityMaxWeight)
					synapse.Dynamic.Confidence = clamp01(synapse.Dynamic.Confidence - decay*0.25)
				}

				synapse.Weight = synapse.Dynamic.Weight
				synapse.Confidence = synapse.Dynamic.Confidence
			}
		}
	})
}
