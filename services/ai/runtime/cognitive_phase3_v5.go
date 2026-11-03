package runtime

import (
	"errors"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// Offline counterfactual evaluation only. No production selector invokes it.
func (r *BrainRuntime) PreviewPhase3DiagnosticV5(req knowledge.Phase2ProbeV5, do knowledge.Phase3InterventionV5) (knowledge.Phase2PredictionV5, error) {
	if r == nil || r.brain == nil {
		return knowledge.Phase2PredictionV5{}, errors.New("runtime unavailable")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if e := r.ensureWritableLocked(); e != nil {
		return knowledge.Phase2PredictionV5{}, e
	}
	return r.brain.PreviewPhase3DiagnosticV5(req, do)
}
