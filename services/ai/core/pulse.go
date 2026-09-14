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
	// Gate 4: the default path must enter the neural runtime before any legacy
	// intent classification or semantic control logic. Legacy behavior is an
	// explicit compatibility path only.
	if !LegacyCognitionEnabled {
		return h.pulseNeural(task)
	}

	prompt := strings.TrimSpace(task.Stimulus)
	contextTokens := strings.Fields(strings.ToLower(task.Context))
	intent := classifyIntent(prompt)

	switch intent {
	case IntentConfirm:
		h.Learning.Confirm()
		return PulseResult{Answer: "Baik, saya perkuat keyakinan saya soal itu.", Confidence: 1, Success: true, Intent: string(intent), Path: "legacy_control", InterpretationSource: "legacy-compatibility"}
	case IntentRetraction:
		h.Learning.Retract()
		return PulseResult{Answer: "Baik, saya lemahkan/lupakan itu.", Confidence: 1, Success: true, Intent: string(intent), Path: "legacy_control", InterpretationSource: "legacy-compatibility"}
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

	return h.pulseLegacy(ctx, prompt, contextTokens, intent, learned)
}

// pulseNeural is the sole default cognitive path. It does not classify intent,
// invoke the legacy thinking/decision pipeline, or use semantic relation rules.
func (h *HorizonEngine) pulseNeural(task TaskPulse) PulseResult {
	if h == nil || h.Runtime == nil || h.Knowledge == nil {
		return PulseResult{Path: "neural_runtime_unavailable", Success: false}
	}

	prompt := strings.TrimSpace(task.Stimulus)
	if prompt == "" {
		return PulseResult{Path: "neural_empty_input", Success: false}
	}

	timestamp := time.Now().UTC()
	output, err := h.Runtime.CognitiveProcess(runtime.Event{
		ID:        fmt.Sprintf("pulse-%d", timestamp.UnixNano()),
		Stimulus:  strings.Fields(strings.ToLower(prompt)),
		Cycles:    8,
		Timestamp: timestamp,
	})
	if err != nil {
		return PulseResult{Path: "neural_runtime_error", Success: false}
	}

	// Gate 4: interpretation is an explicit boundary. The interpreter only
	// translates neural output; it does not mutate or supplement cognition.
	interpretation, err := h.Runtime.Interpret(output, runtime.NeuralInterpreter{})
	if err != nil {
		return PulseResult{Path: "neural_interpretation_error", Success: false}
	}

	// Token strings are presentation anchors only. They are not used to infer
	// semantic relations; the ranked result originates in neural activation.
	concepts := make([]string, 0, len(interpretation.RankedNodeIDs))
	for _, id := range interpretation.RankedNodeIDs {
		if n := h.Knowledge.Registry.GetByID(id); n != nil && n.Token != "" {
			concepts = append(concepts, n.Token)
		}
	}

	answer := "Saya belum cukup tahu untuk memberikan jawaban yang dapat dipastikan."
	if len(concepts) > 0 && interpretation.Resonance >= 0.2 {
		answer = strings.Join(concepts, " ")
	}

	return PulseResult{
		Answer:                answer,
		Concepts:              concepts,
		Confidence:            interpretationConfidence(interpretation),
		Success:               true,
		Path:                  "neural_runtime",
		InterpretationSource:  interpretation.Source,
	}
}

func interpretationConfidence(interpretation runtime.Interpretation) float64 {
	best := 0.0
	for _, id := range interpretation.RankedNodeIDs {
		if value := interpretation.Confidence[id]; value > best {
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
		InterpretationSource: "legacy-compatibility",
	}
}
