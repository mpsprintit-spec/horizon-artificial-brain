package bridge

import (
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

// answerReadout is a transport projection of the native numeric candidate.
// It excludes labels, generated text and the potentially large alternatives
// collection; active_node_ids remains the separate activity diagnostic.
type answerReadout struct {
	Status      runtime.AnswerStatus `json:"status"`
	NodeIDs     []knowledge.NodeID   `json:"node_ids"`
	Scope       string               `json:"scope,omitempty"`
	Confidence  float64              `json:"confidence"`
	Uncertainty float64              `json:"uncertainty"`
}

func readoutForAnswer(answer runtime.Answer) answerReadout {
	out := answerReadout{Status: answer.Status, NodeIDs: answer.NodeIDs, Confidence: answer.Confidence, Uncertainty: answer.Uncertainty.Level}
	if answer.PredictionAssessment != nil {
		out.Scope = answer.PredictionAssessment.Scope
	}
	return out
}
