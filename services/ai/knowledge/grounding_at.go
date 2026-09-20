package knowledge

import (
	"errors"
	"strings"
	"time"
)

// GroundObservationAt is the event-time grounding path used by the cognitive
// orchestrator. It prevents wall-clock timestamps from contaminating replay.
func (k *KnowledgeBase) GroundObservationAt(token, source, modality string, threshold float64, now time.Time) (GroundedRepresentation, error) {
	if k == nil {
		return GroundedRepresentation{}, ErrNilBrain
	}
	canonical := strings.TrimSpace(token)
	if canonical == "" {
		return GroundedRepresentation{}, errors.New("observation token is empty")
	}

	vector := observationVector(canonical, modality)

	// If learning created a canonical token node before this observation was
	// grounded, promote that same node with its first numeric representation.
	// This preserves one neural unit across symbolic ingestion and later
	// perception instead of creating token-node/numeric-node duplicates.
	k.mu.Lock()
	k.Registry.mu.Lock()
	if id, exists := k.Registry.byToken[canonical]; exists {
		if node := k.Registry.byID[id]; node != nil && len(node.Representation) == 0 {
			node.Representation = append([]float64(nil), vector.Values...)
			node.Activation = 1
			if now.IsZero() {
				node.LastActivation = time.Now().UTC()
			} else {
				node.LastActivation = now.UTC()
			}
			k.Registry.mu.Unlock()
			k.mu.Unlock()
			return GroundedRepresentation{
				NodeID: node.ID, Similarity: 1, Status: GroundingExisting,
				Source: source, Modality: modality, Token: canonical,
				Timestamp: now.UTC(),
			}, nil
		}
	}
	k.Registry.mu.Unlock()
	k.mu.Unlock()

	nodeID, similarity, created, err := k.ProjectVectorAt(vector, threshold, now)
	if err != nil {
		return GroundedRepresentation{}, err
	}
	if created {
		// A newly grounded numeric unit can also carry the observed token as a
		// compatibility anchor. The numeric representation remains canonical;
		// the token only lets later textual learning/recovery resolve the same
		// neural unit instead of creating a duplicate.
		k.mu.Lock()
		k.Registry.mu.Lock()
		if node := k.Registry.byID[nodeID]; node != nil && node.Token == "" {
			node.Token = canonical
			if _, exists := k.Registry.byToken[canonical]; !exists {
				k.Registry.byToken[canonical] = nodeID
			}
		}
		k.Registry.mu.Unlock()
		k.mu.Unlock()
	}
	status := GroundingExisting
	if created {
		status = GroundingCandidate
	}
	return GroundedRepresentation{
		NodeID: nodeID, Similarity: similarity, Status: status,
		Source: source, Modality: modality, Token: canonical,
		Timestamp: now.UTC(),
	}, nil
}
