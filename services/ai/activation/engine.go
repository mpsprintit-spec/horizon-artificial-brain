package activation

import (
	"sort"
	"sync"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

type Engine struct {
	Memory     *knowledge.Brain
	Decay      float64
	SpreadRate float64
	Threshold  float64
	Inhibition float64

	mu                 sync.RWMutex
	internalState      map[knowledge.NodeID]float64
	internalConfidence map[knowledge.NodeID]float64
	lastPrediction     map[knowledge.NodeID]float64
	lastPredictionConf map[knowledge.NodeID]float64
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

func NewEngine(memory *knowledge.Brain) *Engine {
	return &Engine{
		Memory:             memory,
		Decay:              0.12,
		SpreadRate:         0.65,
		Threshold:          0.25,
		Inhibition:         0.55,
		internalState:      map[knowledge.NodeID]float64{},
		internalConfidence: map[knowledge.NodeID]float64{},
		lastPrediction:     map[knowledge.NodeID]float64{},
		lastPredictionConf: map[knowledge.NodeID]float64{},
	}
}

func (e *Engine) Activate(tokens []string, cycles int) Result {
	return e.ActivateWith(Request{StimulusTokens: tokens, Cycles: cycles, Now: time.Now().UTC()})
}

// Think advances the brain's internal dynamics without requiring a new
// external stimulus. Prediction is kept as an expectation about the next
// state; it is evaluated when the next actual state arrives.
func (e *Engine) Think(cycles int) Result {
	return e.ThinkWithPrediction(cycles).Result
}

// ThinkWithPrediction performs one internal transition, then predicts the
// following transition. Keeping prediction separate from the actual state is
// essential: an external experience can later violate the prediction and feed
// a measurable prediction error into plasticity.
func (e *Engine) ThinkWithPrediction(cycles int) ThoughtResult {
	if cycles < 1 {
		cycles = 1
	}
	now := time.Now().UTC()

	e.mu.RLock()
	state := cloneState(e.internalState)
	confidence := cloneState(e.internalConfidence)
	previousPrediction := cloneState(e.lastPrediction)
	e.mu.RUnlock()

	if len(state) == 0 {
		return ThoughtResult{Result: Result{
			Activations: map[knowledge.NodeID]float64{},
			Confidence:  map[knowledge.NodeID]float64{},
		}}
	}

	actualState, actualConfidence := e.advance(state, confidence, now, cycles)
	actual := e.converge(actualState, actualConfidence, now)

	// Plasticity observes the actual transition that just occurred. This is
	// intentionally independent from language tokens or symbolic relations.
	e.ApplyActivityPlasticity(state, actual.Activations, now)

	error := 0.0
	if len(previousPrediction) > 0 {
		error = stateDifference(previousPrediction, actual.Activations)
		e.ApplyPredictionErrorPlasticity(previousPrediction, actual.Activations, error, now)
	}

	nextPrediction, nextPredictionConfidence := e.advance(actual.Activations, actual.Confidence, now, cycles)

	e.mu.Lock()
	e.internalState = cloneState(actual.Activations)
	e.internalConfidence = cloneState(actual.Confidence)
	e.lastPrediction = cloneState(nextPrediction)
	e.lastPredictionConf = cloneState(nextPredictionConfidence)
	e.mu.Unlock()

	return ThoughtResult{
		Result:          actual,
		Prediction:      Prediction{State: nextPrediction, Confidence: nextPredictionConfidence},
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

	e.mu.RLock()
	previousPrediction := cloneState(e.lastPrediction)
	e.mu.RUnlock()

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

	preState := cloneState(state)
	state, confidence = e.advance(state, confidence, req.Now, req.Cycles)
	result := e.converge(state, confidence, req.Now)

	e.ApplyActivityPlasticity(preState, result.Activations, req.Now)

	if len(previousPrediction) > 0 {
		error := stateDifference(previousPrediction, result.Activations)
		e.ApplyPredictionErrorPlasticity(previousPrediction, result.Activations, error, req.Now)
	}

	e.mu.Lock()
	e.internalState = cloneState(result.Activations)
	e.internalConfidence = cloneState(result.Confidence)
	e.lastPrediction = map[knowledge.NodeID]float64{}
	e.lastPredictionConf = map[knowledge.NodeID]float64{}
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
		n.Activation = clamp01(level)
		n.LastActivation = now
		if level > e.Threshold {
			n.Frequency++
		}
		resonance += level * confidence[id]
		if level >= e.Threshold {
			ranked = append(ranked, n)
		}
	}
	sort.Slice(ranked, func(i, j int) bool {
		a, b := ranked[i], ranked[j]
		si := a.Activation*confidence[a.ID] + a.Importance*0.1 + float64(a.Frequency)*0.001
		sj := b.Activation*confidence[b.ID] + b.Importance*0.1 + float64(b.Frequency)*0.001
		if si == sj {
			return a.Token < b.Token
		}
		return si > sj
	})
	return Result{Converged: true, Resonance: resonance, Activations: cloneState(state), Confidence: cloneState(confidence), RankedNodes: ranked}
}
