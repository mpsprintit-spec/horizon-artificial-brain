package knowledge

import "time"

// NodeID is the stable internal identifier of one unique neural token node.
type NodeID int64

// ConceptNode represents exactly one token in Horizon's neural semantic memory.
type ConceptNode struct {
	ID                NodeID              `json:"id"`
	Token             string              `json:"token"`
	Activation        float64             `json:"activation"`
	RestingActivation float64             `json:"resting_activation"`
	Threshold         float64             `json:"threshold"`
	Frequency         int64               `json:"frequency"`
	Importance        float64             `json:"importance"`
	Plasticity        float64             `json:"plasticity"`
	UsageHistory      []time.Time         `json:"usage_history,omitempty"`
	LastActivation    time.Time           `json:"last_activation,omitempty"`
	Synapses          map[NodeID]SynapseList `json:"synapses"`
}

func newConceptNode(id NodeID, token string) *ConceptNode {
	now := time.Now().UTC()
	return &ConceptNode{
		ID:                id,
		Token:             token,
		Activation:        0,
		RestingActivation: 0.05,
		Threshold:         0.25,
		Frequency:         0,
		Importance:        0.5,
		Plasticity:        0.3,
		UsageHistory:      []time.Time{now},
		LastActivation:    now,
		Synapses:          make(map[NodeID]SynapseList),
	}
}
