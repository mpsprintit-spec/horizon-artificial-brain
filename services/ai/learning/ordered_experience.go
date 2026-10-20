package learning

import (
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// LearnExperienceWithDynamics is used by the versioned runtime. Historical
// dynamics retain their original member-set behavior; ordered dynamics learn
// conditional neural pattern channels, never grammatical rules or word roles.
func (l *LearningUnit) LearnExperienceWithDynamics(experience Experience, now time.Time, surface knowledge.SurfaceEncoding, dynamics knowledge.DynamicsEncoding) {
	if dynamics == knowledge.LegacyDynamics {
		l.LearnExperienceWithEncoding(experience, now, surface)
		return
	}
	if l == nil || l.Kb == nil || !dynamics.HasOrderedObservations() {
		return
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()
	if dynamics.HasPhysicalMemory() {
		l.Kb.ApplyMemoryDynamicsWithEncoding(now, dynamics)
	}
	frames := func(words []string, populations [][]knowledge.NodeID) [][]knowledge.NodeID {
		if len(populations) > 0 {
			out := (&knowledge.ObservationTrace{Stimulus: populations}).Canonical().Stimulus
			for _, frame := range out {
				for _, id := range frame {
					if node := l.Kb.Registry.GetByID(id); node != nil {
						node.Frequency++
						node.LastActivation = now
					}
				}
			}
			return out
		}
		out := make([][]knowledge.NodeID, 0, len(words))
		for _, word := range words {
			if node := l.Kb.StoreSurfaceAt(word, now, surface); node != nil {
				out = append(out, []knowledge.NodeID{node.ID})
			}
		}
		return out
	}
	context := frames(experience.ContextSequence, experience.NeuralContext)
	data := frames(experience.DataSequence, experience.NeuralData)
	sequence := frames(experience.Sequence, experience.NeuralSequence)
	if len(sequence) == 0 {
		return
	}
	weight, confidence := clamp01(experience.Weight), clamp01(experience.Confidence)
	if weight == 0 {
		weight = .5
	}
	if confidence == 0 {
		confidence = .5
	}
	for i := 1; i < len(sequence); i++ {
		previous := l.Kb.Registry.GetByID(sequence[i-1][0])
		targets := sequence[i][:1]
		if i == len(sequence)-1 {
			targets = sequence[i]
		}
		// Conditional counterevidence belongs to the full learned pattern,
		// not the shared context-free projection of a later sequence step.
		if dynamics == knowledge.ObservedFeedbackDynamics && previous != nil && i == 1 && len(context)+len(data) == 0 {
			l.Kb.ApplyObservedAssociationFeedback(sequence[i-1], sequence[i], confidence, now)
		}
		for _, id := range targets {
			l.Kb.ConnectAt(previous, l.Kb.Registry.GetByID(id), weight, confidence, false, now)
		}
	}
	trace := &knowledge.ObservationTrace{Stimulus: sequence, Context: context, Data: data}
	if dynamics == knowledge.ObservedFeedbackDynamics && len(sequence) > 1 {
		cue := &knowledge.ObservationTrace{Stimulus: sequence[:len(sequence)-1], Context: context, Data: data}
		channels := l.Kb.Patterns.PredictRecurrent(cue, nil, now)
		actual := map[knowledge.NodeID]float64{}
		for _, id := range sequence[len(sequence)-1] {
			actual[id] = 1
		}
		l.Kb.Patterns.ApplyObservedPatternFeedback(channels.Credit, actual, confidence)
	}
	for _, result := range sequence[len(sequence)-1] {
		pattern := l.Kb.Patterns.LearnOrderedTraceWithEvidence(trace, result, weight, confidence, experience.evidence(now))
		if pattern != nil {
			pattern.Confidence = knowledge.CalibratedConfidence(confidence, pattern.Evidence)
		}
	}
}

func (e Experience) HasOrderedFields() bool {
	return len(e.ContextSequence)+len(e.DataSequence)+len(e.NeuralSequence)+len(e.NeuralContext)+len(e.NeuralData) > 0
}
