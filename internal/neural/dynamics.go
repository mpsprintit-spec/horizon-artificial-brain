package neural

import "math"

func (n *Network) Inject(id NeuronID, current float64) error { n.mu.Lock(); defer n.mu.Unlock(); x,ok:=n.neurons[id];if !ok{return ErrInvalidID};if math.IsNaN(current)||math.IsInf(current,0){return ErrInvalidConfig};x.Potential+=current;return nil }

// Tick advances the persistent recurrent state. Previous activity contributes
// to propagation, while current state is retained for the next integration.
func (n *Network) Tick() {
	n.mu.Lock(); defer n.mu.Unlock(); n.step++
	input:=make(map[NeuronID]float64,len(n.neurons))
	for _,s:=range n.synapses { pre:=n.neurons[s.Pre];if pre.Fired{input[s.Post]+=s.Weight*pre.Activity} }
	for id,x:=range n.neurons { x.Age++;x.Potential=(1-n.cfg.Leak)*x.Potential+input[id];x.Fired=x.Potential>=x.Homeostatic;if x.Fired{x.Activity=1;x.Trace=1}else{x.Activity*=0.75;x.Trace*=n.cfg.TraceDecay};x.Homeostatic=clamp(x.Homeostatic+n.cfg.HomeostasisRate*(x.Activity-0.1),n.cfg.Threshold*0.5,n.cfg.Threshold*2);if x.Fired{x.Potential=n.cfg.Reset} }
	for _,s:=range n.synapses { pre,post:=n.neurons[s.Pre],n.neurons[s.Post];s.Trace=s.Trace*n.cfg.TraceDecay+pre.Activity*post.Activity;s.Eligibility=s.Eligibility*n.cfg.TraceDecay+pre.Trace*post.Activity;s.Usage=s.Usage*0.999+pre.Activity*post.Activity;s.Age++ }
	active:=make([]NeuronID,0)
	for id,x:=range n.neurons{if x.Activity>0.5{active=append(active,id)}}
	for i,a:=range active{for j:=i+1;j<len(active);j++{key:=[2]NeuronID{a,active[j]};n.coactivity[key]=n.coactivity[key]*0.999+1}}
}

func (n *Network) SetActivity(id NeuronID, activity float64) error { n.mu.Lock();defer n.mu.Unlock();x,ok:=n.neurons[id];if !ok{return ErrInvalidID};if math.IsNaN(activity)||math.IsInf(activity,0){return ErrInvalidConfig};x.Activity=clamp(activity,0,1);x.Fired=x.Activity>0;if x.Fired{x.Trace=1};return nil }
