package knowledge

import "time"

// RecordLearningSignal converts a prediction outcome into persistent learning
// state. It does not assign semantic meaning to the outcome; it only changes
// numeric learning dynamics used by later experience processing.
func (k *KnowledgeBase) RecordLearningSignal(predictionError float64, now time.Time) {
	if k == nil {
		return
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	predictionError = clamp01(predictionError)

	k.mu.Lock()
	defer k.mu.Unlock()

	if k.BrainState.PredictionState == nil {
		k.BrainState.PredictionState = map[string]float64{}
	}
	if k.BrainState.PlasticityState == nil {
		k.BrainState.PlasticityState = map[string]float64{}
	}
	if k.BrainState.MemoryState == nil {
		k.BrainState.MemoryState = map[string]float64{}
	}

	// Prediction error is retained as a continuous signal. Higher error raises
	// plasticity and memory priority; low error gradually reduces the urgency
	// to modify existing structure.
	k.BrainState.PredictionState["last_error"] = predictionError
	k.BrainState.PredictionState["last_update_unix"] = float64(now.UnixNano())

	baseline := k.BrainState.PlasticityState["baseline"]
	if baseline <= 0 {
		baseline = 0.8
	}
	plasticity := clamp01(baseline + 0.35*predictionError)
	k.BrainState.PlasticityState["current"] = plasticity
	k.BrainState.PlasticityState["error_drive"] = predictionError

	retention := k.BrainState.MemoryState["retention"]
	if retention <= 0 {
		retention = 0.6
	}
	priority := clamp01(retention + 0.4*predictionError)
	k.BrainState.MemoryState["last_priority"] = priority
	k.BrainState.MemoryState["last_update_unix"] = float64(now.UnixNano())
}
