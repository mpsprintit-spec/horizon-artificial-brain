package memory

import (
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// Engine is an operational facade over the canonical Brain substrate. It does
// not own a second neural memory store.
type Engine struct{ Kb *knowledge.KnowledgeBase }

func NewEngine(kb *knowledge.KnowledgeBase) *Engine {
	return &Engine{Kb: kb}
}

// Optimize weakens stale synapses in the canonical dynamic state. Legacy
// mirror fields are updated together so callers cannot observe divergent
// neural state.
func (m *Engine) Optimize(now time.Time, staleAfter time.Duration, decay float64) {
	if m == nil || m.Kb == nil || staleAfter <= 0 || decay <= 0 {
		return
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	for _, node := range m.Kb.Registry.Nodes() {
		for _, synapse := range node.OutboundAll() {
			if synapse == nil || synapse.Dynamic.LastActivation.IsZero() || now.Sub(synapse.Dynamic.LastActivation) <= staleAfter {
				continue
			}
			synapse.Dynamic.Weight = clamp01(synapse.Dynamic.Weight * (1 - decay))
			synapse.Dynamic.Confidence = clamp01(synapse.Dynamic.Confidence * (1 - decay))
			synapse.Dynamic.LastModification = now
		synapse.Weight = synapse.Dynamic.Weight
		synapse.Confidence = synapse.Dynamic.Confidence
		synapse.LastActivation = synapse.Dynamic.LastActivation
		}
	}
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
