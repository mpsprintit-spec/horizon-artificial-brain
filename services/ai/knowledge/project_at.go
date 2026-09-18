package knowledge

import "time"

// ProjectVectorAt projects a numeric observation using event time. This is
// the deterministic path used by runtime replay and temporal cognition.
func (k *KnowledgeBase) ProjectVectorAt(vector NeuralVector, threshold float64, now time.Time) (NodeID, float64, bool, error) {
	if k == nil {
		return 0, 0, false, ErrNilBrain
	}
	node, created, score, err := k.Registry.GetOrCreateRepresentationAt(vector, threshold, now)
	if err != nil {
		return 0, 0, false, err
	}
	if !created {
		score = NewNeuralVector(node.Representation).Similarity(vector)
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	node.Activation = score
	node.LastActivation = now
	return node.ID, score, created, nil
}
