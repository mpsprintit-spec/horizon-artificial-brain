package runtime

import k "github.com/project-horizon/horizon-core/services/ai/knowledge"

func (r *BrainRuntime) PreviewPhase4V5(req k.Phase2ProbeV5, drop k.Phase4DropV5) (k.Phase4PredictionV5, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if e := r.ensureWritableLocked(); e != nil {
		return k.Phase4PredictionV5{}, e
	}
	return r.brain.PreviewPhase4V5(req, drop)
}
func (r *BrainRuntime) PreviewPhase4DiagnosticV5(req k.Phase2ProbeV5, drop k.Phase4DropV5, do k.Phase3InterventionV5) (k.Phase4PredictionV5, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if e := r.ensureWritableLocked(); e != nil {
		return k.Phase4PredictionV5{}, e
	}
	return r.brain.PreviewPhase4DiagnosticV5(req, drop, do)
}
