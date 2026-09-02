package core

import (
	"fmt"
	"context"
	"strings"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/perception"
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

	case IntentConfirmation:
		// Phase 2: do not token-match for cognitive truth. Fall through to H2 + FSU.
		// Legacy extractConfirmationParts is not used as understanding authority.
	}

	// Belajar HANYA untuk pernyataan (Teaching) -- semua bentuk pertanyaan/
	// perintah lain tidak dianggap fakta baru.
	learned := false
	if intent == IntentTeaching {
		signals, _ := h.Perception.Perceive(prompt)
		if len(signals) > 0 {
			h.Learning.Assimilate(signals[0].RawText, 0.85, 0.35)
			learned = true
		}
	}

	// Horizon 2 P0: always retain stimulus tokens in the available subgraph.
	// Whether a token becomes Focus/relevant is decided by Interpretation coherence,
	// not by intent-trigger routing (Recall vs other).
	var thought thinking.Thought
	var ok bool
	thought, ok = h.Thinking.ThinkAbout(prompt, contextTokens)

	if h.WebSearch != nil && h.WebSearch.ShouldSearch(thought.Confidence, len(h.Thinking.LastState.UnknownNodes), len(thought.Conflicts)) {
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

	// Horizon 2 P0: Decision works only on coherent understanding state (I*).
	// Focus is not overridden by string triggers (extractDefinitionTarget / extractTeachingFocus).
	// Those helpers remain in classify.go for legacy routing of confirm/retraction only.
	// Cognitive focus comes from Interpretation evaluation, not hard-coded token lists.
	interp := h.Decision.Resolve(thought)

	var answer string
	if interp != nil && (len(interp.Relations) > 0 || len(interp.Nodes) > 0) {
		// Language is pure realizer — no cognitive re-selection.
		if interp.EvalStatus != "" {
			answer = h.Language.RealizeEvaluation(interp, thought.Confidence, thought.NeedsWebSearch)
		} else {
			answer = h.Language.Realize(interp, thought.Confidence, thought.NeedsWebSearch)
		}
	} else {
		best := h.Decision.Decide(thought.Hypotheses)
		answer = h.Language.Generate(best, thought.Confidence, thought.NeedsWebSearch, thought.Inferences, false)
	}

	if ok && interp != nil && len(interp.Nodes) > 0 {
		if node := h.Knowledge.Registry.GetByID(interp.FocusID); node != nil {
			h.Execution.Dispatch(node.Token, task.Data)
		}
	}
	h.Learning.Optimize(time.Now())
var evidenceSummary []string
	for _, ps := range thought.PatternEvidence {
		var words []string
		for _, id := range ps.Members {
			if n := h.Knowledge.Registry.GetByID(id); n != nil {
				words = append(words, n.Token)
			}
		}
		resultWord := "?"
		if n := h.Knowledge.Registry.GetByID(ps.Result); n != nil {
			resultWord = n.Token
		}
		evidenceSummary = append(evidenceSummary, fmt.Sprintf("{%s} -> %s (confidence=%.2f)", strings.Join(words, ", "), resultWord, ps.Confidence))
	}

	focusTok := ""
	var fnotes []string
	var fsigs []string
	var props, constr, paths []string
	var evalSt string
	if interp != nil {
		if n := h.Knowledge.Registry.GetByID(interp.FocusID); n != nil {
			focusTok = n.Token
		}
		fnotes = append(fnotes, interp.EvidenceNotes...)
		for _, s := range interp.FunctionalSignals {
			fsigs = append(fsigs, formatFunctionalSignal(s))
		}
		for _, pr := range interp.Propositions {
			props = append(props, pr.TargetTok+" -["+string(pr.Relation)+"]-> "+pr.ObjectTok)
		}
		for _, c := range interp.Constraints {
			constr = append(constr, c.Token+" ("+c.Role+")")
		}
		for _, ep := range interp.EvidencePaths {
			paths = append(paths, thinking.FormatEvidencePath(ep))
		}
		evalSt = string(interp.EvalStatus)
	}
	for _, s := range thought.FunctionalSignals {
		line := formatFunctionalSignal(s)
		dup := false
		for _, x := range fsigs {
			if x == line {
				dup = true
				break
			}
		}
		if !dup {
			fsigs = append(fsigs, line)
		}
	}
	return PulseResult{
		Answer:             answer,
		Concepts:           thought.Concepts,
		Confidence:         thought.Confidence,
		NeedsWebSearch:     thought.NeedsWebSearch,
		Learned:            learned,
		Success:            ok,
		Hypotheses:         thought.Hypotheses,
		PatternEvidence:    evidenceSummary,
		Intent:             string(intent),
		Path:               "h2",
		FocusToken:         focusTok,
		FunctionalSignals:  fsigs,
		InterpretationNotes: fnotes,
		Propositions: props,
		Constraints: constr,
		EvidencePaths: paths,
		EvalStatus: evalSt,
	}
}


func formatFunctionalSignal(s thinking.FunctionalSignal) string {
	return s.Kind + " src=" + s.Source + " strength=" + formatFloat(s.Strength) + " note=" + s.Note
}

func formatFloat(v float64) string {
	return fmt.Sprintf("%.3f", v)
}

