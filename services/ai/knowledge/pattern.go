package knowledge

import (
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type PatternID int64

// PatternStep is one ordered activation event inside a distributed pattern.
// Delta is measured from the previous step. No semantic meaning is encoded
// here: NodeID identifies only a participating neural unit.
type PatternStep struct {
	NodeID     NodeID        `json:"node_id"`
	Position   int           `json:"position"`
	Delta      time.Duration `json:"delta"`
	Activation float64       `json:"activation"`
}

// ContextFrame records co-active state surrounding a pattern. Context is
// represented by neural units, not semantic key/value labels.
type ContextFrame struct {
	NodeID     NodeID  `json:"node_id"`
	Activation float64 `json:"activation"`
	Weight     float64 `json:"weight"`
}

// PatternTrace is the temporal/contextual evidence used to learn or match a
// pattern. The existing unordered Members representation remains supported
// for persistence and backward compatibility.
type PatternTrace struct {
	Sequence []PatternStep
	Context  []ContextFrame
}

// PatternSynapse represents a learned distributed co-activation pattern.
// Members/Result are legacy-compatible fields. Sequence and Context carry
// richer temporal and contextual structure without introducing an ontology.
// Result is retained as an activation target for existing callers; it does
// not define what that target means.
type PatternSynapse struct {
	ID         PatternID      `json:"id"`
	Members    []NodeID       `json:"members"`
	Sequence   []PatternStep  `json:"sequence,omitempty"`
	Context    []ContextFrame `json:"context,omitempty"`
	Result     NodeID         `json:"result"`
	Weight     float64        `json:"weight"`
	Confidence float64        `json:"confidence"`
	Frequency  int64          `json:"frequency"`
}

type PatternIndex struct {
	mu       sync.RWMutex
	nextID   PatternID
	patterns map[PatternID]*PatternSynapse
}

func NewPatternIndex() *PatternIndex {
	return &PatternIndex{nextID: 1, patterns: map[PatternID]*PatternSynapse{}}
}

func memberKey(members []NodeID) string {
	sorted := append([]NodeID{}, members...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	parts := make([]string, len(sorted))
	for i, id := range sorted {
		parts[i] = strconv.FormatInt(int64(id), 10)
	}
	return strings.Join(parts, ",")
}

func sequenceKey(sequence []PatternStep) string {
	parts := make([]string, 0, len(sequence))
	for _, step := range sequence {
		parts = append(parts,
			strconv.FormatInt(int64(step.NodeID), 10)+":"+
				strconv.Itoa(step.Position)+":"+
				strconv.FormatInt(int64(step.Delta), 10),
		)
	}
	return strings.Join(parts, ",")
}

func contextKey(context []ContextFrame) string {
	sorted := append([]ContextFrame{}, context...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].NodeID == sorted[j].NodeID {
			return sorted[i].Activation < sorted[j].Activation
		}
		return sorted[i].NodeID < sorted[j].NodeID
	})
	parts := make([]string, 0, len(sorted))
	for _, frame := range sorted {
		parts = append(parts,
			strconv.FormatInt(int64(frame.NodeID), 10)+":"+
				strconv.FormatFloat(frame.Activation, 'g', -1, 64),
		)
	}
	return strings.Join(parts, ",")
}

func normalizeSequence(sequence []PatternStep) []PatternStep {
	out := append([]PatternStep(nil), sequence...)
	for i := range out {
		out[i].Position = i
		out[i].Activation = clamp01(out[i].Activation)
	}
	return out
}

func normalizeContext(context []ContextFrame) []ContextFrame {
	out := append([]ContextFrame(nil), context...)
	for i := range out {
		out[i].Activation = clamp01(out[i].Activation)
		out[i].Weight = clamp01(out[i].Weight)
	}
	return out
}

func sequenceFromMembers(members []NodeID) []PatternStep {
	sequence := make([]PatternStep, len(members))
	for i, id := range members {
		sequence[i] = PatternStep{NodeID: id, Position: i, Activation: 1}
	}
	return sequence
}

func mergeSequence(existing, incoming []PatternStep, frequency int64) []PatternStep {
	if len(existing) != len(incoming) || frequency <= 0 {
		return append([]PatternStep(nil), incoming...)
	}
	merged := append([]PatternStep(nil), existing...)
	n := float64(frequency)
	for i := range merged {
		merged[i].Delta = time.Duration((float64(merged[i].Delta)*n + float64(incoming[i].Delta)) / (n + 1))
		merged[i].Activation = clamp01((merged[i].Activation*n + incoming[i].Activation) / (n + 1))
	}
	return merged
}

func mergeContext(existing, incoming []ContextFrame, frequency int64) []ContextFrame {
	if frequency <= 0 {
		return append([]ContextFrame(nil), incoming...)
	}
	// Context is a sparse population. Merge matching units and retain new
	// units rather than duplicating an entire context trace.
	byID := make(map[NodeID]ContextFrame, len(existing)+len(incoming))
	for _, frame := range existing {
		byID[frame.NodeID] = frame
	}
	n := float64(frequency)
	for _, frame := range incoming {
		if old, ok := byID[frame.NodeID]; ok {
			old.Activation = clamp01((old.Activation*n + frame.Activation) / (n + 1))
			old.Weight = clamp01((old.Weight*n + frame.Weight) / (n + 1))
			byID[frame.NodeID] = old
		} else {
			byID[frame.NodeID] = frame
		}
	}
	out := make([]ContextFrame, 0, len(byID))
	for _, frame := range byID {
		out = append(out, frame)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].NodeID < out[j].NodeID })
	return out
}

// Learn preserves the original unordered-pattern API. The generated
// Sequence records the observed member order but does not change the legacy
// identity rule, so old callers keep their deduplication behavior.
func (p *PatternIndex) Learn(members []NodeID, result NodeID, weight, confidence float64) *PatternSynapse {
	return p.learn(members, sequenceFromMembers(members), nil, result, weight, confidence)
}

// LearnTrace learns an ordered, contextual activation pattern. Identity uses
// sequence and context, allowing different temporal arrangements of the same
// population to remain distinct while repeated evidence reinforces one trace.
func (p *PatternIndex) LearnTrace(sequence []PatternStep, context []ContextFrame, result NodeID, weight, confidence float64) *PatternSynapse {
	sequence = normalizeSequence(sequence)
	context = normalizeContext(context)
	members := make([]NodeID, len(sequence))
	for i, step := range sequence {
		members[i] = step.NodeID
	}
	return p.learn(members, sequence, context, result, weight, confidence)
}

func (p *PatternIndex) learn(members []NodeID, sequence []PatternStep, context []ContextFrame, result NodeID, weight, confidence float64) *PatternSynapse {
	p.mu.Lock()
	defer p.mu.Unlock()

	legacy := len(context) == 0 && sequenceKey(sequence) == sequenceKey(sequenceFromMembers(members))
	for _, existing := range p.patterns {
		if existing.Result != result {
			continue
		}
		matches := false
		if legacy {
			matches = memberKey(existing.Members) == memberKey(members)
		} else {
			matches = sequenceKey(existing.Sequence) == sequenceKey(sequence) && contextKey(existing.Context) == contextKey(context)
		}
		if !matches {
			continue
		}
		frequency := existing.Frequency
		if frequency < 0 {
			frequency = 0
		}
		existing.Weight = clamp01((existing.Weight*float64(frequency) + weight) / float64(frequency+1))
		existing.Confidence = clamp01(1 - (1-existing.Confidence)*(1-confidence))
		existing.Sequence = mergeSequence(existing.Sequence, sequence, frequency)
		existing.Context = mergeContext(existing.Context, context, frequency)
		existing.Frequency = frequency + 1
		return existing
	}

	ps := &PatternSynapse{
		ID:         p.nextID,
		Members:    append([]NodeID{}, members...),
		Sequence:   append([]PatternStep(nil), sequence...),
		Context:    append([]ContextFrame(nil), context...),
		Result:     result,
		Weight:     clamp01(weight),
		Confidence: clamp01(confidence),
		Frequency:  1,
	}
	p.patterns[p.nextID] = ps
	p.nextID++
	return ps
}

// Match keeps the legacy population-based matcher. It intentionally ignores
// order and context so existing learning/recall behavior remains compatible.
func (p *PatternIndex) Match(candidateIDs []NodeID) []*PatternSynapse {
	p.mu.RLock()
	defer p.mu.RUnlock()
	set := map[NodeID]bool{}
	for _, id := range candidateIDs {
		set[id] = true
	}
	var matches []*PatternSynapse
	for _, pattern := range p.patterns {
		if len(pattern.Members) == 0 {
			continue
		}
		allPresent := true
		for _, m := range pattern.Members {
			if !set[m] {
				allPresent = false
				break
			}
		}
		if allPresent {
			matches = append(matches, pattern)
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		if len(matches[i].Members) != len(matches[j].Members) {
			return len(matches[i].Members) > len(matches[j].Members)
		}
		return matches[i].Confidence*matches[i].Weight > matches[j].Confidence*matches[j].Weight
	})
	return matches
}

// MatchTrace first requires the same population, then scores temporal and
// contextual agreement. It returns the strongest traces first; no semantic
// labels or rule tables are consulted.
func (p *PatternIndex) MatchTrace(sequence []PatternStep, context []ContextFrame) []*PatternSynapse {
	sequence = normalizeSequence(sequence)
	context = normalizeContext(context)
	p.mu.RLock()
	defer p.mu.RUnlock()

	candidateSet := make(map[NodeID]bool, len(sequence))
	for _, step := range sequence {
		candidateSet[step.NodeID] = true
	}

	type scored struct {
		pattern *PatternSynapse
		score   float64
	}
	var scoredMatches []scored
	for _, pattern := range p.patterns {
		if len(pattern.Members) == 0 || len(pattern.Sequence) == 0 {
			continue
		}
		allPresent := true
		for _, id := range pattern.Members {
			if !candidateSet[id] {
				allPresent = false
				break
			}
		}
		if !allPresent {
			continue
		}
		temporal := sequenceSimilarity(pattern.Sequence, sequence)
		contextScore := contextSimilarity(pattern.Context, context)
		score := temporal * 0.7 + contextScore*0.3
		score *= clamp01(pattern.Weight) * clamp01(pattern.Confidence)
		scoredMatches = append(scoredMatches, scored{pattern: pattern, score: score})
	}
	sort.Slice(scoredMatches, func(i, j int) bool { return scoredMatches[i].score > scoredMatches[j].score })
	out := make([]*PatternSynapse, len(scoredMatches))
	for i, item := range scoredMatches {
		out[i] = item.pattern
	}
	return out
}

func sequenceSimilarity(a, b []PatternStep) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	limit := len(a)
	if len(b) < limit {
		limit = len(b)
	}
	positionScore := 0.0
	temporalScore := 0.0
	activationScore := 0.0
	for i := 0; i < limit; i++ {
		if a[i].NodeID == b[i].NodeID {
			positionScore += 1
		} else {
			// A different unit at the same position is not equivalent, but
			// still contributes through population overlap below.
			continue
		}
		activationScore += 1 - abs(a[i].Activation-b[i].Activation)
		if i == 0 {
			temporalScore += 1
		} else {
			deltaA := float64(a[i].Delta)
			deltaB := float64(b[i].Delta)
			denom := deltaA + deltaB
			if denom == 0 {
				temporalScore += 1
			} else {
				d := abs(deltaA-deltaB) / denom
				if d > 1 {
					d = 1
				}
				temporalScore += 1 - d
			}
		}
	}
	lengthScore := float64(limit) / float64(maxInt(len(a), len(b)))
	return clamp01((positionScore/float64(limit))*0.55 + (temporalScore/float64(limit))*0.25 + (activationScore/float64(limit))*0.20) * lengthScore
}

func contextSimilarity(a, b []ContextFrame) float64 {
	if len(a) == 0 || len(b) == 0 {
		if len(a) == 0 && len(b) == 0 {
			return 1
		}
		return 0
	}
	bByID := make(map[NodeID]ContextFrame, len(b))
	for _, frame := range b {
		bByID[frame.NodeID] = frame
	}
	matched := 0.0
	for _, frame := range a {
		other, ok := bByID[frame.NodeID]
		if !ok {
			continue
		}
		matched += (1 - abs(frame.Activation-other.Activation)) * (1 - abs(frame.Weight-other.Weight))
	}
	return clamp01(matched / float64(maxInt(len(a), len(b))))
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (p *PatternIndex) All() []*PatternSynapse {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]*PatternSynapse, 0, len(p.patterns))
	for _, ps := range p.patterns {
		out = append(out, ps)
	}
	return out
}

// ResultsFor mengembalikan semua pola yang HASILNYA node tertentu -- dipakai
// waktu node itu SENDIRI yang aktif (bukan seluruh anggota polanya), supaya
// "tidak" tetap bisa membawa pola {saya,bercanda}->tidak sebagai bukti,
// walau cuma "tidak" sendiri yang diketik.
func (p *PatternIndex) ResultsFor(id NodeID) []*PatternSynapse {
	p.mu.RLock()
	defer p.mu.RUnlock()
	var out []*PatternSynapse
	for _, ps := range p.patterns {
		if ps.Result == id {
			out = append(out, ps)
		}
	}
	return out
}
