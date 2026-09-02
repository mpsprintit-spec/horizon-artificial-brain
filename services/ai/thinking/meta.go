package thinking

import "github.com/project-horizon/horizon-core/services/ai/understanding"



type MetaCognitiveLayer struct {
	MinConfidence float64
	MinDepth      int
}
type MetaAssessment struct {
	NeedReprocess  bool
	NeedPerception bool
	NeedWebSearch  bool
	Reasons        []string
}

func NewMetaCognitiveLayer() *MetaCognitiveLayer {
	return &MetaCognitiveLayer{MinConfidence: 0.45, MinDepth: 2}
}
func (m *MetaCognitiveLayer) Evaluate(state CognitiveState, rep understanding.CognitiveRepresentation) MetaAssessment {
	var a MetaAssessment
	if state.ReasoningDepth < m.MinDepth {
		a.NeedReprocess = true
		a.Reasons = append(a.Reasons, "reasoning too short")
	}
	if state.Confidence < m.MinConfidence {
		a.NeedPerception = true
		a.NeedWebSearch = true
		a.Reasons = append(a.Reasons, "confidence low")
	}
	if len(state.ConflictNodes) > 0 || len(rep.CompetingNodes) > 0 {
		a.NeedReprocess = true
		a.Reasons = append(a.Reasons, "conflict detected")
	}
	if len(state.UnknownNodes) > 0 {
		a.NeedPerception = true
		a.NeedWebSearch = true
		a.Reasons = append(a.Reasons, "unknown nodes")
	}
	return a
}
