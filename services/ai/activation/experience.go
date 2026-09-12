package activation

import (
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

const (
	experienceConnectionRate = 0.08
	experienceMinActivation  = 0.20
	experienceMaxWeight      = 0.95
)

// Experience is the modality-neutral internal form of an experience.
type Experience struct {
	Activations map[knowledge.NodeID]float64
	Confidence  map[knowledge.NodeID]float64
	Now         time.Time
}

// LearnFromExperience incorporates an experienced internal state into the same
// neural substrate used by every other experience.
func (e *Engine) LearnFromExperience(exp Experience) {
	if e == nil || e.Memory == nil || len(exp.Activations) == 0 {
		return
	}
	if exp.Now.IsZero() {
		exp.Now = time.Now().UTC()
	}

	active := make([]*knowledge.ConceptNode, 0, len(exp.Activations))
	for id, activation := range exp.Activations {
		if activation < experienceMinActivation {
			continue
		}
		n := e.Memory.Registry.GetByID(id)
		if n == nil {
			continue
		}
		n.Activation = clamp01(activation)
		n.LastActivation = exp.Now
		active = append(active, n)
	}
	if len(active) < 2 {
		return
	}

	for _, source := range active {
		for _, target := range active {
			if source.ID == target.ID {
				continue
			}
			coactivation := clamp01(exp.Activations[source.ID] * exp.Activations[target.ID])
			if coactivation < experienceMinActivation {
				continue
			}

			synapse := source.FindDynamicSynapse(target.ID, false)
			if synapse == nil {
				if coactivation < 0.60 {
					continue
				}
				e.Memory.Connect(source, target, experienceConnectionRate*coactivation, 0.5, false)
				continue
			}

			delta := experienceConnectionRate * coactivation
			synapse.Dynamic.Weight = clamp(synapse.Dynamic.Weight+delta, 0, experienceMaxWeight)
			synapse.Dynamic.Confidence = clamp01(synapse.Dynamic.Confidence + delta*0.5)
			synapse.Dynamic.Eligibility = coactivation
			synapse.Dynamic.Frequency++
			synapse.Dynamic.LastActivation = exp.Now
			synapse.Dynamic.LastModification = exp.Now
			synapse.Weight = synapse.Dynamic.Weight
			synapse.Confidence = synapse.Dynamic.Confidence
			synapse.Frequency = synapse.Dynamic.Frequency
			synapse.LastActivation = synapse.Dynamic.LastActivation
		}
	}
}
