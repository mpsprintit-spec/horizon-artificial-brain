package activation

import (
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// PredictionAssessment reads the frozen real-observation forecast. Thought
// cannot create extra experience support or replace it with simulated evidence.
// It is derived from persisted neural state and needs no additional checkpoint cache.
func (e *Engine) PredictionAssessment(now time.Time, dynamics knowledge.DynamicsEncoding) *knowledge.PredictionAssessment {
	if dynamics != knowledge.EvidenceUncertaintyDynamics {
		return nil
	}
	e.mu.RLock()
	boundary := e.observationPrediction.Clone()
	e.mu.RUnlock()
	if boundary == nil || now.Before(boundary.At) || now.Sub(boundary.At) > knowledge.ContextMaxAge {
		return knowledge.UnknownPredictionAssessment()
	}
	source := sortedNodeIDs(boundary.Source)
	return e.Memory.AssessObservedPrediction(source, boundary.Patterns, boundary.Prediction)
}
