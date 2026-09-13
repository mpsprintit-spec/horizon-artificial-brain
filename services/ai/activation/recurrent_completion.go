package activation

import (
	"sort"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// RecurrentCompletion advances a cue only through the learned dynamic
// synapses. PatternIndex is not consulted. This is the substrate-level path
// intended to eventually make explicit pattern indexing unnecessary for
// online completion.
func (e *Engine) RecurrentCompletion(cue []knowledge.PatternStep, cycles int, now time.Time) Result {
	if e == nil || e.Memory == nil || len(cue) == 0 {
		return Result{Activations: map[knowledge.NodeID]float64{}, Confidence: map[knowledge.NodeID]float64{}}
	}
	if cycles < 1 { cycles = 1 }
	if now.IsZero() { now = time.Now().UTC() }

	state := make(map[knowledge.NodeID]float64, len(cue))
	confidence := make(map[knowledge.NodeID]float64, len(cue))
	for _, step := range cue {
		level := clamp01(step.Activation)
		if level > state[step.NodeID] { state[step.NodeID] = level }
		confidence[step.NodeID] = max(confidence[step.NodeID], level)
	}

	state, confidence = e.advance(state, confidence, now, cycles)
	return e.converge(state, confidence, now)
}

// RecurrentNext returns the strongest learned dynamic targets reached from
// the terminal state of the supplied cue. Position is used only to determine
// which cue state is the current temporal frontier; no semantic relation type
// is assigned to the nodes or synapses.
func (e *Engine) RecurrentNext(cue []knowledge.PatternStep, now time.Time) []knowledge.NodeID {
	if e == nil || e.Memory == nil || len(cue) == 0 { return nil }
	if now.IsZero() { now = time.Now().UTC() }

	frontier := cue[0]
	for _, step := range cue {
		if step.Position > frontier.Position { frontier = step }
	}

	level := clamp01(frontier.Activation)
	n := e.Memory.Registry.GetByID(frontier.NodeID)
	if n == nil { return nil }

	type candidate struct { id knowledge.NodeID; score float64 }
	byID := map[knowledge.NodeID]float64{}
	for _, syn := range n.OutboundAll() {
		if syn == nil || syn.Inhibitory { continue }
		age := temporalPenalty(now, syn.LastActivation)
		score := level * syn.Weight * syn.Confidence * age
		// A learned target remains a candidate even when temporal attenuation
		// currently drives its score to zero. Temporal decay ranks candidates;
		// it must not erase the existence of a learned pathway.
		if _, exists := byID[syn.TargetID]; !exists || score > byID[syn.TargetID] {
			byID[syn.TargetID] = score
		}
	}
	candidates := make([]candidate, 0, len(byID))
	for id, score := range byID { candidates = append(candidates, candidate{id, score}) }
	sort.Slice(candidates, func(i,j int) bool { if candidates[i].score == candidates[j].score { return candidates[i].id < candidates[j].id }; return candidates[i].score > candidates[j].score })
	out := make([]knowledge.NodeID, len(candidates))
	for i, candidate := range candidates { out[i] = candidate.id }
	return out
}
