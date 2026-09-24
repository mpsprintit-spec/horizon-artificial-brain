package knowledge

import (
	"errors"
	"hash/fnv"
	"strings"
	"time"
)

// GroundingStatus describes how an observation entered the shared brain.
// Candidate means the representation is new and must not be treated as
// permanently trusted knowledge.
type GroundingStatus string

const (
	GroundingExisting GroundingStatus = "existing"
	GroundingCandidate GroundingStatus = "candidate"
)

// GroundedRepresentation is provenance for one observed input. The neural
// substrate stores the numeric representation; this record describes how it
// was obtained without turning the source token into semantic knowledge.
type GroundedRepresentation struct {
	NodeID     NodeID
	Population []NodeID
	Similarity float64
	Status     GroundingStatus
	Source     string
	Modality   string
	Token      string
	Timestamp  time.Time
}

// GroundObservation converts an unknown textual observation into a numeric
// candidate and projects it into the same brain substrate used by non-language
// experience. Existing compatible representations are reused; otherwise a new
// tentative neural unit is created.
func (k *KnowledgeBase) GroundObservation(token, source, modality string, threshold float64) (GroundedRepresentation, error) {
	if k == nil {
		return GroundedRepresentation{}, errors.New("brain is nil")
	}
	canonical := strings.TrimSpace(token)
	if canonical == "" {
		return GroundedRepresentation{}, errors.New("observation token is empty")
	}

	vector := observationVector(canonical, modality)
	_, hadPopulation := k.findExactProjectionPopulation(vector)
	population, err := k.ProjectVectorPopulation(vector, threshold, defaultProjectionPopulation)
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
	anchor := populationIDs[0]
	status := GroundingExisting
	if !hadPopulation {
		status = GroundingCandidate
	}
	return GroundedRepresentation{
		NodeID: anchor, Population: populationIDs, Similarity: population.Units[0].Activation,
		Status: status, Source: source, Modality: modality,
		Token: canonical, Timestamp: time.Now().UTC(),
	}, nil
}

// observationVector is a deterministic numeric encoder used only to bridge
// raw observations into the substrate. It does not assign semantic labels.
// A stable hash makes repeated identical observations reproducible while the
// brain remains responsible for deciding whether the resulting pattern matches
// an existing representation.
func observationVector(token, modality string) NeuralVector {
	const dimensions = 16
	values := make([]float64, dimensions)
	h := fnv.New64a()
	_, _ = h.Write([]byte(strings.ToLower(strings.TrimSpace(modality))))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(strings.ToLower(token)))
	seed := h.Sum64()
	for i := range values {
		seed ^= seed << 13
		seed ^= seed >> 7
		seed ^= seed << 17
		values[i] = (float64(seed%2001)/1000.0) - 1.0
	}
	return NewNeuralVector(values)
}


// EncodeObservation is the boundary encoder for language and sensor inputs.
// It produces a modality-neutral numeric representation; the lexical or
// sensor surface form is not retained by the neural substrate.
func EncodeObservation(value, modality string) NeuralVector {
	return observationVector(value, modality)
}
