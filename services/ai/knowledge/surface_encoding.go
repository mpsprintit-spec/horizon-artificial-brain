package knowledge

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

// SurfaceEncoding versions the adapter's numerical encoding, not intelligence
// or neural identity. Empty is the historical modality-specific contract.
type SurfaceEncoding string

const (
	LegacySurfaceEncoding SurfaceEncoding = ""
	WrittenTextEncoding   SurfaceEncoding = "written-text-v1"
	NeuralWrittenEncoding SurfaceEncoding = "written-neural-v2"
)

func (e SurfaceEncoding) Validate() error {
	if e != LegacySurfaceEncoding && e != WrittenTextEncoding && e != NeuralWrittenEncoding {
		return fmt.Errorf("unsupported surface encoding %q", e)
	}
	return nil
}

// CanonicalSurfaceModality unifies names for written input only. Audio, image,
// IMU and other sensory coordinate systems keep their own encoder namespace.
// "language" preserves the vectors already used by learned word sequences.
func CanonicalSurfaceModality(modality string) string {
	switch strings.ToLower(strings.TrimSpace(modality)) {
	case "text", "language", "web-search":
		return "language"
	default:
		return modality
	}
}

func observationVectorWithEncoding(value, modality string, encoding SurfaceEncoding) NeuralVector {
	if (encoding == WrittenTextEncoding || encoding == NeuralWrittenEncoding) && CanonicalSurfaceModality(modality) == "language" {
		return observationVector(canonicalToken(value), "language")
	}
	return observationVector(value, modality)
}

// StoreSurfaceAt is the sequence-learning adapter. It preserves the existing
// single anchor learning rule while recording a reversible surface label.
// A versioned binding may point to an existing distributed population.
func (k *KnowledgeBase) StoreSurfaceAt(token string, now time.Time, encoding SurfaceEncoding) *ConceptNode {
	if encoding == LegacySurfaceEncoding {
		return k.StoreAt(token, now)
	}
	if k == nil || k.Registry == nil || encoding.Validate() != nil {
		return nil
	}
	canonical := canonicalToken(token)
	if canonical == "" {
		return nil
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	vector := observationVectorWithEncoding(canonical, "language", encoding)
	if encoding == NeuralWrittenEncoding {
		population, _, err := k.projectWrittenNumeric(vector, .999999, 1, now, true)
		if err != nil || len(population.Units) == 0 {
			return nil
		}
		ids := writtenPopulationIDs(population)
		k.recordEncodedSurfaceAnnotation(canonical, "language", ids[0], ids, now, encoding)
		return k.Registry.GetByID(ids[0])
	}
	if population, ok := k.populationForWrittenSurface(canonical, vector, now); ok {
		ids := make([]NodeID, len(population.Units))
		for i, unit := range population.Units {
			ids[i] = unit.NodeID
		}
		k.recordEncodedSurfaceAnnotation(canonical, "language", ids[0], ids, now, encoding)
		node := k.Registry.GetByID(population.Units[0].NodeID)
		node.Frequency++ // one occurrence per experience, as in StoreAt
		return node
	}
	node := k.StoreAt(canonical, now)
	if node != nil {
		k.recordEncodedSurfaceAnnotation(canonical, "language", node.ID, []NodeID{node.ID}, now, encoding)
	}
	return node
}

func (k *KnowledgeBase) recordEncodedSurfaceAnnotation(surface, modality string, nodeID NodeID, population []NodeID, at time.Time, encoding SurfaceEncoding) {
	if (encoding != WrittenTextEncoding && encoding != NeuralWrittenEncoding) || CanonicalSurfaceModality(modality) != "language" {
		k.recordSurfaceAnnotation(surface, modality, nodeID, population, at)
		return
	}
	surface = canonicalToken(surface)
	k.annotationMu.Lock()
	defer k.annotationMu.Unlock()
	for i := range k.SurfaceAnnotations {
		a := &k.SurfaceAnnotations[i]
		if a.Surface == surface && a.Modality == "language" && a.Encoding == encoding {
			if a.NodeID != nodeID || !slices.Equal(a.Population, population) {
				a.ReboundFrom, a.ReferenceHash = "", ""
			}
			a.NodeID = nodeID
			a.Population = append(a.Population[:0], population...)
			a.LastSeen = at.UTC()
			return
		}
	}
	k.SurfaceAnnotations = append(k.SurfaceAnnotations, SurfaceAnnotation{Surface: surface, Modality: "language", NodeID: nodeID, Population: append([]NodeID(nil), population...), LastSeen: at.UTC(), Encoding: encoding})
}

func (k *KnowledgeBase) populationForWrittenSurface(surface string, vector NeuralVector, at time.Time) (ProjectionPopulation, bool) {
	// Explicit versioned bindings win over historical annotations, including
	// a reviewed rebinding of a legacy text population. Historical language
	// annotations remain usable without rewriting any prior checkpoint state.
	k.annotationMu.RLock()
	var ids []NodeID
	for _, encoding := range []SurfaceEncoding{WrittenTextEncoding, LegacySurfaceEncoding} {
		for _, a := range k.SurfaceAnnotations {
			if canonicalToken(a.Surface) != surface || a.Modality != "language" || a.Encoding != encoding {
				continue
			}
			ids = append([]NodeID(nil), a.Population...)
			break
		}
		if len(ids) > 0 {
			break
		}
	}
	k.annotationMu.RUnlock()
	return k.activateSurfacePopulation(ids, vector, at)
}
