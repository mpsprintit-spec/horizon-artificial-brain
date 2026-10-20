package knowledge

import (
	"sort"
	"time"
)

// ApplyObservedAssociationFeedback handles an explicit supervised population
// transition. A different observed successor is counterevidence to existing
// outgoing alternatives. It does not delete history or increment confirmation
// frequency for those alternatives. Context-specific pattern credit is separate.
func (k *KnowledgeBase) ApplyObservedAssociationFeedback(sources []NodeID, actual []NodeID, confidence float64, now time.Time) {
	if k == nil || len(actual) == 0 {
		return
	}
	observed := map[NodeID]bool{}
	for _, id := range actual {
		observed[id] = true
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	for _, source := range sources {
		node := k.Registry.GetByID(source)
		if node == nil {
			continue
		}
		for _, s := range node.OutboundAll() {
			if s == nil || s.Inhibitory || observed[s.TargetID] {
				continue
			}
			delta := ReconsolidationGain * clamp01(confidence) * clamp01(s.Dynamic.Weight) * clamp01(s.Dynamic.Confidence)
			if delta <= 0 {
				continue
			}
			s.Dynamic.Weight = clamp01(s.Dynamic.Weight - delta)
			s.Dynamic.Confidence = clamp01(s.Dynamic.Confidence - delta*.5)
			s.Dynamic.LastModification = now
			syncSynapseLegacyState(s)
		}
	}
}

// ObservedActionCredit selects the existing context-free action/outcome traces
// whose first source belongs to the runtime-validated action binding.
func (p *PatternIndex) ObservedActionCredit(sources map[NodeID]float64, predicted map[NodeID]float64) map[PatternID]float64 {
	out := map[PatternID]float64{}
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, pattern := range p.patterns {
		if pattern.Dynamics == LegacyDynamics && len(pattern.Context) == 0 && len(pattern.Sequence) >= 2 && sources[pattern.Sequence[0].NodeID] > 0 && predicted[pattern.Result] > 0 {
			out[pattern.ID] = clamp01(predicted[pattern.Result])
		}
	}
	return out
}

// ApplyObservedPatternFeedback revises only channels credited by the actual
// prior cue/context. History and experience counts remain intact. This is
// numeric outcome feedback, never semantic contradiction recognition.
func (p *PatternIndex) ApplyObservedPatternFeedback(credit map[PatternID]float64, actual map[NodeID]float64, errorSignal float64) {
	if p == nil || errorSignal <= 0 || len(actual) == 0 {
		return
	}
	ids := make([]PatternID, 0, len(credit))
	for id := range credit {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, id := range ids {
		pattern := p.patterns[id]
		if pattern == nil {
			continue
		}
		delta := ReconsolidationGain * clamp01(errorSignal) * clamp01(credit[id])
		if actual[pattern.Result] <= 0 {
			delta = -delta
		}
		pattern.Weight = clamp01(pattern.Weight + delta)
		pattern.Confidence = clamp01(pattern.Confidence + delta*.5)
	}
}
