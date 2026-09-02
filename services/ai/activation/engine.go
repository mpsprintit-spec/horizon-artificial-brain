package activation

import (
	"sort"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

type Engine struct {
	Memory     *knowledge.KnowledgeBase
	Decay      float64
	SpreadRate float64
	Threshold  float64
	Inhibition float64
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

func NewEngine(memory *knowledge.KnowledgeBase) *Engine {
	return &Engine{Memory: memory, Decay: 0.12, SpreadRate: 0.65, Threshold: 0.25, Inhibition: 0.55}
}

func (e *Engine) Activate(tokens []string, cycles int) Result {
	return e.ActivateWith(Request{StimulusTokens: tokens, Cycles: cycles, Now: time.Now().UTC()})
}

func (e *Engine) ActivateWith(req Request) Result {
	if req.Cycles < 1 {
		req.Cycles = 1
	}
	if req.Now.IsZero() {
		req.Now = time.Now().UTC()
	}
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

	for i := 0; i < req.Cycles; i++ {
		next := map[knowledge.NodeID]float64{}
		nextConfidence := map[knowledge.NodeID]float64{}
		// PENTING: cuma node yang MEMANG sedang aktif (ada di `state`) yang
		// disuntik ulang resting activation-nya -- bukan SELURUH registry.
		// Sebelumnya, semua node yang pernah ada (bahkan yang tidak
		// berhubungan sama sekali dengan kalimat sekarang) ikut disuntik
		// ulang tiap siklus, dan kalau hubungannya kebetulan sudah sangat
		// sering diulang di masa lalu, sinyal kecil itu bisa menumpuk 8 kali
		// berturut-turut dan "menyusup" jadi jawaban untuk topik yang sama
		// sekali tidak relevan.
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
				agePenalty := temporalPenalty(req.Now, s.LastActivation)
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
	return e.converge(state, confidence, req.Now)
}

func (e *Engine) converge(state, confidence map[knowledge.NodeID]float64, now time.Time) Result {
	var ranked []*knowledge.ConceptNode
	var resonance float64
	for id, level := range state {
		n := e.Memory.Registry.GetByID(id)
		if n == nil {
			continue
		}
		n.Activation = level
		if level >= max(e.Threshold, n.Threshold) {
			n.Frequency++
			n.LastActivation = now
			ranked = append(ranked, n)
			resonance += level * max(confidence[id], 0.1)
		}
	}
	sort.Slice(ranked, func(i, j int) bool {
		left, right := ranked[i], ranked[j]
		if left.Activation != right.Activation {
			return left.Activation > right.Activation
		}
		if confidence[left.ID] != confidence[right.ID] {
			return confidence[left.ID] > confidence[right.ID]
		}
		if left.Importance != right.Importance {
			return left.Importance > right.Importance
		}
		if left.Frequency != right.Frequency {
			return left.Frequency > right.Frequency
		}
		return left.Token < right.Token
	})
	return Result{Converged: len(ranked) > 0, Resonance: resonance, Activations: state, Confidence: confidence, RankedNodes: ranked}
}

func temporalPenalty(now, last time.Time) float64 {
	if last.IsZero() {
		return 0.7
	}
	days := now.Sub(last).Hours() / 24
	if days <= 1 {
		return 1
	}
	return clamp(1-(days*0.01), 0.35, 1)
}

func normalize(in map[knowledge.NodeID]float64) map[knowledge.NodeID]float64 {
	for k, v := range in {
		in[k] = squash(v)
	}
	return in
}

// squash mempertahankan nilai APA ADANYA selama masih di rentang wajar [0,1]
// (persis seperti cara lama) -- supaya kalibrasi ambang batas yang sudah ada
// tidak berubah. Cuma begitu nilai melebihi 1 yang diperhalus (tidak dipotong
// rata ke 1) -- supaya node yang jauh lebih kuat tetap kelihatan lebih kuat
// dari yang cuma sedikit melewati batas, bukan numpuk sama rata.
func squash(v float64) float64 {
	if v <= 1 {
		return clamp(v, 0, 1)
	}
	return 1 + (v-1)/v
}

func clamp(v, low, high float64) float64 {
	if v < low {
		return low
	}
	if v > high {
		return high
	}
	return v
}
