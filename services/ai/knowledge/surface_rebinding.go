package knowledge

import (
	"errors"
	"reflect"
	"strings"
	"time"
)

// LegacyWrittenSurface selects exactly the checkpointed adapter association
// requested by a migration. It does not activate, merge or create a neuron.
func (k *KnowledgeBase) LegacyWrittenSurface(surface, modality string) (SurfaceAnnotation, error) {
	if k == nil || CanonicalSurfaceModality(modality) != "language" {
		return SurfaceAnnotation{}, errors.New("rebinding requires a written surface")
	}
	var selected *SurfaceAnnotation
	for _, a := range k.SurfaceAnnotationsSnapshot() {
		if a.Surface != surface || a.Modality != modality || a.Encoding != LegacySurfaceEncoding {
			continue
		}
		if selected != nil && !reflect.DeepEqual(*selected, a) {
			return SurfaceAnnotation{}, errors.New("ambiguous historical surface association")
		}
		copy := a
		selected = &copy
	}
	if selected == nil || len(selected.Population) == 0 || selected.NodeID != selected.Population[0] {
		return SurfaceAnnotation{}, errors.New("historical surface population is unavailable")
	}
	for _, id := range selected.Population {
		if id == 0 || k.Registry.GetByID(id) == nil {
			return SurfaceAnnotation{}, errors.New("historical surface references an unavailable neuron")
		}
	}
	return *selected, nil
}

func (k *KnowledgeBase) ValidateWrittenSurfaceSource(source SurfaceAnnotation) error {
	actual, err := k.LegacyWrittenSurface(source.Surface, source.Modality)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(source, actual) {
		return errors.New("historical surface association changed")
	}
	return nil
}

// RebindWrittenSurface changes only adapter metadata. The original annotation,
// neural populations, representations, synapses and learned traces survive.
func (k *KnowledgeBase) RebindWrittenSurface(source SurfaceAnnotation, referenceHash string, at time.Time) error {
	if err := k.ValidateWrittenSurfaceSource(source); err != nil {
		return err
	}
	if canonicalToken(source.Surface) == "" || !strings.HasPrefix(referenceHash, "sha256:") || at.IsZero() {
		return errors.New("rebinding requires a reference and event time")
	}
	a := SurfaceAnnotation{Surface: canonicalToken(source.Surface), Modality: "language", NodeID: source.NodeID, Population: append([]NodeID(nil), source.Population...), LastSeen: at.UTC(), Encoding: WrittenTextEncoding, ReboundFrom: source.Modality + ":" + source.Surface, ReferenceHash: referenceHash}
	k.annotationMu.Lock()
	defer k.annotationMu.Unlock()
	for i, old := range k.SurfaceAnnotations {
		if old.Surface == a.Surface && old.Modality == a.Modality && old.Encoding == a.Encoding {
			k.SurfaceAnnotations[i] = a
			return nil
		}
	}
	k.SurfaceAnnotations = append(k.SurfaceAnnotations, a)
	return nil
}
