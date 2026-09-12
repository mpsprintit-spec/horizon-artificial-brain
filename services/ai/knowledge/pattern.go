package knowledge

import (
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type PatternID int64

type PatternStep struct {
	NodeID     NodeID        `json:"node_id"`
	Position   int           `json:"position"`
	Delta      time.Duration `json:"delta"`
	Activation float64       `json:"activation"`
}

type ContextFrame struct {
	NodeID     NodeID  `json:"node_id"`
	Activation float64 `json:"activation"`
	Weight     float64 `json:"weight"`
}

type PatternTrace struct {
	Sequence  []PatternStep
	Context   []ContextFrame
	Signature []float64
}

// PatternSynapse represents learned distributed co-activation evidence.
// Signature is a modality-neutral numeric trace. It is not a semantic label
// and is never interpreted as text or as a fixed ontology.
type PatternSynapse struct {
	ID         PatternID      `json:"id"`
	Members    []NodeID       `json:"members"`
	Sequence   []PatternStep  `json:"sequence,omitempty"`
	Context    []ContextFrame `json:"context,omitempty"`
	Signature  []float64      `json:"signature,omitempty"`
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
	for i, id := range sorted { parts[i] = strconv.FormatInt(int64(id), 10) }
	return strings.Join(parts, ",")
}

func sequenceKey(sequence []PatternStep) string {
	parts := make([]string, 0, len(sequence))
	for _, step := range sequence {
		parts = append(parts, strconv.FormatInt(int64(step.NodeID), 10)+":"+strconv.Itoa(step.Position)+":"+strconv.FormatInt(int64(step.Delta), 10))
	}
	return strings.Join(parts, ",")
}

func contextKey(context []ContextFrame) string {
	sorted := append([]ContextFrame{}, context...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].NodeID == sorted[j].NodeID { return sorted[i].Activation < sorted[j].Activation }
		return sorted[i].NodeID < sorted[j].NodeID
	})
	parts := make([]string, 0, len(sorted))
	for _, frame := range sorted {
		parts = append(parts, strconv.FormatInt(int64(frame.NodeID), 10)+":"+strconv.FormatFloat(frame.Activation, 'g', -1, 64))
	}
	return strings.Join(parts, ",")
}

func normalizeSequence(sequence []PatternStep) []PatternStep {
	out := append([]PatternStep(nil), sequence...)
	for i := range out { out[i].Position = i; out[i].Activation = clamp01(out[i].Activation) }
	return out
}

func normalizeContext(context []ContextFrame) []ContextFrame {
	out := append([]ContextFrame(nil), context...)
	for i := range out { out[i].Activation = clamp01(out[i].Activation); out[i].Weight = clamp01(out[i].Weight) }
	return out
}

func normalizeSignature(signature []float64) []float64 {
	out := append([]float64(nil), signature...)
	for i, value := range out { out[i] = clamp(value, -1, 1) }
	return out
}

func sequenceFromMembers(members []NodeID) []PatternStep {
	sequence := make([]PatternStep, len(members))
	for i, id := range members { sequence[i] = PatternStep{NodeID: id, Position: i, Activation: 1} }
	return sequence
}

func mergeSequence(existing, incoming []PatternStep, frequency int64) []PatternStep {
	if len(existing) != len(incoming) || frequency <= 0 { return append([]PatternStep(nil), incoming...) }
	merged := append([]PatternStep(nil), existing...)
	n := float64(frequency)
	for i := range merged {
		merged[i].Delta = time.Duration((float64(merged[i].Delta)*n + float64(incoming[i].Delta)) / (n + 1))
		merged[i].Activation = clamp01((merged[i].Activation*n + incoming[i].Activation) / (n + 1))
	}
	return merged
}

func mergeContext(existing, incoming []ContextFrame, frequency int64) []ContextFrame {
	if frequency <= 0 { return append([]ContextFrame(nil), incoming...) }
	byID := make(map[NodeID]ContextFrame, len(existing)+len(incoming))
	for _, frame := range existing { byID[frame.NodeID] = frame }
	n := float64(frequency)
	for _, frame := range incoming {
		if old, ok := byID[frame.NodeID]; ok {
			old.Activation = clamp01((old.Activation*n + frame.Activation) / (n + 1))
			old.Weight = clamp01((old.Weight*n + frame.Weight) / (n + 1))
			byID[frame.NodeID] = old
		} else { byID[frame.NodeID] = frame }
	}
	out := make([]ContextFrame, 0, len(byID))
	for _, frame := range byID { out = append(out, frame) }
	sort.Slice(out, func(i, j int) bool { return out[i].NodeID < out[j].NodeID })
	return out
}

func mergeSignature(existing, incoming []float64, frequency int64) []float64 {
	if len(incoming) == 0 { return append([]float64(nil), existing...) }
	if len(existing) != len(incoming) || frequency <= 0 { return normalizeSignature(incoming) }
	merged := make([]float64, len(existing))
	n := float64(frequency)
	for i := range merged { merged[i] = clamp((existing[i]*n+incoming[i])/(n+1), -1, 1) }
	return merged
}

func signatureSimilarity(a, b []float64) float64 {
	if len(a) == 0 || len(b) == 0 || len(a) != len(b) { return 0 }
	var distance float64
	for i := range a { distance += abs(a[i] - b[i]) }
	return clamp01(1 - distance/(2*float64(len(a))))
}

func (p *PatternIndex) Learn(members []NodeID, result NodeID, weight, confidence float64) *PatternSynapse {
	return p.learn(members, sequenceFromMembers(members), nil, nil, result, weight, confidence)
}

// LearnTrace learns ordered/contextual evidence and an optional numeric
// signature. Signature matching is deliberately generic so text, vision,
// audio, and sensor encoders can feed the same substrate.
func (p *PatternIndex) LearnTrace(sequence []PatternStep, context []ContextFrame, signature []float64, result NodeID, weight, confidence float64) *PatternSynapse {
	sequence = normalizeSequence(sequence)
	context = normalizeContext(context)
	signature = normalizeSignature(signature)
	members := make([]NodeID, len(sequence))
	for i, step := range sequence { members[i] = step.NodeID }
	return p.learn(members, sequence, context, signature, result, weight, confidence)
}

func (p *PatternIndex) learn(members []NodeID, sequence []PatternStep, context []ContextFrame, signature []float64, result NodeID, weight, confidence float64) *PatternSynapse {
	p.mu.Lock(); defer p.mu.Unlock()
	legacy := len(context) == 0 && len(signature) == 0 && sequenceKey(sequence) == sequenceKey(sequenceFromMembers(members))
	for _, existing := range p.patterns {
		if existing.Result != result { continue }
		matches := false
		if legacy { matches = memberKey(existing.Members) == memberKey(members) } else {
			matches = sequenceKey(existing.Sequence) == sequenceKey(sequence) && contextKey(existing.Context) == contextKey(context)
			if matches && len(signature) > 0 && len(existing.Signature) > 0 { matches = signatureSimilarity(existing.Signature, signature) >= 0.98 }
		}
		if !matches { continue }
		frequency := existing.Frequency
		if frequency < 0 { frequency = 0 }
		existing.Weight = clamp01((existing.Weight*float64(frequency)+weight)/float64(frequency+1))
		existing.Confidence = clamp01(1-(1-existing.Confidence)*(1-confidence))
		existing.Sequence = mergeSequence(existing.Sequence, sequence, frequency)
		existing.Context = mergeContext(existing.Context, context, frequency)
		existing.Signature = mergeSignature(existing.Signature, signature, frequency)
		existing.Frequency = frequency + 1
		return existing
	}
	ps := &PatternSynapse{ID: p.nextID, Members: append([]NodeID{}, members...), Sequence: append([]PatternStep(nil), sequence...), Context: append([]ContextFrame(nil), context...), Signature: normalizeSignature(signature), Result: result, Weight: clamp01(weight), Confidence: clamp01(confidence), Frequency: 1}
	p.patterns[p.nextID] = ps
	p.nextID++
	return ps
}
