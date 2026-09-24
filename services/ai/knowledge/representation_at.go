package knowledge

import "time"

// GetOrCreateRepresentationAt is the event-time variant of numeric grounding.
// It keeps replayed representation activation independent of wall-clock time.
func (r *NeuralRegistry) GetOrCreateRepresentationAt(vector NeuralVector, threshold float64, now time.Time) (*ConceptNode, bool, float64, error) {
	if vector.Empty() {
		return nil, false, 0, ErrEmptyNeuralVector
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	threshold = clamp(threshold, 0, 1)

	r.mu.Lock()
	defer r.mu.Unlock()
	best := (*ConceptNode)(nil)
	bestScore := 0.0
	for _, node := range r.byID {
		if len(node.Representation) != len(vector.Values) {
			continue
		}
		score := NewNeuralVector(node.Representation).Similarity(vector)
		if score > bestScore {
			best, bestScore = node, score
		}
	}
	if best != nil && bestScore >= threshold {
		best.Frequency++
		best.LastActivation = now
		return best, false, bestScore, nil
	}
	n := newRepresentationNodeAt(r.nextID, vector.Values, now)
	n.UsageHistory = []time.Time{now}
	n.LastActivation = now
	n.Frequency = 1
	r.nextID++
	r.byID[n.ID] = n
	return n, true, 1, nil
}
