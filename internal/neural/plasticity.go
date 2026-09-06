package neural

import "math"

// Learn applies three coupled mechanisms: eligibility-based association,
// homeostatic normalization, and slow decay. No symbolic relation is created.
func (n *Network) Learn(reward float64) {
	n.mu.Lock(); defer n.mu.Unlock()
	if math.IsNaN(reward) || math.IsInf(reward,0) { return }
	reward = clamp(reward, -1, 1)
	for _, s := range n.synapses {
		pre, post := n.neurons[s.Pre], n.neurons[s.Post]
		// Reward-modulated three-factor rule. Eligibility carries temporal
		// credit from prior co-activity; reward determines direction.
		dw := n.cfg.LearningRate * reward * s.Eligibility
		// Unrewarded co-activity has a small Hebbian component, while anti-
		// correlation depresses unused associations.
		dw += n.cfg.LearningRate * 0.1 * (pre.Activity*post.Activity - 0.05*s.Usage)
		s.Weight = clamp(s.Weight+dw, n.cfg.MinWeight, n.cfg.MaxWeight)
		s.Weight *= 1 - n.cfg.SynapseDecay
	}
	n.normalizeIncomingLocked()
}

func (n *Network) normalizeIncomingLocked() {
	for post, ids := range n.in {
		_ = post
		var norm float64
		for _, id := range ids { if s:=n.synapses[id]; s!=nil { norm += s.Weight*s.Weight } }
		if norm == 0 { continue }
		norm = math.Sqrt(norm)
		limit := math.Sqrt(float64(len(ids))) * n.cfg.MaxWeight * 0.5
		if norm <= limit { continue }
		scale := limit/norm
		for _, id := range ids { if s:=n.synapses[id]; s!=nil { s.Weight *= scale } }
	}
}

// StructuralAdaptation grows a connection only when persistent co-activity
// indicates an unconnected pair, and prunes persistently unused edges. It is
// intentionally conservative to prevent combinatorial graph explosion.
func (n *Network) StructuralAdaptation() {
	n.mu.Lock(); defer n.mu.Unlock()
	for id, s := range n.synapses {
		if s.Age > 100 && s.Usage < 0.001 { n.removeSynapseLocked(id) }
	}
}

func (n *Network) removeSynapseLocked(id SynapseID) {
	s, ok := n.synapses[id]; if !ok { return }
	delete(n.synapses,id)
	filter := func(xs []SynapseID) []SynapseID { out:=xs[:0]; for _,x:=range xs { if x!=id { out=append(out,x) } }; return out }
	n.out[s.Pre] = filter(n.out[s.Pre]); n.in[s.Post] = filter(n.in[s.Post])
}
