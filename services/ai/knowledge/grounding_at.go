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

	// Event-time grounding preserves modality-specific surface populations.
	// Cross-modal meaning is learned later from co-occurring experience rather
	// than inferred from lexical equality.
	_, hadPopulation := k.findExactProjectionPopulation(vector)
	population, err := k.ProjectVectorPopulationAt(vector, threshold, defaultProjectionPopulation, now)
	if err != nil || len(population.Units) == 0 {
		if err != nil {
			return GroundedRepresentation{}, err
		}
		return GroundedRepresentation{}, ErrEmptyNeuralVector
	}
	populationIDs := make([]NodeID, 0, len(population.Units))
	for _, unit := range population.Units {
		populationIDs = append(populationIDs, unit.NodeID)
	}
	status := GroundingExisting
	if !hadPopulation {
		status = GroundingCandidate
	}
	k.recordSurfaceAnnotation(canonical, modality, populationIDs[0], populationIDs, now.UTC())
	return GroundedRepresentation{
		NodeID: populationIDs[0], Population: populationIDs,
		Similarity: population.Units[0].Activation, Status: status,
		Source: source, Modality: modality, Token: canonical,
		Timestamp: now.UTC(),
	}, nil
}
