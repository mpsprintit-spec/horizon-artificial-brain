package dnf

import (
	"errors"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

// Plasticity applies already-validated evidence to an exact dynamic channel.
// It owns no state: the canonical Brain remains the only substrate.
func (f *Fabric) Plasticity(target learning.SynapsePromotion, evidence learning.Evidence, now time.Time) error {
	if f == nil || f.brain == nil {
		return errors.New("dnf is not initialized")
	}
	return learning.PromoteSynapse(f.brain, target, evidence, now)
}

// SynapseState exposes a snapshot of the exact dynamic channel for integration
// tests and diagnostics without creating a second representation of the brain.
func (f *Fabric) SynapseState(sourceID, targetID knowledge.NodeID, inhibitory bool) (knowledge.DynamicState, error) {
	if f == nil || f.brain == nil || f.brain.Registry == nil {
		return knowledge.DynamicState{}, errors.New("dnf is not initialized")
	}
	source := f.brain.Registry.GetByID(sourceID)
	if source == nil {
		return knowledge.DynamicState{}, errors.New("synapse source node not found")
	}
	for _, synapse := range source.Synapses[targetID] {
		if synapse != nil && synapse.IsDynamic() && synapse.Inhibitory == inhibitory {
			return synapse.Dynamic, nil
		}
	}
	return knowledge.DynamicState{}, errors.New("dynamic synapse not found")
}
