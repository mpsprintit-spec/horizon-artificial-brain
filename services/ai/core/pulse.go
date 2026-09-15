package core

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
	"github.com/project-horizon/horizon-core/services/ai/perception"
	"github.com/project-horizon/horizon-core/services/ai/runtime"
	"github.com/project-horizon/horizon-core/services/ai/thinking"
)

func (h *HorizonEngine) Pulse(ctx context.Context, task TaskPulse) PulseResult {
	if !LegacyCognitionEnabled { return h.pulseNeural(task) }
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
		experience := learning.Experience{Sequence: strings.Fields(strings.ToLower(prompt)), Weight: 0.85, Confidence: 0.35}
		_, err := h.Runtime.LearnExperience(experience, time.Now().UTC())
		learned = err == nil && len(experience.Sequence) > 0
	}
	return h.pulseLegacy(ctx, prompt, contextTokens, intent, learned)
}

// pulseNeural is the sole default cognitive path. It does not classify intent,
// invoke the legacy thinking/decision pipeline, or use semantic relation rules.
func (h *HorizonEngine) pulseNeural(task TaskPulse) PulseResult {
	if h == nil || h.Runtime == nil || h.Knowledge == nil { return PulseResult{Path: "neural_runtime_unavailable", Success: false} }
	prompt := strings.TrimSpace(task.Stimulus)
	if prompt == "" { return PulseResult{Path: "neural_empty_input", Success: false} }

	timestamp := time.Now().UTC()
	eventID := fmt.Sprintf("pulse-%d", timestamp.UnixNano())

	// Normalize the external input through the perception boundary before it
	// reaches BrainRuntime. This keeps sensing/perception separate from neural
	// representation while allowing future sensor adapters to use the same
	// PerceptionEvent contract.
	perceptionEvent, err := perception.NewUserInputEvent(eventID, prompt, "pulse", timestamp)
	if err != nil || len(perceptionEvent.Signals) == 0 {
		return PulseResult{Path: "neural_perception_error", Success: false}
	}
	stimulusTokens := append([]string(nil), perceptionEvent.Signals[0].Tokens...)
	contextTokens := strings.Fields(strings.ToLower(task.Context))
	dataTokens := strings.Fields(strings.ToLower(task.Data))
	perceptionEvent.Context = append([]string(nil), contextTokens...)
	perceptionEvent.Data = append([]string(nil), dataTokens...)

	// Context and Data remain modality/provenance-bearing input. Existing neural
	// units are boosted when they are already represented in the same Brain;
	// unknown tokens are not created merely to manufacture context semantics.
	contextBoosts := make(map[knowledge.NodeID]float64)
	addContextBoosts := func(tokens []string, boost float64) {
		for _, token := range tokens {
			if node := h.Knowledge.Registry.Get(token); node != nil {
				if current := contextBoosts[node.ID]; boost > current { contextBoosts[node.ID] = boost }
			}
		}
	}
	addContextBoosts(contextTokens, 0.35)
	addContextBoosts(dataTokens, 0.20)

	output, err := h.Runtime.CognitiveProcess(runtime.Event{
		ID: eventID, Stimulus: stimulusTokens, Context: contextBoosts,
		ContextTokens: perceptionEvent.Context, DataTokens: perceptionEvent.Data,
		Source: perceptionEvent.Source, Modality: perceptionEvent.Modality,
		Cycles: 8, Timestamp: perceptionEvent.ObservedAt,
	})
	if err != nil { return PulseResult{Path: "neural_runtime_error", Success: false} }

	// The default neural path records the interaction on the same persistent
	// Brain. Provenance is attached to the experience rather than a second store.
	experienceSequence := append([]string(nil), stimulusTokens...)
	experienceSequence = append(experienceSequence, contextTokens...)
	experienceSequence = append(experienceSequence, dataTokens...)
	learned := false
	if len(experienceSequence) > 0 {
		_, learnErr := h.Runtime.LearnExperience(learning.Experience{
			ExperienceID: eventID, Sequence: experienceSequence, Weight: 0.50, Confidence: 0.20,
			Source: perceptionEvent.Source, Modality: perceptionEvent.Modality, Timestamp: perceptionEvent.ObservedAt, Reliability: 0.50,
			IndependenceGroup: eventID,
		}, perceptionEvent.ObservedAt)
		learned = learnErr == nil
	}

	interpretation, err := h.Runtime.Interpret(output, runtime.NeuralInterpreter{})
	if err != nil { return PulseResult{Path: "neural_interpretation_error", Success: false, Learned: learned} }
	concepts := make([]string, 0, len(interpretation.RankedNodeIDs))
	for _, id := range interpretation.RankedNodeIDs {
		if n := h.Knowledge.Registry.GetByID(id); n != nil && n.Token != "" { concepts = append(concepts, n.Token) }
	}
	answer := "Saya belum cukup tahu untuk memberikan jawaban yang dapat dipastikan."
	if len(concepts) > 0 && interpretation.Resonance >= 0.2 { answer = strings.Join(concepts, " ") }
	return PulseResult{Answer: answer, Concepts: concepts, Confidence: interpretationConfidence(interpretation), Success: true, Learned: learned, Path: "neural_runtime", InterpretationSource: interpretation.Source}
}

func interpretationConfidence(interpretation runtime.Interpretation) float64 {
	best := 0.0
	for _, id := range interpretation.RankedNodeIDs { if value := interpretation.Confidence[id]; value > best { best = value } }
	return best
}

func (h *HorizonEngine) pulseLegacy(ctx context.Context, prompt string, contextTokens []string, intent Intent, learned bool) PulseResult {
	var thought thinking.Thought
	var ok bool
	thought, ok = h.Thinking.ThinkAbout(prompt, contextTokens)
	if h.WebSearchEnabled && h.WebSearch != nil && h.WebSearch.ShouldSearch(thought.Confidence, len(h.Thinking.LastState.UnknownNodes), len(thought.Conflicts)) {
		thought.NeedsWebSearch = true
		results, err := h.WebSearch.Perceive(ctx, prompt)
		if err == nil {
			for _, r := range results { signal := perception.FromWebSearch(r.Source, r.Tokens, r.Confidence); h.Learning.Assimilate(signal.RawText, signal.Confidence, 0.7) }
			thought, ok = h.Thinking.ThinkAbout(prompt, contextTokens)
		}
	}
	interp := h.Decision.Resolve(thought)
	answer := ""
	if interp != nil && (len(interp.Relations) > 0 || len(interp.Nodes) > 0) {
		if interp.EvalStatus != "" { answer = h.Language.RealizeEvaluation(interp, thought.Confidence, thought.NeedsWebSearch) } else { answer = h.Language.Realize(interp, thought.Confidence, thought.NeedsWebSearch) }
	} else {
		best := h.Decision.Decide(thought.Hypotheses)
		answer = h.Language.Generate(best, thought.Confidence, thought.NeedsWebSearch, thought.Inferences, false)
	}
	return PulseResult{Answer: answer, Concepts: thought.Concepts, Confidence: thought.Confidence, NeedsWebSearch: thought.NeedsWebSearch, Learned: learned, Success: ok, Hypotheses: thought.Hypotheses, Intent: string(intent), Path: "legacy_compatibility", InterpretationSource: "legacy-compatibility"}
}
