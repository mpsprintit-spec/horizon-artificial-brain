package runtime

import k "github.com/project-horizon/horizon-core/services/ai/knowledge"

func (r *BrainRuntime) PreviewPhase4TerminalFeaturesV5(features [][]float64, draws []float64, greedy bool, drop k.Phase4DropV5) ([]k.Phase4PredictionV5, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if e := r.ensureWritableLocked(); e != nil {
		return nil, e
	}
	return r.brain.PreviewPhase4TerminalFeaturesV5(features, draws, greedy, drop)
}
