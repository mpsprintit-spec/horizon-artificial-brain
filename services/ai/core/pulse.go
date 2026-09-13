package core

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/learning"
	"github.com/project-horizon/horizon-core/services/ai/perception"
	"github.com/project-horizon/horizon-core/services/ai/runtime"
	"github.com/project-horizon/horizon-core/services/ai/thinking"
)

func (h *HorizonEngine) Pulse(ctx context.Context, task TaskPulse) PulseResult {
	prompt := strings.TrimSpace(task.Stimulus)
	contextTokens := strings.Fields(strings.ToLower(task.Context))
	intent := classifyIntent(prompt)

	switch intent {
	case IntentConfirm:
		h.Learning.Confirm()
		return PulseResult{Answer: "Baik, saya perkuat keyakinan saya soal itu.", Confidence: 1, Success: true, Intent: string(intent), Path: "legacy_control"}
	case IntentRetraction:
		h.Learning.Retract()
		return PulseResult{Answer: "Baik, saya lemahkan/lupakan itu.", Confidence: 1, Success: true, Intent: string(intent), Path: "legacy_control"}
	}

	learned := false
	if intent == IntentTeaching && h.Runtime != nil {
		experience := learning.Experience{
			Sequence:   strings.Fields(strings.ToLower(prompt)),
			Weight:     0.85,
			Confidence: 0.35,
		}
		_, err := h.Runtime.LearnExperience(experience, time.Now().UTC())
		learned = err == nil && len(experience.Sequence) > 0
	}

	// Legacy cognition is isolated behind an explicit opt-in switch. The neural
	// runtime is the only default cognitive authority.
	if LegacyCognitionEnabled {
		return h.pulseLegacy(ctx, prompt, contextTokens, intent, learned)
	}

	if h.Runtime == nil {
		return PulseResult{Intent: string(intent), Path: "neural_runtime_unavailable", Learned: learned}
	}
	output, err := h.Runtime.CognitiveProcess(runtime.Event{
		ID:        fmt.Sprintf("pulse-%d", time.Now().UnixNano()),
		Stimulus:  strings.Fields(strings.ToLower(prompt)),
		Cycles:    8,
		Timestamp: time.Now().UTC(),
	})
	if err != nil {
		return PulseResult{Intent: string(intent), Path: "neural_runtime_error", Learned: learned}
	}

	concepts := make([]string, 0, len(output.RankedNodes))
	for _, id := range output.RankedNodes {
		if n := h.Knowledge.Registry.GetByID(id); n != nil && n.Token != "" {
			concepts = append(concepts, n.Token)
		}
	}

	answer := "Saya belum cukup tahu untuk memberikan jawaban yang dapat dipastikan."
	if len(concepts) > 0 && output.Resonance >= 0.2 {
		answer = strings.Join(concepts, " ")
	}

	return PulseResult{
		Answer:         answer,
		Concepts:       concepts,
		Confidence:     populationConfidence(output),
		Learned:        learned,
		Success:        true,
		Intent:         string(intent),
		Path:           "neural_runtime",
	}
}

func populationConfidence(output runtime.CognitiveOutput) float64 {
	best := 0.0
	for _, id := range output.RankedNodes {
		if value := output.Confidence[id]; value > best {
			best = value
		}
	}
	return best
}

// pulseLegacy is retained solely for controlled compatibility experiments.
// It is not part of the default Horizon cognitive path. Web search is never
// invoked here while WebSearchEnabled is false.
func (h *HorizonEngine) pulseLegacy(ctx context.Context, prompt string, contextTokens []string, intent Intent, learned bool) PulseResult {
	var thought thinking.Thought
	var ok bool
	thought, ok = h.Thinking.ThinkAbout(prompt, contextTokens)

	// Web search is deliberately dormant during the neural migration. The
	// capability remains installed but cannot fetch or inject external data.
	if h.WebSearchEnabled && h.WebSearch != nil && h.WebSearch.ShouldSearch(thought.Confidence, len(h.Thinking.LastState.UnknownNodes), len(thought.Conflicts)) {
		thought.NeedsWebSearch = true
		results, err := h.WebSearch.Perceive(ctx, prompt)
		if err == nil {
			for _, r := range results {
				signal := perception.FromWebSearch(r.Source, r.Tokens, r.Confidence)
				h.Learning.Assimilate(signal.RawText, signal.Confidence, 0.7)
			}
			thought, ok = h.Thinking.ThinkAbout(prompt, contextTokens)
		}
	}

	interp := h.Decision.Resolve(thought)
	answer := ""
	if interp != nil && (len(interp.Relations) > 0 || len(interp.Nodes) > 0) {
		if interp.EvalStatus != "" {
			answer = h.Language.RealizeEvaluation(interp, thought.Confidence, thought.NeedsWebSearch)
		} else {
			answer = h.Language.Realize(interp, thought.Confidence, thought.NeedsWebSearch)
		}
	} else {
		best := h.Decision.Decide(thought.Hypotheses)
		answer = h.Language.Generate(best, thought.Confidence, thought.NeedsWebSearch, thought.Inferences, false)
	}

	return PulseResult{
		Answer: answer, Concepts: thought.Concepts, Confidence: thought.Confidence,
		NeedsWebSearch: thought.NeedsWebSearch, Learned: learned, Success: ok,
		Hypotheses: thought.Hypotheses, Intent: string(intent), Path: "legacy_compatibility",
	}
}
