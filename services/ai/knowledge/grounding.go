package knowledge

import (
	"errors"
	"hash/fnv"
	"strings"
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
	Similarity float64
	Status     GroundingStatus
	Source     string
	Modality   string
	Token      string
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
	nodeID, similarity, created, err := k.ProjectVector(vector, threshold)
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
