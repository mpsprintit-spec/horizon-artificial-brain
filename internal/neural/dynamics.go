package neural

import "math"

// Inject adds external current to a neuron. Inputs are transient activity,
// not symbolic facts or semantic labels.
func (n *Network) Inject(id NeuronID, current float64) error {
	n.mu.Lock(); defer n.mu.Unlock()
	x, ok := n.neurons[id]; if !ok { return ErrInvalidID }
	if math.IsNaN(current) || math.IsInf(current,0) { return ErrInvalidConfig }
	x.Potential += current
	return nil
}

// Tick advances the entire recurrent substrate by one discrete integration
// step. State is retained between ticks; no per-input reset occurs.
func (n *Network) Tick() {
	n.mu.Lock(); defer n.mu.Unlock()
	n.step++

	input := make(map[NeuronID]float64, len(n.neurons))
	for _, s := range n.synapses {
		pre := n.neurons[s.Pre]
		if pre.Fired { input[s.Post] += s.Weight * pre.Activity }
	}

	for id, x := range n.neurons {
		x.Age++
		x.Potential = (1-n.cfg.Leak)*x.Potential + input[id]
		x.Fired = x.Potential >= x.Homeostatic
		if x.Fired { x.Activity = 1; x.Trace = 1 } else { x.Activity *= 0.75; x.Trace *= n.cfg.TraceDecay }
		// Slow homeostatic target adaptation prevents permanently silent or
		// permanently saturated units without imposing symbolic semantics.
		target := n.cfg.Threshold
		err := x.Activity - 0.1
		x.Homeostatic += n.cfg.HomeostasisRate * err
		x.Homeostatic = clamp(x.Homeostatic, target*0.5, target*2)
		if x.Fired { x.Potential = n.cfg.Reset }
	}

	for _, s := range n.synapses {
		pre, post := n.neurons[s.Pre], n.neurons[s.Post]
		s.Trace = s.Trace*n.cfg.TraceDecay + pre.Activity*post.Activity
		s.Eligibility = s.Eligibility*n.cfg.TraceDecay + pre.Trace*post.Activity
		s.Usage = s.Usage*0.999 + pre.Activity*post.Activity
		s.Age++
	}
}

// SetActivity is intended for deterministic experiments and controlled
// replay. Production sensors should normally use Inject + Tick.
func (n *Network) SetActivity(id NeuronID, activity float64) error {
	n.mu.Lock(); defer n.mu.Unlock()
	x, ok := n.neurons[id]; if !ok { return ErrInvalidID }
	if math.IsNaN(activity) || math.IsInf(activity,0) { return ErrInvalidConfig }
	x.Activity = clamp(activity, 0, 1)
	x.Fired = x.Activity > 0
	if x.Fired { x.Trace = 1 }
	return nil
}
