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

	nodeID, similarity, created, err := k.ProjectVectorAt(vector, threshold, now)
	if err != nil {
		return GroundedRepresentation{}, err
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
