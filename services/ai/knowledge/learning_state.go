package knowledge

import (
	"math"
	"time"
)

const maxExperienceTraces = 64

// ExperienceTrace records the temporal binding created by actual runtime
// observations. It contains only substrate state and provenance; it does not
// encode a semantic relation or a lexical definition.
type ExperienceTrace struct {
	PreviousPopulation []NodeID  `json:"previous_population"`
	CurrentPopulation  []NodeID  `json:"current_population"`
	PreviousAt         time.Time `json:"previous_at"`
	CurrentAt          time.Time `json:"current_at"`
	PredictionError    float64   `json:"prediction_error"`
	Plasticity         float64   `json:"plasticity"`
	MemoryPriority     float64   `json:"memory_priority"`
}

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


// ReinforceTransitionPopulation binds two consecutive observed populations.
// Plasticity and memory priority are read from the persistent BrainState so
// those signals change how strongly later experience modifies the substrate.
func (k *KnowledgeBase) ReinforceTransitionPopulation(previous, current []NodeID, previousAt, currentAt time.Time, baseStrength float64) error {
	if k == nil {
		return ErrNilBrain
	}
	if len(previous) == 0 || len(current) == 0 {
		return nil
	}
	if currentAt.IsZero() {
		currentAt = time.Now().UTC()
	}
	if previousAt.IsZero() {
		previousAt = currentAt
	}

	k.mu.Lock()
	defer k.mu.Unlock()

	plasticity := k.BrainState.PlasticityState["current"]
	if plasticity <= 0 {
		plasticity = k.BrainState.PlasticityState["baseline"]
	}
	if plasticity <= 0 {
		plasticity = 0.8
	}
	plasticity = clamp01(plasticity)

	memoryPriority := k.BrainState.MemoryState["last_priority"]
	if memoryPriority <= 0 {
		memoryPriority = k.BrainState.MemoryState["retention"]
	}
	if memoryPriority <= 0 {
		memoryPriority = 0.6
	}
	memoryPriority = clamp01(memoryPriority)

	baseStrength = clamp01(baseStrength)
	strength := clamp01(baseStrength * (0.5 + 0.5*plasticity) * (0.5 + 0.5*memoryPriority))
	confidence := clamp01(0.10 + 0.70*memoryPriority)
	eligibility := clamp01(0.50 + 0.50*plasticity)

	for _, sourceID := range previous {
		source := k.Registry.GetByID(sourceID)
		if source == nil {
			continue
		}
		if source.Synapses == nil {
			source.Synapses = make(map[NodeID]SynapseList)
		}
		for _, targetID := range current {
			if sourceID == targetID {
				continue
			}
			target := k.Registry.GetByID(targetID)
			if target == nil {
				continue
			}

			list := source.Synapses[target.ID]
			synapse := list.FindDynamic(false)
			if synapse == nil {
				synapse = &Synapse{TargetID: target.ID, Inhibitory: false}
				source.Synapses[target.ID] = append(list, synapse)
			}
			synapse.Dynamic.Reinforce(strength, confidence, eligibility, currentAt)
			syncSynapseLegacyState(synapse)
		}
		source.LastActivation = currentAt
	}

	predictionError := clamp01(k.BrainState.PredictionState["last_error"])
	trace := ExperienceTrace{
		PreviousPopulation: append([]NodeID(nil), previous...),
		CurrentPopulation:  append([]NodeID(nil), current...),
		PreviousAt:         previousAt,
		CurrentAt:          currentAt,
		PredictionError:    predictionError,
		Plasticity:         plasticity,
		MemoryPriority:     memoryPriority,
	}
	k.BrainState.ExperienceTraces = append(k.BrainState.ExperienceTraces, trace)
	if len(k.BrainState.ExperienceTraces) > maxExperienceTraces {
		start := len(k.BrainState.ExperienceTraces) - maxExperienceTraces
		k.BrainState.ExperienceTraces = append([]ExperienceTrace(nil), k.BrainState.ExperienceTraces[start:]...)
	}

	k.BrainState.PlasticityState["last_transition_strength"] = strength
	k.BrainState.MemoryState["last_transition_priority"] = memoryPriority
	return nil
}


// ApplyMemoryDynamics performs time-dependent retention on the existing
// substrate. Forgetting reduces the influence of old connections and the
// priority of old temporal traces, but does not delete historical structure.
func (k *KnowledgeBase) ApplyMemoryDynamics(now time.Time) {
	if k == nil {
		return
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	k.mu.Lock()
	defer k.mu.Unlock()

	retention := k.BrainState.MemoryState["retention"]
	if retention <= 0 {
		retention = k.BrainState.LearningPolicyState.MemoryRetention
	}
	if retention <= 0 {
		retention = 0.6
	}
	retention = clamp01(retention)

	// Retention controls the time constant: stronger retention means slower
	// forgetting. The floor prevents old traces from disappearing in one pass.
	const halfLifeHours = 24.0
	for _, source := range k.Registry.Nodes() {
		if source == nil {
			continue
		}
		for _, list := range source.Synapses {
			for _, synapse := range list {
				if synapse == nil || synapse.Dynamic.LastModification.IsZero() {
					continue
				}
				ageHours := now.Sub(synapse.Dynamic.LastModification).Hours()
				if ageHours <= 0 {
					continue
				}
				forgetRate := (1.0 - retention) / halfLifeHours
				factor := expDecay(ageHours * forgetRate)
				synapse.Dynamic.Weight = clamp01(synapse.Dynamic.Weight * factor)
				synapse.Dynamic.Confidence = clamp01(synapse.Dynamic.Confidence * (0.5 + 0.5*factor))
				synapse.Dynamic.Eligibility = clamp01(synapse.Dynamic.Eligibility * factor)
				syncSynapseLegacyState(synapse)
			}
		}
	}

	for i := range k.BrainState.ExperienceTraces {
		trace := &k.BrainState.ExperienceTraces[i]
		if trace.CurrentAt.IsZero() {
			continue
		}
		ageHours := now.Sub(trace.CurrentAt).Hours()
		if ageHours <= 0 {
			continue
		}
		forgetRate := (1.0 - retention) / halfLifeHours
		factor := expDecay(ageHours * forgetRate)
		trace.MemoryPriority = clamp01(trace.MemoryPriority * factor)
	}

	k.BrainState.MemoryState["last_decay_unix"] = float64(now.UnixNano())
	k.BrainState.MemoryState["effective_retention"] = retention
}

// expDecay is a small local exponential decay helper. It intentionally keeps
// memory dynamics numeric and independent of semantic categories.
func expDecay(x float64) float64 {
	if x <= 0 {
		return 1
	}
	return clamp01(math.Exp(-x))
}
