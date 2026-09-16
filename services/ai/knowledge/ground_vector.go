package knowledge

import "errors"

var (
	ErrNilKnowledgeBase = errors.New("knowledge base is nil")
	ErrEmptyNeuralVector = errors.New("neural vector is empty")
)

// GroundVector maps an encoded observation onto the canonical shared neural
// substrate. Similar representations are reused; novel vectors remain
// structural candidates. No semantic identity or truth is assigned here.
func (b *KnowledgeBase) GroundVector(vector NeuralVector) (GroundingResult, error) {
	if b == nil {
		return GroundingResult{}, ErrNilKnowledgeBase
	}
	if vector.Empty() {
		return GroundingResult{}, ErrEmptyNeuralVector
	}

	nodeID, similarity, existing := b.GetOrCreateRepresentation(vector.Values)
	return GroundingResult{
		NodeID: nodeID,
		Existing: existing,
		Similarity: similarity,
	}, nil
}
