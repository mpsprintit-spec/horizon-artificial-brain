package knowledge

import "time"

// NeuralTrace is a modality-neutral experience presented directly to the
// neural substrate. The trace contains internal neural units and their
// temporal/contextual state, not text, labels, or semantic relation types.
type NeuralTrace struct {
	Sequence  []PatternStep
	Context   []ContextFrame
	Weight    float64
	Confidence float64
	Now       time.Time
}

// LearnNeuralTrace records one experience in the canonical brain substrate.
// Existing neural units and dynamic synapses are reused and reinforced; this
// method does not create a parallel memory store or semantic ontology.
func (k *KnowledgeBase) LearnNeuralTrace(trace NeuralTrace) {
	if k == nil || len(trace.Sequence) == 0 {
		return
	}
	if trace.Now.IsZero() {
		trace.Now = time.Now().UTC()
	}
	weight := clamp01(trace.Weight)
	confidence := clamp01(trace.Confidence)
	if weight == 0 {
		weight = 1
	}
	if confidence == 0 {
		confidence = 1
	}

	for i := range trace.Sequence {
		node := k.Registry.GetByID(trace.Sequence[i].NodeID)
		if node == nil {
			continue
		}
		node.Activation = clamp01(trace.Sequence[i].Activation)
		node.LastActivation = trace.Now
		if i == 0 {
			continue
		}
		previous := k.Registry.GetByID(trace.Sequence[i-1].NodeID)
		if previous == nil || previous.ID == node.ID {
			continue
		}
		k.Connect(previous, node, weight, confidence, false)
	}

	k.Patterns.LearnTrace(trace.Sequence, trace.Context, 0, weight, confidence)
}
