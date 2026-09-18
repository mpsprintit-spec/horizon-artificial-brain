package knowledge

import "time"

// ConnectAt applies a dynamic connection update using the supplied event time.
// The mutation is performed directly so replay never performs an intermediate
// wall-clock mutation that another goroutine could observe.
func (k *KnowledgeBase) ConnectAt(source, target *ConceptNode, weight, confidence float64, inhibitory bool, now time.Time) {
	if k == nil || source == nil || target == nil {
		return
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	k.Registry.mu.Lock()
	defer k.Registry.mu.Unlock()

	if source.Synapses == nil {
		source.Synapses = make(map[NodeID]SynapseList)
	}
	list := source.Synapses[target.ID]
	s := list.FindDynamic(inhibitory)
	if s == nil {
		s = &Synapse{TargetID: target.ID, Weight: clamp01(weight), Confidence: clamp01(confidence), Inhibitory: inhibitory}
		s.Dynamic.Weight = clamp01(weight)
		s.Dynamic.Confidence = clamp01(confidence)
		s.Dynamic.Activation = 1
		s.Dynamic.Frequency = 0
		s.Dynamic.LastActivation = now
		s.Dynamic.LastModification = now
		s.Dynamic.LastEligibilityUpdate = now
		s.Dynamic.Frequency = 1
		s.Dynamic.Eligibility = 1
		s.Weight = s.Dynamic.Weight
		s.Confidence = s.Dynamic.Confidence
		s.Activation = s.Dynamic.Activation
		s.Frequency = s.Dynamic.Frequency
		s.LastActivation = now
		source.Synapses[target.ID] = append(list, s)
	} else {
		s.Dynamic.Reinforce(weight, confidence, 1, now)
		syncSynapseLegacyState(s)
	}
	source.LastActivation = now
}
