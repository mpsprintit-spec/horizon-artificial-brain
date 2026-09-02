package context

import "github.com/project-horizon/horizon-core/services/ai/knowledge"

type Engine struct{ Memory *knowledge.KnowledgeBase }

type Result struct{ Boosts map[knowledge.NodeID]float64 }

func NewEngine(memory *knowledge.KnowledgeBase) *Engine { return &Engine{Memory: memory} }

func (e *Engine) Focus(tokens []string) Result {
	boosts := map[knowledge.NodeID]float64{}
	for _, token := range tokens {
		node := e.Memory.Fetch(token)
		if node == nil {
			continue
		}
		// Boost konteks sengaja KECIL -- ini cuma "dorongan ringan" kalau
		// ada kaitan relevan, BUKAN kekuatan yang boleh mengalahkan kata
		// yang baru saja diketik user. Konteks giliran sebelumnya tidak
		// boleh lebih kuat dari apa yang sedang dibicarakan sekarang.
		boosts[node.ID] = max(boosts[node.ID], 0.3)
		for _, synapse := range node.OutboundAll() {
			boosts[synapse.TargetID] = max(boosts[synapse.TargetID], synapse.Weight*synapse.Confidence*0.2)
		}
	}
	return Result{Boosts: boosts}
}
