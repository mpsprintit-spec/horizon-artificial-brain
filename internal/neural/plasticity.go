package neural

import "math"

func (n *Network) Learn(reward float64) {
	n.mu.Lock(); defer n.mu.Unlock(); if math.IsNaN(reward)||math.IsInf(reward,0){return};reward=clamp(reward,-1,1)
	for _,s:=range n.synapses { pre,post:=n.neurons[s.Pre],n.neurons[s.Post];dw:=n.cfg.LearningRate*reward*s.Eligibility+n.cfg.LearningRate*0.1*(pre.Activity*post.Activity-0.05*s.Usage);s.Weight=clamp((s.Weight+dw)*(1-n.cfg.SynapseDecay),n.cfg.MinWeight,n.cfg.MaxWeight) }
	n.normalizeIncomingLocked()
}

func (n *Network) normalizeIncomingLocked(){for _,ids:=range n.in{var norm float64;for _,id:=range ids{if s:=n.synapses[id];s!=nil{norm+=s.Weight*s.Weight}};if norm==0{continue};norm=math.Sqrt(norm);limit:=math.Sqrt(float64(len(ids)))*n.cfg.MaxWeight*0.5;if norm>limit{scale:=limit/norm;for _,id:=range ids{if s:=n.synapses[id];s!=nil{s.Weight*=scale}}}}}

// StructuralAdaptation changes topology only after sustained coactivity. The
// coactivity trace is an emergent property of activity, never a semantic key.
func (n *Network) StructuralAdaptation(){
	n.mu.Lock();defer n.mu.Unlock()
	if len(n.synapses)<n.cfg.MaxSynapses { for pair,score:=range n.coactivity { if score<10 {continue};if n.hasConnectionLocked(pair[0],pair[1])||n.hasConnectionLocked(pair[1],pair[0]){continue};w:=n.cfg.StructuralRate*score;if w>n.cfg.MaxWeight*0.1{w=n.cfg.MaxWeight*0.1};n.nextS++;s:=&Synapse{ID:n.nextS,Pre:pair[0],Post:pair[1],Weight:w};n.synapses[s.ID]=s;n.out[s.Pre]=append(n.out[s.Pre],s.ID);n.in[s.Post]=append(n.in[s.Post],s.ID);if len(n.synapses)>=n.cfg.MaxSynapses{break} } }
	for id,s:=range n.synapses{if s.Age>100&&s.Usage<0.001{n.removeSynapseLocked(id)}}
}

func (n *Network) hasConnectionLocked(pre,post NeuronID)bool{for _,id:=range n.out[pre]{if s:=n.synapses[id];s!=nil&&s.Post==post{return true}};return false}
func (n *Network) removeSynapseLocked(id SynapseID){s,ok:=n.synapses[id];if !ok{return};delete(n.synapses,id);filter:=func(xs []SynapseID)[]SynapseID{out:=xs[:0];for _,x:=range xs{if x!=id{out=append(out,x)}};return out};n.out[s.Pre]=filter(n.out[s.Pre]);n.in[s.Post]=filter(n.in[s.Post])}
