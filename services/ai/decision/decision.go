package decision

import "github.com/project-horizon/horizon-core/services/ai/thinking"

type Engine struct{}

func NewEngine() *Engine { return &Engine{} }

func (e *Engine) Decide(hypotheses []thinking.Hypothesis) thinking.Hypothesis {
	var best thinking.Hypothesis
	for _, h := range hypotheses {
		if h.Confidence > best.Confidence {
			best = h
		}
	}
	return best
}

func (e *Engine) DecideInterpretation(candidates []thinking.Interpretation) *thinking.Interpretation {
	return thinking.SelectBestInterpretation(candidates)
}

// Resolve returns I*. Evaluation status must already be set by thinking/FSU decision path.
func (e *Engine) Resolve(thought thinking.Thought) *thinking.Interpretation {
	if thought.BestInterpretation != nil {
		return thought.BestInterpretation
	}
	if len(thought.Interpretations) > 0 {
		return e.DecideInterpretation(thought.Interpretations)
	}
	best := e.Decide(thought.Hypotheses)
	if len(best.Nodes) == 0 {
		return nil
	}
	return &thinking.Interpretation{
		FocusID:       best.Nodes[0],
		Nodes:         best.Nodes,
		TotalScore:    best.Confidence,
		SemanticScore: best.Confidence,
		EvidenceNotes: []string{"fallback from legacy hypothesis"},
		EvalStatus:    thinking.EvalUnknown,
	}
}
