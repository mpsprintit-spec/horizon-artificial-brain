package knowledge

import "time"

// ConnectAt applies the same dynamic connection update as Connect, but binds
// the mutation timestamps to the supplied event time. This keeps replay from
// depending on wall-clock time while preserving the existing Brain substrate.
func (k *KnowledgeBase) ConnectAt(source, target *ConceptNode, weight, confidence float64, inhibitory bool, now time.Time) {
	if k == nil || source == nil || target == nil {
		return
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	k.Connect(source, target, weight, confidence, inhibitory)
	if source.Synapses == nil {
		return
	}
	s := source.Synapses[target.ID].FindDynamic(inhibitory)
	if s == nil {
		return
	}
	s.Dynamic.LastActivation = now
	s.Dynamic.LastModification = now
	s.LastActivation = now
	syncSynapseLegacyState(s)
	source.LastActivation = now
}
