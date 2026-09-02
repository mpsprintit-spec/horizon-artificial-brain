package knowledge

import "time"

type RelationKind string

const (
	RelationAssociation RelationKind = "association"
	RelationCause       RelationKind = "cause"
	RelationFunction    RelationKind = "function"
	RelationPartWhole   RelationKind = "part_whole"
	RelationTime        RelationKind = "time"
	RelationLocation    RelationKind = "location"
	RelationAffix       RelationKind = "affix"
	RelationContrast    RelationKind = "contrast"
	RelationCondition   RelationKind = "condition"
	RelationSequence    RelationKind = "sequence"
	RelationIsA         RelationKind = "is_a"
	RelationHas         RelationKind = "has"
	RelationCanDo       RelationKind = "can_do"
	RelationEquivalentTo RelationKind = "equivalent_to"
)

// Synapse is a weighted neural connection, not a definition or sentence store.
type Synapse struct {
	TargetID       NodeID       `json:"target_id"`
	Kind           RelationKind `json:"kind"`
	Weight         float64      `json:"weight"`
	Activation     float64      `json:"activation"`
	Frequency      int64        `json:"frequency"`
	Confidence     float64      `json:"confidence"`
	Inhibitory     bool         `json:"inhibitory"`
	LastActivation time.Time    `json:"last_activation,omitempty"`
}
