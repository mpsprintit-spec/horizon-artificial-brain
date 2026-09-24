package knowledge

import "time"

// NodeID is the stable internal identifier of one neural unit.
type NodeID int64

// ConceptNode is a physical/internal neural unit in the shared substrate.
// It carries only substrate state and numeric representation. Semantic meaning
// is distributed across populations, connectivity, temporal dynamics, and
// context; no lexical token is stored on the unit.
type ConceptNode struct {
	ID                NodeID                `json:"id"`
	Token             string                `json:"token,omitempty"`
	Representation    []float64             `json:"representation,omitempty"`
	Activation        float64               `json:"activation"`
	RestingActivation float64               `json:"resting_activation"`
	Threshold         float64               `json:"threshold"`
	Frequency         int64                 `json:"frequency"`
	Importance        float64               `json:"importance"`
	Plasticity        float64               `json:"plasticity"`
	UsageHistory      []time.Time           `json:"usage_history,omitempty"`
	LastActivation    time.Time             `json:"last_activation,omitempty"`
	Synapses          map[NodeID]SynapseList `json:"synapses"`
}

func newConceptNode(id NodeID) *ConceptNode {
	return newConceptNodeAt(id, time.Now().UTC())
}

func newConceptNodeAt(id NodeID, now time.Time) *ConceptNode {
	if now.IsZero() { now = time.Now().UTC() } else { now = now.UTC() }
	return &ConceptNode{
		ID:                id,
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

func newRepresentationNode(id NodeID, representation []float64) *ConceptNode {
	return newRepresentationNodeAt(id, representation, time.Now().UTC())
}

func newRepresentationNodeAt(id NodeID, representation []float64, now time.Time) *ConceptNode {
	n := newConceptNodeAt(id, now)
	n.UsageHistory = nil
	n.Representation = append([]float64(nil), representation...)
	return n
}
