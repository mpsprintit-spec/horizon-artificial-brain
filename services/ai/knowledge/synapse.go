package knowledge

import "time"

// RelationKind is a legacy semantic annotation kept only for compatibility.
// It is NOT part of Horizon's neural substrate. New substrate code must use
// DynamicState and must not require a semantic relation vocabulary.
type RelationKind string

const (
	RelationAssociation  RelationKind = "association"
	RelationCause        RelationKind = "cause"
	RelationFunction     RelationKind = "function"
	RelationPartWhole    RelationKind = "part_whole"
	RelationTime         RelationKind = "time"
	RelationLocation     RelationKind = "location"
	RelationAffix        RelationKind = "affix"
	RelationContrast     RelationKind = "contrast"
	RelationCondition    RelationKind = "condition"
	RelationSequence     RelationKind = "sequence"
	RelationIsA          RelationKind = "is_a"
	RelationHas          RelationKind = "has"
	RelationCanDo        RelationKind = "can_do"
	RelationEquivalentTo RelationKind = "equivalent_to"
)

// Synapse is a dynamic neural connection. Kind is optional legacy metadata;
// substrate behavior must never depend on its value.
type Synapse struct {
	TargetID       NodeID       `json:"target_id"`
	Kind           RelationKind `json:"kind,omitempty"`
	Weight         float64      `json:"weight"`
	Activation     float64      `json:"activation"`
	Frequency      int64        `json:"frequency"`
	Confidence     float64      `json:"confidence"`
	Inhibitory     bool         `json:"inhibitory"`
	LastActivation time.Time    `json:"last_activation,omitempty"`

	// Dynamic contains adaptive state independent of semantic labels.
	Dynamic DynamicState `json:"dynamic"`
}

// IsDynamic reports whether this connection belongs to the ontology-free
// substrate. A dynamic connection may still carry legacy Kind metadata while
// migrating existing data and consumers.
func (s *Synapse) IsDynamic() bool {
	return s != nil && s.Kind == ""
}
