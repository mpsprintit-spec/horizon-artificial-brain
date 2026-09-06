package neural

import (
	"encoding/json"
	"os"
	"sort"
)

type snapshot struct {
	Version uint32 `json:"version"`
	Step uint64 `json:"step"`
	NextN NeuronID `json:"next_neuron_id"`
	NextS SynapseID `json:"next_synapse_id"`
	Config Config `json:"config"`
	Neurons []*Neuron `json:"neurons"`
	Synapses []*Synapse `json:"synapses"`
}

func (n *Network) Save(path string) error {
	n.mu.RLock(); defer n.mu.RUnlock()
	s := snapshot{Version:1,Step:n.step,NextN:n.nextN,NextS:n.nextS,Config:n.cfg}
	nids:=make([]NeuronID,0,len(n.neurons)); for id:=range n.neurons { nids=append(nids,id) }; sort.Slice(nids,func(i,j int)bool{return nids[i]<nids[j]})
	s.Neurons=make([]*Neuron,0,len(n.neurons)); for _,id:=range nids { y:=*n.neurons[id]; s.Neurons=append(s.Neurons,&y) }
	sids:=make([]SynapseID,0,len(n.synapses)); for id:=range n.synapses { sids=append(sids,id) }; sort.Slice(sids,func(i,j int)bool{return sids[i]<sids[j]})
	s.Synapses=make([]*Synapse,0,len(n.synapses)); for _,id:=range sids { y:=*n.synapses[id]; s.Synapses=append(s.Synapses,&y) }
	data,err:=json.Marshal(s); if err!=nil{return err}
	tmp:=path+".tmp"
	if err=os.WriteFile(tmp,data,0600); err!=nil{return err}
	if err=os.Rename(tmp,path); err!=nil { _=os.Remove(tmp); return err }
	return nil
}

func Load(path string) (*Network,error) {
	data,err:=os.ReadFile(path); if err!=nil{return nil,err}
	var s snapshot; if err=json.Unmarshal(data,&s); err!=nil{return nil,err}
	if s.Version!=1{return nil,ErrInvalidConfig}
	n,err:=New(s.Config); if err!=nil{return nil,err}
	n.step,n.nextN,n.nextS=s.Step,s.NextN,s.NextS
	for _,x:=range s.Neurons { if x==nil || x.ID==0 || len(n.neurons)>=n.cfg.MaxNeurons{return nil,ErrInvalidConfig}; if _,exists:=n.neurons[x.ID];exists{return nil,ErrInvalidConfig}; y:=*x; n.neurons[y.ID]=&y }
	for _,x:=range s.Synapses { if x==nil || x.ID==0{return nil,ErrInvalidConfig}; if _,ok:=n.neurons[x.Pre];!ok{return nil,ErrInvalidID}; if _,ok:=n.neurons[x.Post];!ok{return nil,ErrInvalidID}; if _,exists:=n.synapses[x.ID];exists{return nil,ErrInvalidConfig}; y:=*x; y.Weight=clamp(y.Weight,n.cfg.MinWeight,n.cfg.MaxWeight); n.synapses[y.ID]=&y; n.out[y.Pre]=append(n.out[y.Pre],y.ID); n.in[y.Post]=append(n.in[y.Post],y.ID) }
	return n,nil
}
