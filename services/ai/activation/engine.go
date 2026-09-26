package activation

import (
	"sort"
	"strings"
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

type Request struct {
	StimulusTokens []string
	StimulusNodeIDs []knowledge.NodeID
	ContextBoosts map[knowledge.NodeID]float64
	Cycles int
	Now time.Time
}
type Result struct { Converged bool; Resonance float64; Activations map[knowledge.NodeID]float64; Confidence map[knowledge.NodeID]float64; RankedNodes []*knowledge.ConceptNode; PredictionError float64 }
type Prediction struct { State, Confidence map[knowledge.NodeID]float64 }
type ThoughtResult struct { Result; Prediction Prediction; PredictionError float64 }

func NewEngine(memory *knowledge.Brain) *Engine { return &Engine{Memory:memory, Decay:.12, SpreadRate:.65, Threshold:.25, Inhibition:.55, internalState:map[knowledge.NodeID]float64{}, internalConfidence:map[knowledge.NodeID]float64{}, lastPrediction:map[knowledge.NodeID]float64{}, lastPredictionConf:map[knowledge.NodeID]float64{}} }
func (e *Engine) Activate(tokens []string, cycles int) Result { return e.ActivateWith(Request{StimulusTokens:tokens,Cycles:cycles,Now:time.Now().UTC()}) }
func (e *Engine) Think(cycles int) Result { return e.ThinkWithPrediction(cycles).Result }
func (e *Engine) ThinkWithPrediction(cycles int) ThoughtResult { return e.ThinkWithPredictionAt(cycles,time.Now().UTC()) }

func (e *Engine) ActivateWith(req Request) Result {
	if req.Cycles<1 { req.Cycles=1 }; if req.Now.IsZero(){req.Now=time.Now().UTC()}
	e.mu.RLock(); previousPrediction:=cloneState(e.lastPrediction); e.mu.RUnlock()
	state:=map[knowledge.NodeID]float64{}; conf:=map[knowledge.NodeID]float64{}
	// Grounded node IDs are the canonical substrate input path. This lets
	// vision/audio/touch observations enter cognition without converting them
	// back into lexical lookup.
	for _, id := range req.StimulusNodeIDs {
		if node := e.Memory.Registry.GetByID(id); node != nil {
			state[id] = max(state[id], 1)
			conf[id] = max(conf[id], 1)
		}
	}
	// Once a modality adapter has supplied grounded substrate IDs, they are
	// authoritative for this event. Do not re-encode the same surface tokens
	// as a second modality population.
	if len(req.StimulusNodeIDs) > 0 {
		req.StimulusTokens = nil
	}
	for _,token:=range req.StimulusTokens {
		canonical := strings.TrimSpace(token)
		if canonical == "" {
			continue
		}
		// Language activation is an adapter boundary: when a canonical
		// language representation already exists, reactivate that exact unit.
		// Do not turn every repeated linguistic stimulus into a new distributed
		// population. Unknown non-canonical numeric experience is still allowed
		// to enter through the distributed projection path.
		if node := e.Memory.Registry.Get(canonical); node != nil {
			level := 1.0
			state[node.ID] = max(state[node.ID], level)
			conf[node.ID] = max(conf[node.ID], level)
			continue
		}
		population, err := e.Memory.ProjectVectorPopulation(knowledge.EncodeObservation(canonical, "language"), e.Threshold, 4)
		if err != nil {
			continue
		}
		for _, unit := range population.Units {
			level := clamp01(unit.Activation)
			if level <= 0 {
				continue
			}
			state[unit.NodeID] = max(state[unit.NodeID], level)
			conf[unit.NodeID] = max(conf[unit.NodeID], level)
		}
	}
	for id,b:=range req.ContextBoosts { state[id]+=b; conf[id]=max(conf[id],b) }
	state=normalize(state); pre:=cloneState(state); state,conf=e.advance(state,conf,req.Now,req.Cycles); result:=e.converge(state,conf,req.Now)
	e.ApplyActivityPlasticity(pre,result.Activations,req.Now)

	predictionError:=0.0
	if len(previousPrediction)>0 {
		predictionError=stateDifference(previousPrediction,result.Activations)
		e.ApplyPredictionErrorPlasticity(previousPrediction,result.Activations,predictionError,req.Now)
	}
	nextPrediction, nextPredictionConfidence := e.advance(result.Activations,result.Confidence,req.Now,req.Cycles)
	nextPrediction, nextPredictionConfidence = e.applyPatternPrediction(nextPrediction, nextPredictionConfidence, result.Activations)

	e.mu.Lock()
	e.internalState=cloneState(result.Activations)
	e.internalConfidence=cloneState(result.Confidence)
	e.lastPrediction=cloneState(nextPrediction)
	e.lastPredictionConf=cloneState(nextPredictionConfidence)
	e.mu.Unlock()
	result.PredictionError=predictionError
	return result
}

func (e *Engine) applyPatternPrediction(prediction, confidence, actual map[knowledge.NodeID]float64) (map[knowledge.NodeID]float64, map[knowledge.NodeID]float64) {
	if e == nil || e.Memory == nil || e.Memory.Patterns == nil || len(actual) == 0 {
		return prediction, confidence
	}
	out := cloneState(prediction)
	outConfidence := cloneState(confidence)
	for _, id := range sortedNodeIDs(actual) {
		if actual[id] <= e.Threshold { continue }
		cue := []knowledge.PatternStep{{NodeID: id, Position: 0, Activation: actual[id]}}
		matches := e.Memory.Patterns.CompleteTrace(cue, nil)
		if len(matches) == 0 { continue }
		limit := len(matches)
		if limit > 3 { limit = 3 }
		for _, pattern := range matches[:limit] {
			if pattern == nil || len(pattern.Sequence) == 0 || pattern.Result == id { continue }
			strength := clamp01(pattern.Weight) * clamp01(pattern.Confidence)
			frequency := float64(pattern.Frequency)
			if frequency > 10 { frequency = 10 }
			strength *= frequency / 10
			if strength <= 0 { continue }
			if strength > 0.35 { strength = 0.35 }
			out[pattern.Result] = max(out[pattern.Result], strength)
			outConfidence[pattern.Result] = max(outConfidence[pattern.Result], strength)
		}
	}
	return normalize(out), normalize(outConfidence)
}


func (e *Engine) advance(state, confidence map[knowledge.NodeID]float64, now time.Time, cycles int) (map[knowledge.NodeID]float64,map[knowledge.NodeID]float64) {
	if e == nil || e.Memory == nil { return cloneState(state), cloneState(confidence) }
	e.Memory.Lock()
	defer e.Memory.Unlock()
	state=cloneState(state); confidence=cloneState(confidence)
	for i:=0;i<cycles;i++ { next:=map[knowledge.NodeID]float64{}; nextConfidence:=map[knowledge.NodeID]float64{}
		for _,id:=range sortedNodeIDs(state) { n:=e.Memory.Registry.GetByID(id); if n==nil{continue}; adaptive:=n.Threshold-(n.Importance*.05)-(float64(n.Frequency)*.001); n.Threshold=clamp(adaptive,.12,.8); next[id]=n.RestingActivation }
		for _,id:=range sortedNodeIDs(state) { level:=state[id]; n:=e.Memory.Registry.GetByID(id); if n==nil{continue}; next[id]+=level*(1-e.Decay); nextConfidence[id]=max(nextConfidence[id],confidence[id]); synapses:=n.OutboundAll(); sort.Slice(synapses,func(i,j int)bool{ return synapses[i].TargetID<synapses[j].TargetID }); for _,s:=range synapses { age:=temporalPenalty(now,s.LastActivation); pulse:=level*s.Weight*s.Confidence*e.SpreadRate*age; if s.Inhibitory{next[s.TargetID]-=pulse*e.Inhibition}else{next[s.TargetID]+=pulse}; nextConfidence[s.TargetID]=max(nextConfidence[s.TargetID],confidence[id]*s.Confidence*age); s.Activation=pulse } }
		state=normalize(next); confidence=normalize(nextConfidence)
	}
	return state,confidence
}

func (e *Engine) converge(state,confidence map[knowledge.NodeID]float64,now time.Time) Result {
	if e == nil || e.Memory == nil { return Result{} }
	e.Memory.Lock()
	defer e.Memory.Unlock()
	var ranked []*knowledge.ConceptNode; var resonance float64; for _,id:=range sortedNodeIDs(state){level:=state[id]; n:=e.Memory.Registry.GetByID(id); if n==nil{continue}; n.Activation=clamp01(level); n.LastActivation=now; if level>e.Threshold{n.Frequency++}; resonance+=level*confidence[id]; if level>=e.Threshold{ranked=append(ranked,n)} }; sort.Slice(ranked,func(i,j int)bool{a,b:=ranked[i],ranked[j]; si:=a.Activation*confidence[a.ID]+a.Importance*.1+float64(a.Frequency)*.001; sj:=b.Activation*confidence[b.ID]+b.Importance*.1+float64(b.Frequency)*.001; if si==sj{return a.ID<b.ID}; return si>sj}); return Result{Converged:true,Resonance:resonance,Activations:cloneState(state),Confidence:cloneState(confidence),RankedNodes:ranked} }
