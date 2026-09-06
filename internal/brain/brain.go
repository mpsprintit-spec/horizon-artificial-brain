package brain

import (
	"math"

	"github.com/project-horizon/horizon-core/internal/neural"
)

// Brain is the single persistent intelligence substrate. It deliberately has
// no semantic vocabulary, rule tables, intent classifier, or fact store.
type Brain struct {
	net   *neural.Network
	input []neural.NeuronID
	read  []neural.NeuronID
}

func New(inputSize, readoutSize int, cfg neural.Config) (*Brain, error) {
	if inputSize <= 0 || readoutSize <= 0 { return nil, neural.ErrInvalidConfig }
	n,err:=neural.New(cfg); if err!=nil{return nil,err}
	input:=make([]neural.NeuronID,inputSize); read:=make([]neural.NeuronID,readoutSize)
	for i:=range input { input[i],err=n.AddNeuron(); if err!=nil{return nil,err} }
	for i:=range read { read[i],err=n.AddNeuron(); if err!=nil{return nil,err} }
	// Dense initialization is restricted to the declared substrate interface;
	// subsequent structure can be adapted by activity-dependent plasticity.
	for _,a:=range input { for _,b:=range read { if _,err=n.Connect(a,b,0.01);err!=nil{return nil,err} } }
	return &Brain{net:n,input:input,read:read},nil
}

func (b *Brain) Step(experience []float64, reward float64) ([]float64,error) {
	if len(experience)!=len(b.input){return nil,neural.ErrInvalidConfig}
	for i,v:=range experience { if math.IsNaN(v)||math.IsInf(v,0){return nil,neural.ErrInvalidConfig}; if err:=b.net.Inject(b.input[i],v);err!=nil{return nil,err} }
	b.net.Tick(); b.net.Learn(reward)
	out:=make([]float64,len(b.read)); for i,id:=range b.read { x,_:=b.net.Neuron(id); out[i]=x.Activity }
	return out,nil
}

func (b *Brain) Network() *neural.Network { return b.net }
