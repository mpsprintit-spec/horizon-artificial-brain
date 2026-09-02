package memory

import (
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// Engine mengelola penguatan/pelemahan hubungan antar konsep dari waktu ke
// waktu. Knowledge cuma menjaga identitas konsep -- Memory yang mengurus
// dinamika sinapsnya (menguat/melemah) seiring waktu.
type Engine struct{ Kb *knowledge.KnowledgeBase }

func NewEngine(kb *knowledge.KnowledgeBase) *Engine {
	return &Engine{Kb: kb}
}

// Optimize melemahkan sinaps yang lama tidak diperkuat lagi, tanpa menghapus node.
func (m *Engine) Optimize(now time.Time, staleAfter time.Duration, decay float64) {
	if staleAfter <= 0 || decay <= 0 {
		return
	}
	for _, node := range m.Kb.Registry.Nodes() {
		for _, synapse := range node.OutboundAll() {
			if synapse.LastActivation.IsZero() || now.Sub(synapse.LastActivation) > staleAfter {
				synapse.Weight = clamp01(synapse.Weight * (1 - decay))
			}
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
