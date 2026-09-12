package activation

import (
	"sort"
	"sync"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

type Engine struct {
	Memory     *knowledge.KnowledgeBase
	Decay      float64
	SpreadRate float64
	Threshold  float64
	Inhibition float64

	mu                 sync.RWMutex
	internalState      map[knowledge.NodeID]float64
	internalConfidence map[knowledge.NodeID]float64
}

type Request struct {
	StimulusTokens []string
	ContextBoosts  map[knowledge.NodeID]float64
	Cycles         int
	Now            time.Time
}

type Result struct {
	Converged   bool
	Resonance   float64
	Activations map[knowledge.NodeID]float64
	Confidence  map[knowledge.NodeID]float64
	RankedNodes []*knowledge.ConceptNode
}

type Prediction struct {
	State      map[knowledge.NodeID]float64
	Confidence map[knowledge.NodeID]float64
}

type ThoughtResult struct {
	Result
	Prediction      Prediction
	PredictionError float64
}

func NewEngine(memory *knowledge.KnowledgeBase) *Engine {
	return &Engine{
		Memory: memory,
		Decay: 0.12,
		SpreadRate: 0.65,
		Threshold: 0.25,
		Inhibition: 0.55,
		internalState: map[knowledge.NodeID]float64{},
		internalConfidence: map[knowledge.NodeID]float64{},
	}
}

func (e *Engine) Activate(tokens []string, cycles int) Result {
	return e.ActivateWith(Request{StimulusTokens: tokens, Cycles: cycles, Now: time.Now().UTC()})
}

// Think continues the brain's own internal dynamics without requiring a new
// external stimulus. The previous internal activation becomes the starting
// state for another recurrent activation cycle. It deliberately has no
// answer lookup, semantic rule table, or required output.
func (e *Engine) Think(cycles int) Result {
	return e.ThinkWithPrediction(cycles).Result
}

// ThinkWithPrediction performs one internal prediction step, then advances
// the actual internal state. Prediction is a state transition, not a lookup
// from a pattern to an answer. The prediction is intentionally kept separate
// from output generation so internal cognition can continue without output.
func (e *Engine) ThinkWithPrediction(cycles int) ThoughtResult {
	if cycles < 1 {
		cycles = 1
	}

	e.mu.RLock()
	state := cloneState(e.internalState)
	confidence := cloneState(e.internalConfidence)
	e.mu.RUnlock()

	if len(state) == 0 {
		return ThoughtResult{Result: Result{
			Activations: map[knowledge.NodeID]float64{},
			Confidence:  map[knowledge.NodeID]float64{},
		}}
	}

	predictionState, predictionConfidence := e.advance(state, confidence, time.Now().UTC(), cycles)
	actual := e.ActivateWith(Request{ContextBoosts: state, Cycles: cycles, Now: time.Now().UTC()})
	error := stateDifference(predictionState, actual.Activations)

	return ThoughtResult{
		Result:          actual,
		Prediction:      Prediction{State: predictionState, Confidence: predictionConfidence},
		PredictionError: error,
	}
}

func (e *Engine) ActivateWith(req Request) Result {
	if req.Cycles < 1 {
		req.Cycles = 1
	}
	if req.Now.IsZero() {
		req.Now = time.Now().UTC()
	}
	state := map[knowledge.NodeID]float64{}
	confidence := map[knowledge.NodeID]float64{}
	for _, token := range req.StimulusTokens {
		if n := e.Memory.Fetch(token); n != nil {
			state[n.ID] = 1
			confidence[n.ID] = 1
		}
	}
	for id, boost := range req.ContextBoosts {
		state[id] += boost
		confidence[id] = max(confidence[id], boost)
	}
	state = normalize(state)

	state, confidence = e.advance(state, confidence, req.Now, req.Cycles)
	result := e.converge(state, confidence, req.Now)
	e.mu.Lock()
	e.internalState = cloneState(state)
	e.internalConfidence = cloneState(confidence)
	e.mu.Unlock()
	return result
}

func (e *Engine) advance(state, confidence map[knowledge.NodeID]float64, now time.Time, cycles int) (map[knowledge.NodeID]float64, map[knowledge.NodeID]float64) {
	state = cloneState(state)
	confidence = cloneState(confidence)
	for i := 0; i < cycles; i++ {
		next := map[knowledge.NodeID]float64{}
		nextConfidence := map[knowledge.NodeID]float64{}
		for id := range state {
			n := e.Memory.Registry.GetByID(id)
			if n == nil {
				continue
			}
			adaptive := n.Threshold - (n.Importance * 0.05) - (float64(n.Frequency) * 0.001)
			n.Threshold = clamp(adaptive, 0.12, 0.8)
			next[id] = n.RestingActivation
		}
		for id, level := range state {
			n := e.Memory.Registry.GetByID(id)
			if n == nil {
				continue
			}
			next[id] += level * (1 - e.Decay)
			nextConfidence[id] = max(nextConfidence[id], confidence[id])
			for _, s := range n.OutboundAll() {
				agePenalty := temporalPenalty(now, s.LastActivation)
				pulse := level * s.Weight * s.Confidence * e.SpreadRate * agePenalty
				if s.Inhibitory {
					next[s.TargetID] -= pulse * e.Inhibition
				} else {
					next[s.TargetID] += pulse
				}
				nextConfidence[s.TargetID] = max(nextConfidence[s.TargetID], confidence[id]*s.Confidence*agePenalty)
				s.Activation = pulse
			}
		}
		state = normalize(next)
		confidence = normalize(nextConfidence)
	}
	return state, confidence
}

func (e *Engine) converge(state, confidence map[knowledge.NodeID]float64, now time.Time) Result {
	var ranked []*knowledge.ConceptNode
	var resonance float64
	for id, level := range state {
		n := e.Memory.Registry.GetByID(id)
		if n == nil {
			continue
		}
		n.Activation = level
		if level >= max(e.Threshold, n.Threshold) {
			n.Frequency++
			n.LastActivation = now
			ranked = append(ranked, n)
			resonance += level * max(confidence[id], 0.1)
		}
	}
	sort.Slice(ranked, func(i, j int) bool {
		left, right := ranked[i], ranked[j]
		if left.Activation != right.Activation {
			return left.Activation > right.Activation
		}
		if confidence[left.ID] != confidence[right.ID] {
			return confidence[left.ID] > confidence[right.ID]
		}
		if left.Importance != right.Importance {
			return left.Importance > right.Importance
		}
		if left.Frequency != right.Frequency {
			return left.Frequency > right.Frequency
		}
		return left.Token < right.Token
	})
	return Result{Converged: len(ranked) > 0, Resonance: resonance, Activations: state, Confidence: confidence, RankedNodes: ranked}
}

func temporalPenalty(now, last time.Time) float64 {
	if last.IsZero() {
		return 0.7
	}
	days := now.Sub(last).Hours() / 24
	if days <= 1 {
		return 1
	}
	return clamp(1-(days*0.01), 0.35, 1)
}

func normalize(in map[knowledge.NodeID]float64) map[knowledge.NodeID]float64 {
	for k, v := range in {
		in[k] = squash(v)
	}
	return in
}

func squash(v float64) float64 {
	if v <= 1 {
		return clamp(v, 0, 1)
	}
	return 1 + (v-1)/v
}

func clamp(v, low, high float64) float64 {
	if v < low {
		return low
	}
	if v > high {
		return high
	}
	return v
}

func cloneState(in map[knowledge.NodeID]float64) map[knowledge.NodeID]float64 {
	out := make(map[knowledge.NodeID]float64, len(in))
	for id, value := range in {
		out[id] = value
	}
	return out
}

func stateDifference(a, b map[knowledge.NodeID]float64) float64 {
	keys := make(map[knowledge.NodeID]struct{}, len(a)+len(b))
	for id := range a { keys[id] = struct{}{} }
	for id := range b { keys[id] = struct{}{} }
	if len(keys) == 0 { return 0 }
	var total float64
	for id := range keys { total += abs(a[id] - b[id]) }
	return clamp01(total / float64(len(keys)))
}

func abs(v float64) float64 {
	if v < 0 { return -v }
	return v
}

func clamp01(v float64) float64 { return clamp(v, 0, 1) }
