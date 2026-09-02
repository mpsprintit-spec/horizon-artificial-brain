package knowledge

import (
	"encoding/json"
	"fmt"
	"time"
)

// SynapseList is multiple independent relations to the same target.
type SynapseList []*Synapse

// SynapseKey uniquely identifies one relation channel between two nodes.
func SynapseKey(target NodeID, kind RelationKind, inhibitory bool) string {
	pol := "exc"
	if inhibitory {
		pol = "inh"
	}
	return fmt.Sprintf("%d|%s|%s", target, kind, pol)
}

// Find matches kind+polarity on a target; nil if absent.
func (list SynapseList) Find(kind RelationKind, inhibitory bool) *Synapse {
	for _, s := range list {
		if s != nil && s.Kind == kind && s.Inhibitory == inhibitory {
			return s
		}
	}
	return nil
}

// All returns non-nil synapses.
func (list SynapseList) All() []*Synapse {
	var out []*Synapse
	for _, s := range list {
		if s != nil {
			out = append(out, s)
		}
	}
	return out
}

// OutboundAll flattens all outbound synapses from a node.
func (n *ConceptNode) OutboundAll() []*Synapse {
	if n == nil {
		return nil
	}
	var out []*Synapse
	for _, list := range n.Synapses {
		out = append(out, list.All()...)
	}
	return out
}

// SynapsesTo returns all relations toward target.
func (n *ConceptNode) SynapsesTo(target NodeID) SynapseList {
	if n == nil {
		return nil
	}
	return n.Synapses[target]
}

// FindSynapse returns the exact relation channel.
func (n *ConceptNode) FindSynapse(target NodeID, kind RelationKind, inhibitory bool) *Synapse {
	return n.SynapsesTo(target).Find(kind, inhibitory)
}

// conceptNodeJSON supports backward-compatible load of map[NodeID]*Synapse.
type conceptNodeJSON struct {
	ID                NodeID                     `json:"id"`
	Token             string                     `json:"token"`
	Activation        float64                    `json:"activation"`
	RestingActivation float64                    `json:"resting_activation"`
	Threshold         float64                    `json:"threshold"`
	Frequency         int64                      `json:"frequency"`
	Importance        float64                    `json:"importance"`
	Plasticity        float64                    `json:"plasticity"`
	UsageHistory      []interface{}              `json:"usage_history,omitempty"`
	LastActivation    interface{}                `json:"last_activation,omitempty"`
	Synapses          map[NodeID]json.RawMessage `json:"synapses"`
}

// NormalizeSynapses ensures map values are lists (call after naive unmarshal if needed).
func NormalizeSynapses(n *ConceptNode) {
	if n == nil {
		return
	}
	if n.Synapses == nil {
		n.Synapses = make(map[NodeID]SynapseList)
	}
}

// UnmarshalJSON accepts both legacy {"targetId": {synapse}} and new {"targetId": [synapse,...]}.
func (n *ConceptNode) UnmarshalJSON(data []byte) error {
	type alias ConceptNode
	// Try modern shape first via raw map
	var raw struct {
		ID                NodeID                     `json:"id"`
		Token             string                     `json:"token"`
		Activation        float64                    `json:"activation"`
		RestingActivation float64                    `json:"resting_activation"`
		Threshold         float64                    `json:"threshold"`
		Frequency         int64                      `json:"frequency"`
		Importance        float64                    `json:"importance"`
		Plasticity        float64                    `json:"plasticity"`
		UsageHistory      []time.Time                `json:"usage_history"`
		LastActivation    time.Time                  `json:"last_activation"`
		Synapses          map[NodeID]json.RawMessage `json:"synapses"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	n.ID = raw.ID
	n.Token = raw.Token
	n.Activation = raw.Activation
	n.RestingActivation = raw.RestingActivation
	n.Threshold = raw.Threshold
	n.Frequency = raw.Frequency
	n.Importance = raw.Importance
	n.Plasticity = raw.Plasticity
	n.UsageHistory = raw.UsageHistory
	n.LastActivation = raw.LastActivation
	n.Synapses = make(map[NodeID]SynapseList)
	for tid, blob := range raw.Synapses {
		if len(blob) == 0 || string(blob) == "null" {
			continue
		}
		// array form
		var list []*Synapse
		if err := json.Unmarshal(blob, &list); err == nil {
			n.Synapses[tid] = list
			continue
		}
		// legacy single object
		var one Synapse
		if err := json.Unmarshal(blob, &one); err == nil {
			cp := one
			n.Synapses[tid] = SynapseList{&cp}
			continue
		}
	}
	return nil
}
