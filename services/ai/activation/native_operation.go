package activation

import (
	"errors"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"math"
	"slices"
	"sort"
	"time"
)

// NativeOperationStats counts actual substrate work, never inferred evidence.
type NativeOperationStats struct {
	ActiveNodes  int `json:"active_nodes"`
	ReadSynapses int `json:"read_synapses"`
	MutatedNodes int `json:"mutated_nodes"`
}

// ExecuteNativeOperation is an opt-in compute primitive over the same recurrent
// state and canonical synapses. No observation, pattern evidence, consequence,
// or predictive certainty is synthesized. The output becomes the next input;
// there is no required pipeline, external stimulus, or language output.
func (e *Engine) PrepareNativeOperation(op knowledge.NativeOperation, at time.Time, maxNodes, maxEdges int) (*NativeTransition, error) {
	out := NativeOperationStats{}
	if maxNodes < 1 || maxEdges < 1 {
		return nil, errors.New("native work budget required")
	}
	if err := op.Validate(); err != nil {
		return nil, err
	}
	if at.IsZero() {
		return nil, errors.New("native operation needs event time")
	}

	e.mu.RLock()
	defer e.mu.RUnlock()
	e.Memory.RLock()
	defer e.Memory.RUnlock()
	reserved := e.Memory.NativeOwnershipLocked()
	state := e.internalState
	if len(state) > maxNodes {
		return nil, errors.New("native active-state budget exceeded")
	}
	if op.Kind == "stop" {
		return &NativeTransition{owner: e, before: cloneState(state), beforeConfidence: cloneState(e.internalConfidence), state: cloneState(state), confidence: cloneState(e.internalConfidence), stop: true}, nil
	}
	ids := sortedNodeIDs(state)
	next := make(map[knowledge.NodeID]float64, len(state))
	conf := make(map[knowledge.NodeID]float64, len(state))
	for _, id := range ids {
		if reserved.Has(id) || state[id] == 0 {
			continue
		}
		out.ActiveNodes++
		conf[id] = e.internalConfidence[id]
	}
	switch op.Kind {
	case "propagate":
		for _, id := range ids {
			if reserved.Has(id) || state[id] == 0 {
				continue
			}
			n := e.Memory.Registry.GetByID(id)
			if n == nil {
				return nil, errors.New("native state neuron missing")
			}
			next[id] += state[id] * (1 - op.Gain)
			// Total ordering fixes opposing-channel accumulation without changing the
			// historical recurrent law. No cache can become stale after graph growth.
			targets := make([]knowledge.NodeID, 0, len(n.Synapses))
			for target := range n.Synapses {
				if !reserved.Has(target) {
					targets = append(targets, target)
				}
			}
			slices.Sort(targets)
			for _, target := range targets {
				channels := n.Synapses[target]
				if len(channels) > 1 {
					channels = append(knowledge.SynapseList(nil), channels...)
					sort.SliceStable(channels, func(i, j int) bool {
						a, b := channels[i], channels[j]
						if a == nil {
							return b != nil
						}
						if b == nil {
							return false
						}
						if a.Inhibitory != b.Inhibitory {
							return !a.Inhibitory
						}
						if a.Weight != b.Weight {
							return a.Weight < b.Weight
						}
						if a.Confidence != b.Confidence {
							return a.Confidence < b.Confidence
						}
						return a.LastActivation.Before(b.LastActivation)
					})
				}
				for _, s := range channels {
					if s == nil {
						continue
					}
					out.ReadSynapses++
					if out.ReadSynapses > maxEdges {
						return nil, errors.New("native synapse-read budget exceeded")
					}
					pulse := state[id] * s.Weight * s.Confidence * op.Gain * temporalPenalty(at, s.LastActivation)
					if s.Inhibitory {
						pulse = -pulse
					}
					next[target] += pulse
					if len(next) > maxNodes {
						return nil, errors.New("native active-state budget exceeded")
					}
					conf[target] = max(conf[target], e.internalConfidence[id]*s.Confidence)
				}
			}
		}
	case "focus":
		mean := 0.0
		for _, id := range ids {
			if !reserved.Has(id) {
				mean += state[id]
			}
		}
		if out.ActiveNodes > 0 {
			mean /= float64(out.ActiveNodes)
		}
		for _, id := range ids {
			if !reserved.Has(id) {
				next[id] = math.Max(0, state[id]-op.Gain*mean)
			}
		}
	case "relax":
		for _, id := range ids {
			if !reserved.Has(id) {
				next[id] = state[id] * (1 - op.Gain)
			}
		}
	}
	for _, id := range sortedNodeIDs(next) {
		v := squash(next[id])
		if v == 0 {
			delete(next, id)
			delete(conf, id)
		} else {
			next[id] = v
		}
	}
	return &NativeTransition{owner: e, before: cloneState(state), beforeConfidence: cloneState(e.internalConfidence), state: next, confidence: conf, stats: out}, nil
}

type NativeTransition struct {
	owner                    *Engine
	consumed                 bool
	before, beforeConfidence map[knowledge.NodeID]float64
	state, confidence        map[knowledge.NodeID]float64
	stats                    NativeOperationStats
	stop                     bool
}

func (e *Engine) CommitNativeOperation(plan *NativeTransition) (NativeOperationStats, error) {
	if plan == nil {
		return NativeOperationStats{}, errors.New("missing native transition")
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.Memory.Lock()
	defer e.Memory.Unlock()
	if plan.owner != e || plan.consumed || !equalNativeState(e.internalState, plan.before) || !equalNativeState(e.internalConfidence, plan.beforeConfidence) {
		return NativeOperationStats{}, errors.New("stale native transition")
	}
	plan.consumed = true
	if plan.stop {
		return plan.stats, nil
	}
	touched := map[knowledge.NodeID]bool{}
	for id := range e.internalState {
		touched[id] = true
	}
	for id := range plan.state {
		touched[id] = true
	}
	out := plan.stats
	for id := range touched {
		if n := e.Memory.Registry.GetByID(id); n != nil {
			n.Activation = clamp01(plan.state[id])
			out.MutatedNodes++
		}
	}
	e.internalState = plan.state
	e.internalConfidence = plan.confidence
	return out, nil
}
func equalNativeState(a, b map[knowledge.NodeID]float64) bool {
	if len(a) != len(b) {
		return false
	}
	for id, v := range a {
		other, ok := b[id]
		if !ok || other != v {
			return false
		}
	}
	return true
}
func (e *Engine) ExecuteNativeOperation(op knowledge.NativeOperation, at time.Time) (NativeOperationStats, error) {
	plan, err := e.PrepareNativeOperation(op, at, 1000000, 10000000)
	if err != nil {
		return NativeOperationStats{}, err
	}
	return e.CommitNativeOperation(plan)
}

// ApplySparseObservedPlasticity is an explicit lazy-time variant of the local
// activity law. Untouched eligibility stays at its last materialized timestamp;
// decayEligibility yields its effective value at any requested later time.
// Only direct observed pre/post states may call this; thought does not call it.
func (e *Engine) ApplySparseObservedPlasticity(pre, post map[knowledge.NodeID]float64, at time.Time) int {
	if e == nil || e.Memory == nil || at.IsZero() {
		return 0
	}
	e.Memory.Lock()
	defer e.Memory.Unlock()
	native := e.Memory.NativeOwnershipLocked()
	mutations := 0
	visit := func(n *knowledge.ConceptNode) {
		p := clamp01(pre[n.ID])
		if p <= 0 {
			return
		}
		for target, list := range n.Synapses {
			co := p * clamp01(post[target])
			if co <= 0 {
				continue
			}
			for _, s := range list {
				if s == nil || (native.Enabled() && (native.Has(n.ID) || native.Has(target))) {
					continue
				}
				elig := decayEligibility(s.Dynamic.Eligibility, s.Dynamic.LastEligibilityUpdate, at)
				elig = clamp01(elig + .15*co*(1-elig))
				s.Dynamic.Eligibility = elig
				if s.Dynamic.LastEligibilityUpdate.IsZero() || at.After(s.Dynamic.LastEligibilityUpdate) {
					s.Dynamic.LastEligibilityUpdate = at
				}
				if co >= activityPlasticityMinActive {
					d := activityPlasticityLearningRate * co * elig
					s.Dynamic.Weight = clamp(s.Dynamic.Weight+d, activityPlasticityMinWeight, activityPlasticityMaxWeight)
					s.Dynamic.LastModification = at
					if !s.Inhibitory {
						s.Dynamic.Confidence = clamp01(s.Dynamic.Confidence + d*.5)
						s.Dynamic.Frequency++
						s.Dynamic.LastActivation = at
					}
				}
				s.Weight = s.Dynamic.Weight
				if !s.Inhibitory {
					s.Confidence = s.Dynamic.Confidence
				}
				mutations++
			}
		}
	}
	if len(pre)*2 >= e.Memory.Registry.Len() {
		e.Memory.Registry.VisitNodes(visit)
	} else {
		for id, v := range pre {
			if v <= 0 {
				continue
			}
			if n := e.Memory.Registry.GetByID(id); n != nil {
				visit(n)
			}
		}
	}
	return mutations
}
