package knowledge

// Brain is the canonical name for Horizon's single persistent neural
// substrate. It is an alias of the existing substrate type so the migration
// does not create a second storage implementation or duplicate state.
//
// All long-lived neural state belongs here: nodes, dynamic synapses, temporal
// patterns, activation-related state, and their persisted representation.
// Experience sources are inputs to learning and are not additional memory
// stores.
type Brain = KnowledgeBase

// NewBrain creates the single Horizon brain storage substrate.
func NewBrain() *Brain {
	return NewKnowledgeBase()
}
