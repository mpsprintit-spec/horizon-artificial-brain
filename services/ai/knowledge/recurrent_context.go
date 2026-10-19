package knowledge

import (
	"math"
	"sort"
	"time"
)

// ContextSample is transient neural eligibility, never a separate knowledge
// store. Population membership and physical observation time are retained.
type ContextSample struct {
	Population []NodeID  `json:"population"`
	At         time.Time `json:"observed_at"`
	Ordinal    uint64    `json:"ordinal"`
}

const ContextMaxFrames = 64
const ContextMaxMembers = 4096
const ContextMaxAge = 2 * time.Minute
const ContextHalfLife = 30 * time.Second

// contextMatch selects only ordered evidence required by a learned pattern.
// It can skip intervening observations; it never injects the whole history as
// current input. Greedy backward matching chooses the most recent completion.
func contextMatch(expected [][]NodeID, history []ContextSample, now time.Time) float64 {
	if len(expected) == 0 {
		return 1
	}
	index := len(history) - 1
	first := -1
	for i := len(expected) - 1; i >= 0; i-- {
		for index >= 0 && !populationMatches(expected[i], history[index].Population) {
			index--
		}
		if index < 0 {
			return 0
		}
		first = index
		index--
	}
	age := now.Sub(history[first].At)
	if age < 0 || age > ContextMaxAge {
		return 0
	}
	skipped := len(history) - first - len(expected)
	return math.Exp2(-age.Seconds()/ContextHalfLife.Seconds()) * math.Pow(.9, float64(skipped))
}

// PredictRecurrent extends the ordered pattern substrate with bounded prior
// population eligibility. Current input order remains mandatory; history can
// supply a missing prefix/context, never repair a reversed current stimulus.
func (p *PatternIndex) PredictRecurrent(trace *ObservationTrace, history []ContextSample, now time.Time) OrderedPatternPrediction {
	out := OrderedPatternPrediction{Support: make(map[NodeID]float64), Strength: make(map[NodeID]float64)}
	if p == nil || trace == nil || len(trace.Stimulus) == 0 {
		return out
	}
	type candidate struct {
		result                   NodeID
		quality, score, strength float64
	}
	var candidates []candidate
	best := 0.0
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, pattern := range p.patterns {
		if pattern.Dynamics != OrderedDynamics || pattern.Observation == nil || len(pattern.Observation.Stimulus) < 2 {
			continue
		}
		observed := pattern.Observation
		stem := observed.Stimulus[:len(observed.Stimulus)-1]
		take := min(len(stem), len(trace.Stimulus))
		suffix := stem[len(stem)-take:]
		if !orderedSuffixBagMatches(suffix, trace.Stimulus) {
			continue
		}
		matched := orderedSuffixMatches(suffix, trace.Stimulus) && orderedSuffixMatches(observed.Data, trace.Data)
		missing := ClonePopulationFrames(stem[:len(stem)-take])
		if len(trace.Context) > 0 {
			matched = matched && orderedSuffixMatches(observed.Context, trace.Context)
		} else {
			missing = append(ClonePopulationFrames(observed.Context), missing...)
		}
		quality := 0.0
		if matched {
			quality = contextMatch(missing, history, now)
		}
		specificity := len(stem) + len(observed.Context) + len(observed.Data)
		score := float64(specificity) * quality
		best = max(best, score)
		frequency := min(float64(max(int64(0), pattern.Frequency)), 10)
		strength := clamp01(pattern.Weight) * clamp01(pattern.Confidence) * (.5 + .5*frequency/10)
		if strength > 0 {
			strength = max(.01, min(.35, strength))
		}
		candidates = append(candidates, candidate{pattern.Result, quality, score, strength})
	}
	if best == 0 {
		return out
	}
	competitors := make(map[NodeID]bool)
	for _, c := range candidates {
		competitors[c.result] = true
		// Relative competition separates recent from older competing contexts;
		// absolute eligibility still decays with physical time and interference.
		support := c.quality * math.Pow(c.score/best, 4)
		out.Support[c.result] = max(out.Support[c.result], support)
		out.Strength[c.result] = max(out.Strength[c.result], c.strength*support)
	}
	for id := range competitors {
		out.Competitors = append(out.Competitors, id)
	}
	sort.Slice(out.Competitors, func(i, j int) bool { return out.Competitors[i] < out.Competitors[j] })
	return out
}
