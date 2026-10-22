package knowledge

import (
	"encoding/json"
	"slices"
	"sort"
)

// LearnOrderedTraceWithEvidence keeps the exact positional identity. Historical
// LearnTrace/Member-set semantics remain available for versioned replay.
func (p *PatternIndex) LearnOrderedTraceWithEvidence(trace *ObservationTrace, result NodeID, weight, confidence float64, evidence ExperienceEvidence) *PatternSynapse {
	trace = trace.Canonical()
	if p == nil || trace == nil || len(trace.Stimulus) == 0 || trace.Validate(nil) != nil {
		return nil
	}
	if !slices.Contains(trace.Stimulus[len(trace.Stimulus)-1], result) {
		return nil
	}
	sequence := TraceSteps(trace.Stimulus)
	sequence[len(sequence)-1].NodeID = result
	key := observationTraceKey(trace)
	p.mu.Lock()
	defer p.mu.Unlock()
	var existing *PatternSynapse
	for _, candidate := range p.patterns {
		if candidate.Dynamics == OrderedDynamics && candidate.Result == result && observationTraceKey(candidate.Observation) == key {
			if existing == nil || candidate.ID < existing.ID {
				existing = candidate
			}
		}
	}
	if existing != nil {
		frequency := max(int64(0), existing.Frequency)
		existing.Weight = clamp01((existing.Weight*float64(frequency) + weight) / float64(frequency+1))
		existing.Confidence = clamp01(1 - (1-existing.Confidence)*(1-confidence))
		existing.Sequence = mergeSequence(existing.Sequence, sequence, frequency)
		existing.Evidence = appendEvidence(existing.Evidence, evidence)
		existing.Frequency = frequency + 1
		return existing
	}
	members := make([]NodeID, len(sequence))
	for i, step := range sequence {
		members[i] = step.NodeID
	}
	pattern := &PatternSynapse{ID: p.nextID, Dynamics: OrderedDynamics, Members: members, Sequence: sequence, Observation: trace, Result: result, Weight: clamp01(weight), Confidence: clamp01(confidence), Frequency: 1, Evidence: appendEvidence(nil, evidence)}
	p.patterns[p.nextID] = pattern
	p.nextID++
	return pattern
}

type OrderedPatternPrediction struct {
	Support     map[NodeID]float64
	Strength    map[NodeID]float64
	Competitors []NodeID
	Credit map[PatternID]float64
}

func observationTraceKey(trace *ObservationTrace) string {
	data, _ := json.Marshal(trace)
	return string(data)
}

func populationMatches(expected, actual []NodeID) bool {
	for _, id := range expected {
		if !slices.Contains(actual, id) {
			return false
		}
	}
	return true
}

func orderedSuffixMatches(expected, frames [][]NodeID) bool {
	if len(expected) > len(frames) {
		return false
	}
	frames = frames[len(frames)-len(expected):]
	for i, frame := range expected {
		if !populationMatches(frame, frames[i]) {
			return false
		}
	}
	return true
}

func legacyStepFrames(steps []PatternStep) [][]NodeID {
	frames := make([][]NodeID, len(steps))
	for i, step := range steps {
		frames[i] = []NodeID{step.NodeID}
	}
	return frames
}

// Bag compatibility identifies competing learned channels, including reversed
// and repeated inputs. Bipartite assignment preserves occurrence counts while
// treating each frame as an unordered simultaneous neural population.
func orderedSuffixBagMatches(steps, frames [][]NodeID) bool {
	if len(steps) == 0 || len(steps) > len(frames) {
		return false
	}
	frames = frames[len(frames)-len(steps):]
	assigned := make([]int, len(frames))
	for i := range assigned {
		assigned[i] = -1
	}
	var assign func(int, []bool) bool
	assign = func(step int, seen []bool) bool {
		for i, frame := range frames {
			if seen[i] || !populationMatches(steps[step], frame) {
				continue
			}
			seen[i] = true
			if assigned[i] < 0 || assign(assigned[i], seen) {
				assigned[i] = step
				return true
			}
		}
		return false
	}
	for i := range steps {
		if !assign(i, make([]bool, len(frames))) {
			return false
		}
	}
	return true
}

// PredictOrdered exposes detached conductance/support for learned pattern
// channels. Positional and contextual evidence gate prediction; observed input
// activity is not erased. Unknown cues retain their ordinary neural priors.
func (p *PatternIndex) PredictOrdered(trace *ObservationTrace) OrderedPatternPrediction {
	out := OrderedPatternPrediction{Support: make(map[NodeID]float64), Strength: make(map[NodeID]float64)}
	if p == nil || trace == nil || len(trace.Stimulus) == 0 {
		return out
	}
	type candidate struct {
		result      NodeID
		specificity int
		strength    float64
		matched     bool
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	var candidates []candidate
	best := 0
	for _, pattern := range p.patterns {
		if len(pattern.Sequence) < 2 || pattern.Result == 0 || pattern.Sequence[len(pattern.Sequence)-1].NodeID != pattern.Result {
			continue
		}
		stem := legacyStepFrames(pattern.Sequence[:len(pattern.Sequence)-1])
		var context, data [][]NodeID
		if pattern.Dynamics == OrderedDynamics {
			if pattern.Observation == nil || len(pattern.Observation.Stimulus) < 2 {
				continue
			}
			stem = pattern.Observation.Stimulus[:len(pattern.Observation.Stimulus)-1]
			context, data = pattern.Observation.Context, pattern.Observation.Data
		}
		if !orderedSuffixBagMatches(stem, trace.Stimulus) {
			continue
		}
		matched := orderedSuffixMatches(stem, trace.Stimulus) && orderedSuffixMatches(context, trace.Context) && orderedSuffixMatches(data, trace.Data)
		if len(pattern.Context) > 0 {
			matched = false
		} // unordered historical context cannot supply positional proof
		specificity := len(stem) + len(context) + len(data)
		if matched {
			best = max(best, specificity)
		}
		frequency := min(float64(max(int64(0), pattern.Frequency)), 10)
		strength := clamp01(pattern.Weight) * clamp01(pattern.Confidence) * (.5 + .5*frequency/10)
		if strength > 0 {
			strength = max(.01, min(.35, strength))
		}
		candidates = append(candidates, candidate{pattern.Result, specificity, strength, matched})
	}
	if best == 0 {
		return out
	}
	competitors := make(map[NodeID]bool)
	for _, c := range candidates {
		competitors[c.result] = true
		if c.matched {
			support := float64(c.specificity) / float64(best)
			out.Support[c.result] = max(out.Support[c.result], support)
			out.Strength[c.result] = max(out.Strength[c.result], c.strength*support)
		}
	}
	for id := range competitors {
		out.Competitors = append(out.Competitors, id)
	}
	sort.Slice(out.Competitors, func(i, j int) bool { return out.Competitors[i] < out.Competitors[j] })
	return out
}
