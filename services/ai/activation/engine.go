package activation

import (
	"sort"
	"sync"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

type Engine struct {
	Memory *knowledge.Brain
	Decay, SpreadRate, Threshold, Inhibition float64
	mu sync.RWMutex
	internalState, internalConfidence map[knowledge.NodeID]float64
	lastPrediction, lastPredictionConf map[knowledge.NodeID]float64
}

type Request struct { StimulusTokens []string; ContextBoosts map[knowledge.NodeID]float64; Cycles int; Now time.Time }
type Result struct { Converged bool; Resonance float64; Activations map[knowledge.NodeID]float64; Confidence map[knowledge.NodeID]float64; RankedNodes []*knowledge.ConceptNode }
type Prediction struct { State, Confidence map[knowledge.NodeID]float64 }
type ThoughtResult struct { Result; Prediction Prediction; PredictionError float64 }

func NewEngine(memory *knowledge.Brain) *Engine { return &Engine{Memory:memory, Decay:.12, SpreadRate:.65, Threshold:.25, Inhibition:.55, internalState:map[knowledge.NodeID]float64{}, internalConfidence:map[knowledge.NodeID]float64{}, lastPrediction:map[knowledge.NodeID]float64{}, lastPredictionConf:map[knowledge.NodeID]float64{}} }
func (e *Engine) Activate(tokens []string, cycles int) Result { return e.ActivateWith(Request{StimulusTokens:tokens,Cycles:cycles,Now:time.Now().UTC()}) }
func (e *Engine) Think(cycles int) Result { return e.ThinkWithPrediction(cycles).Result }
func (e *Engine) ThinkWithPrediction(cycles int) ThoughtResult { return e.ThinkWithPredictionAt(cycles,time.Now().UTC()) }

func (e *Engine) ActivateWith(req Request) Result {
	if req.Cycles<1 { req.Cycles=1 }; if req.Now.IsZero(){req.Now=time.Now().UTC()}
	e.mu.RLock(); previous:=cloneState(e.lastPrediction); e.mu.RUnlock()
	state:=map[knowledge.NodeID]float64{}; conf:=map[knowledge.NodeID]float64{}
	for _,token:=range req.StimulusTokens { if n:=e.Memory.Fetch(token); n!=nil { state[n.ID]=1; conf[n.ID]=1 } }
	for id,b:=range req.ContextBoosts { state[id]+=b; conf[id]=max(conf[id],b) }
	state=normalize(state); pre:=cloneState(state); state,conf=e.advance(state,conf,req.Now,req.Cycles); result:=e.converge(state,conf,req.Now)
	e.ApplyActivityPlasticity(pre,result.Activations,req.Now)
	if len(previous)>0 { err:=stateDifference(previous,result.Activations); e.ApplyPredictionErrorPlasticity(previous,result.Activations,err,req.Now) }
	e.mu.Lock(); e.internalState=cloneState(result.Activations); e.internalConfidence=cloneState(result.Confidence); e.lastPrediction=map[knowledge.NodeID]float64{}; e.lastPredictionConf=map[knowledge.NodeID]float64{}; e.mu.Unlock()
	return result
}

func (e *Engine) advance(state, confidence map[knowledge.NodeID]float64, now time.Time, cycles int) (map[knowledge.NodeID]float64,map[knowledge.NodeID]float64) {
	state=cloneState(state); confidence=cloneState(confidence)
	for i:=0;i<cycles;i++ { next:=map[knowledge.NodeID]float64{}; nextConfidence:=map[knowledge.NodeID]float64{}
		for _,id:=range sortedNodeIDs(state) { n:=e.Memory.Registry.GetByID(id); if n==nil{continue}; adaptive:=n.Threshold-(n.Importance*.05)-(float64(n.Frequency)*.001); n.Threshold=clamp(adaptive,.12,.8); next[id]=n.RestingActivation }
		for _,id:=range sortedNodeIDs(state) { level:=state[id]; n:=e.Memory.Registry.GetByID(id); if n==nil{continue}; next[id]+=level*(1-e.Decay); nextConfidence[id]=max(nextConfidence[id],confidence[id]); synapses:=n.OutboundAll(); sort.Slice(synapses,func(i,j int)bool{ return synapses[i].TargetID<synapses[j].TargetID }); for _,s:=range synapses { age:=temporalPenalty(now,s.LastActivation); pulse:=level*s.Weight*s.Confidence*e.SpreadRate*age; if s.Inhibitory{next[s.TargetID]-=pulse*e.Inhibition}else{next[s.TargetID]+=pulse}; nextConfidence[s.TargetID]=max(nextConfidence[s.TargetID],confidence[id]*s.Confidence*age); s.Activation=pulse } }
		state=normalize(next); confidence=normalize(nextConfidence)
	}
	return state,confidence
}

func (e *Engine) converge(state,confidence map[knowledge.NodeID]float64,now time.Time) Result { var ranked []*knowledge.ConceptNode; var resonance float64; for _,id:=range sortedNodeIDs(state){level:=state[id]; n:=e.Memory.Registry.GetByID(id); if n==nil{continue}; n.Activation=clamp01(level); n.LastActivation=now; if level>e.Threshold{n.Frequency++}; resonance+=level*confidence[id]; if level>=e.Threshold{ranked=append(ranked,n)} }; sort.Slice(ranked,func(i,j int)bool{a,b:=ranked[i],ranked[j]; si:=a.Activation*confidence[a.ID]+a.Importance*.1+float64(a.Frequency)*.001; sj:=b.Activation*confidence[b.ID]+b.Importance*.1+float64(b.Frequency)*.001; if si==sj{return a.ID<b.ID}; return si>sj}); return Result{Converged:true,Resonance:resonance,Activations:cloneState(state),Confidence:cloneState(confidence),RankedNodes:ranked} }
