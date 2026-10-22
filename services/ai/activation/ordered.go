package activation

import (
	"strings"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func (e *Engine) activateOrderedWith(req Request) Result {
	if req.Cycles < 1 {
		req.Cycles = 1
	}
	if req.Now.IsZero() {
		req.Now = time.Now().UTC()
	}
	if req.DynamicsEncoding != nil && req.DynamicsEncoding.HasPhysicalMemory() {
		e.Memory.ApplyMemoryDynamicsWithEncoding(req.Now, *req.DynamicsEncoding)
	}
	trace := req.NeuralTrace.Canonical()
	lexical := false
	if trace == nil {
		trace = &knowledge.ObservationTrace{}
		if len(req.StimulusNodeIDs) > 0 {
			// A raw population is one simultaneous frame, never a sequence
			// inferred from the incidental order of its member IDs.
			trace.Stimulus = [][]knowledge.NodeID{append([]knowledge.NodeID(nil), req.StimulusNodeIDs...)}
		} else {
			lexical = true
			encoding := knowledge.WrittenTextEncoding
			if req.SurfaceEncoding != nil {
				encoding = *req.SurfaceEncoding
			}
			for _, token := range req.StimulusTokens {
				canonical := strings.TrimSpace(token)
				if canonical == "" {
					continue
				}
				if node := e.Memory.Registry.Get(canonical); node != nil {
					trace.Stimulus = append(trace.Stimulus, []knowledge.NodeID{node.ID})
					continue
				}
				grounded, err := e.Memory.GroundObservationWithEncodingAt(canonical, "activation", "language", e.Threshold, req.Now, encoding)
				if err == nil {
					trace.Stimulus = append(trace.Stimulus, grounded.Population)
				}
			}
		}
		trace = trace.Canonical()
	}
	e.mu.RLock()
	previousPrediction := cloneState(e.lastPrediction)
	previousTrace := e.lastTrace.Clone()
	context := e.contextState.Clone()
	previousObservation := e.observationPrediction.Clone()
	e.mu.RUnlock()
	observedFeedback := req.DynamicsEncoding != nil && req.DynamicsEncoding.HasObservedFeedback()
	recurrent := req.DynamicsEncoding != nil && req.DynamicsEncoding.HasRecurrentContext()
	resetContext := false
	if recurrent {
		context, resetContext = context.prepare(req.EpisodeID, req.ResetContext, req.Now)
		if resetContext {
			previousPrediction = nil
			previousObservation = nil
		}
	}
	if observedFeedback && previousObservation != nil && req.Now.Before(previousObservation.At) {
		previousObservation = nil
		previousPrediction = nil
		resetContext = true
		context = &ContextState{EpisodeID: req.EpisodeID}
	}
	if observedFeedback {
		previousPrediction = nil
		if previousObservation != nil {
			previousPrediction = cloneState(previousObservation.Prediction)
		}
	}
	if req.PredictionOverride != nil {
		previousPrediction = cloneState(req.PredictionOverride.State)
		// Explicit action forecasts are evaluated against direct outcome input.
		// Their bound consequence path owns causal credit, not an unrelated cue.
		if observedFeedback {
			previousObservation = nil
		}
	}
	if !recurrent && (req.CarryPreviousContext || lexical) && len(trace.Context) == 0 && len(trace.Stimulus) > 0 && previousTrace != nil {
		trace.Context = knowledge.ClonePopulationFrames(previousTrace.Stimulus)
	}
	if len(trace.Stimulus)+len(trace.Context)+len(trace.Data) == 0 {
		if recurrent {
			e.mu.Lock()
			e.contextState = context
			if !observedFeedback || resetContext {
				e.lastTrace = nil
			}
			if resetContext {
				e.lastPrediction = nil
				e.lastPredictionConf = nil
				e.observationPrediction = nil
			}
			e.mu.Unlock()
		}
		if observedFeedback {
			return e.ThinkWithContextDynamicsAt(req.Cycles, req.ContextBoosts, req.Now, *req.DynamicsEncoding).Result
		}
		return e.activateLegacyWith(req)
	}
	state, confidence := map[knowledge.NodeID]float64{}, map[knowledge.NodeID]float64{}
	for id, boost := range req.ContextBoosts {
		state[id] += boost
		confidence[id] = max(confidence[id], boost)
	}
	step := 0
	// Context/data are supplied evidence streams; the current stimulus follows
	// them. These are recurrent microsteps inside one event, not invented
	// physical timestamps or additional externally visible revisions.
	for _, stream := range [][][]knowledge.NodeID{trace.Context, trace.Data, trace.Stimulus} {
		for _, frame := range stream {
			if step > 0 {
				state, confidence = e.advance(state, confidence, req.Now, 1)
			}
			for _, id := range frame {
				if node := e.Memory.Registry.GetByID(id); node != nil {
					state[id] = max(state[id], 1)
					prior := .5 * clamp01(node.Importance)
					if lexical {
						prior = 1
					}
					confidence[id] = max(confidence[id], prior)
				}
			}
			step++
		}
	}
	state = normalize(state)
	pre := cloneState(state)
	state, confidence = e.advance(state, confidence, req.Now, req.Cycles)
	result := e.converge(state, confidence, req.Now)
	actualEvidence := observedPopulation(trace)
	result.ObservedInput = observedFeedback && len(actualEvidence) > 0
	if observedFeedback {
		e.ApplyActivityPlasticityWithDynamics(actualEvidence, actualEvidence, req.Now, *req.DynamicsEncoding)
	} else {
		e.ApplyActivityPlasticityWithDynamics(pre, result.Activations, req.Now, *req.DynamicsEncoding)
	}
	predictionError := 0.0
	if observedFeedback {
		predictionError = e.applyObservedFeedback(previousObservation, previousPrediction, actualEvidence, req.Now)
		if req.PredictionOverride != nil {
			predictionError = stateDifference(previousPrediction, actualEvidence)
		}
		if predictionError > 0 {
			e.Memory.ApplyPredictionErrorRepresentationPlasticity(previousPrediction, actualEvidence, predictionError, .20, req.Now)
		}
	} else if len(previousPrediction) > 0 {
		predictionError = stateDifference(previousPrediction, result.Activations)
		e.ApplyPredictionErrorPlasticityWithDynamics(previousPrediction, result.Activations, predictionError, req.Now, *req.DynamicsEncoding)
		e.Memory.ApplyPredictionErrorRepresentationPlasticity(previousPrediction, result.Activations, predictionError, .20, req.Now)
	}
	if observedFeedback {
		e.Memory.UpdateInformationDynamics(previousPrediction, actualEvidence, predictionError, req.Now)
	} else {
		e.Memory.UpdateInformationDynamics(previousPrediction, result.Activations, predictionError, req.Now)
	}
	next, nextConfidence := e.advance(result.Activations, result.Confidence, req.Now, req.Cycles)
	next, nextConfidence = e.applyPatternPrediction(next, nextConfidence, result.Activations)
	var patternCredit map[knowledge.PatternID]float64
	if observedFeedback {
		channels := e.Memory.Patterns.PredictRecurrent(trace, context.before(req.Now, false), req.Now)
		next, nextConfidence = e.applyPatternChannels(next, nextConfidence, channels)
		patternCredit = channels.Credit
		context.observe(trace, req.Now)
	} else if recurrent {
		next, nextConfidence = e.applyRecurrentPatternPrediction(next, nextConfidence, trace, context, req.Now, false)
		context.observe(trace, req.Now)
	} else {
		next, nextConfidence = e.applyOrderedPatternPrediction(next, nextConfidence, trace)
	}
	e.mu.Lock()
	e.internalState, e.internalConfidence = cloneState(result.Activations), cloneState(result.Confidence)
	e.lastPrediction, e.lastPredictionConf = cloneState(next), cloneState(nextConfidence)
	e.lastTrace = trace.Clone()
	if recurrent {
		e.contextState = context
	}
	if observedFeedback {
		e.observationPrediction = nil
		if len(actualEvidence) > 0 {
			e.observationPrediction = &ObservationPrediction{EpisodeID: req.EpisodeID, At: req.Now, Source: cloneState(actualEvidence), Prediction: cloneState(next), Patterns: patternCredit}
		}
	}
	e.mu.Unlock()
	result.PredictionError = predictionError
	return result
}

func (e *Engine) applyOrderedPatternPrediction(prediction, confidence map[knowledge.NodeID]float64, trace *knowledge.ObservationTrace) (map[knowledge.NodeID]float64, map[knowledge.NodeID]float64) {
	if e == nil || e.Memory == nil || e.Memory.Patterns == nil || trace == nil {
		return prediction, confidence
	}
	channels := e.Memory.Patterns.PredictOrdered(trace)
	return e.applyPatternChannels(prediction, confidence, channels)
}

func (e *Engine) applyPatternChannels(prediction, confidence map[knowledge.NodeID]float64, channels knowledge.OrderedPatternPrediction) (map[knowledge.NodeID]float64, map[knowledge.NodeID]float64) {
	if len(channels.Competitors) == 0 {
		return prediction, confidence
	}
	out, outConfidence := cloneState(prediction), cloneState(confidence)
	for _, id := range channels.Competitors {
		if e.Memory.Registry.GetByID(id) == nil {
			continue
		}
		support := channels.Support[id]
		out[id] = out[id]*support + channels.Strength[id]
		outConfidence[id] = max(outConfidence[id]*support, channels.Strength[id])
	}
	return normalize(out), normalize(outConfidence)
}
