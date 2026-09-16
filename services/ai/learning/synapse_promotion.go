package learning

import (
	"errors"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// SynapsePromotion identifies the exact dynamic connection affected by a
// validated outcome. No semantic RelationKind is required.
type SynapsePromotion struct {
	SourceNodeID knowledge.NodeID
	TargetNodeID knowledge.NodeID
	Inhibitory    bool
}

// PromoteSynapse reinforces only an existing dynamic synapse. Promotion never
// invents a causal connection from an outcome alone.
func PromoteSynapse(brain *knowledge.Brain, target SynapsePromotion, evidence Evidence, now time.Time) error {
	if brain == nil || brain.Registry == nil {
		return errors.New("brain is not initialized")
	}
	source := brain.Registry.GetByID(target.SourceNodeID)
	if source == nil {
		return errors.New("synapse source node not found")
	}
	list := source.Synapses[target.TargetNodeID]
	var synapse *knowledge.Synapse
	for i := range list {
		candidate := list[i]
		if candidate != nil && candidate.Kind == "" && candidate.Inhibitory == target.Inhibitory {
			synapse = candidate
			break
		}
	}
	if synapse == nil {
		return errors.New("dynamic synapse not found")
	}
	if evidence.Confidence <= 0 || evidence.Reliability <= 0 {
		return nil
	}
	gain := 0.10 * clampPromotion(evidence.Confidence) * clampPromotion(evidence.Reliability)
	synapse.Weight = clampPromotion(synapse.Weight + gain)
	synapse.Confidence = clampPromotion(synapse.Confidence + 0.05*evidence.Confidence)
	synapse.Frequency++
	synapse.LastActivation = now.UTC()
	if synapse.Dynamic.Frequency == 0 {
		synapse.Dynamic.Weight = synapse.Weight
		synapse.Dynamic.Confidence = synapse.Confidence
		synapse.Dynamic.Frequency = synapse.Frequency
		synapse.Dynamic.LastActivation = synapse.LastActivation
		synapse.Dynamic.LastModification = synapse.LastActivation
	} else {
		synapse.Dynamic.Weight = synapse.Weight
		synapse.Dynamic.Confidence = synapse.Confidence
		synapse.Dynamic.Frequency = synapse.Frequency
		synapse.Dynamic.LastActivation = synapse.LastActivation
		synapse.Dynamic.LastModification = synapse.LastActivation
	}
	return nil
}
