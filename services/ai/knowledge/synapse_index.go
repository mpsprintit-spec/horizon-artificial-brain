package knowledge

import (
	"encoding/json"
	"fmt"
	"time"
)

// SynapseList stores multiple independent connections to the same target.
type SynapseList []*Synapse

// SynapseKey is retained for legacy semantic-channel indexing only. The
// ontology-free substrate must not use semantic kind as connection identity.
func SynapseKey(target NodeID, kind RelationKind, inhibitory bool) string {
	pol := "exc"
	if inhibitory {
		pol = "inh"
	}
	return fmt.Sprintf("%d|%s|%s", target, kind, pol)
}

// Find matches a legacy semantic channel. New substrate code should use
// FindDynamic instead.
func (list SynapseList) Find(kind RelationKind, inhibitory bool) *Synapse {
	for _, s := range list {
		if s != nil && s.Kind == kind && s.Inhibitory == inhibitory {
			return s
		}
	}
	return nil
}

// FindDynamic returns the ontology-free connection for a target/polarity.
// Dynamic connections have no semantic Kind.
func (list SynapseList) FindDynamic(inhibitory bool) *Synapse {
	for _, s := range list {
		if s != nil && s.Kind == "" && s.Inhibitory == inhibitory {
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

func (n *ConceptNode) SynapsesTo(target NodeID) SynapseList {
	if n == nil {
		return nil
	}
	return n.Synapses[target]
}

func (n *ConceptNode) FindSynapse(target NodeID, kind RelationKind, inhibitory bool) *Synapse {
	return n.SynapsesTo(target).Find(kind, inhibitory)
}

// FindDynamicSynapse returns the semantic-free connection to target.
func (n *ConceptNode) FindDynamicSynapse(target NodeID, inhibitory bool) *Synapse {
	return n.SynapsesTo(target).FindDynamic(inhibitory)
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

func NormalizeSynapses(n *ConceptNode) {
	if n == nil {
		return
	}
	if n.Synapses == nil {
		n.Synapses = make(map[NodeID]SynapseList)
	}
}

// UnmarshalJSON accepts both legacy {"targetId": {synapse}} and modern
// {"targetId": [synapse,...]} representations.
func (n *ConceptNode) UnmarshalJSON(data []byte) error {
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
		var list []*Synapse
		if err := json.Unmarshal(blob, &list); err == nil {
			n.Synapses[tid] = list
			continue
		}
		var one Synapse
		if err := json.Unmarshal(blob, &one); err == nil {
			cp := one
			n.Synapses[tid] = SynapseList{&cp}
		}
	}
	return nil
}
