package learning

import "github.com/project-horizon/horizon-core/services/ai/knowledge"

// TouchedRelation identifies exact relation channel touched this cycle.
type TouchedRelation struct {
	SourceID   knowledge.NodeID
	TargetID   knowledge.NodeID
	Kind       knowledge.RelationKind
	Inhibitory bool
}

// Confirm strengthens only the exact channels recorded in LastTouched.
func (l *LearningUnit) Confirm() {
	for _, t := range l.LastTouched {
		source := l.Kb.Registry.GetByID(t.SourceID)
		if source == nil {
			continue
		}
		s := source.FindSynapse(t.TargetID, t.Kind, t.Inhibitory)
		if s == nil {
			continue
		}
		s.Confidence = 1 - (1-s.Confidence)*(1-0.95)
	}
}

// Retract weakens/deletes only the exact channels in LastTouched.
func (l *LearningUnit) Retract() {
	for _, t := range l.LastTouched {
		source := l.Kb.Registry.GetByID(t.SourceID)
		if source == nil {
			continue
		}
		list := source.SynapsesTo(t.TargetID)
		s := list.Find(t.Kind, t.Inhibitory)
		if s == nil {
			continue
		}
		if s.Confidence < 0.5 {
			// remove only this channel
			var kept []*knowledge.Synapse
			for _, x := range list {
				if x == s {
					continue
				}
				kept = append(kept, x)
			}
			if len(kept) == 0 {
				delete(source.Synapses, t.TargetID)
			} else {
				source.Synapses[t.TargetID] = kept
			}
		} else {
			s.Confidence *= 0.3
		}
	}
}
