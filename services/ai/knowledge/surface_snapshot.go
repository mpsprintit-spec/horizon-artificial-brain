package knowledge

import (
	"errors"
	"fmt"
)

// SurfaceAnnotationsSnapshot exposes a detached copy of adapter provenance.
// These language/sensor surfaces are NOT intrinsic neural identity and must
// never be embedded in CanonicalJSON or the persisted Brain substrate.
func (k *KnowledgeBase) SurfaceAnnotationsSnapshot() []SurfaceAnnotation {
	if k == nil { return nil }
	k.annotationMu.RLock()
	defer k.annotationMu.RUnlock()
	out := make([]SurfaceAnnotation, len(k.SurfaceAnnotations))
	for i, annotation := range k.SurfaceAnnotations {
		out[i] = annotation
		out[i].Population = append([]NodeID(nil), annotation.Population...)
	}
	return out
}

// RestoreSurfaceAnnotations reconnects checkpointed adapter metadata to the
// same canonical neural substrate. Every referenced unit must actually exist.
// The supplied slices are copied so later caller mutations cannot alter state.
func (k *KnowledgeBase) RestoreSurfaceAnnotations(annotations []SurfaceAnnotation) error {
	if k == nil || k.Registry == nil { return errors.New("brain is not initialized") }
	copyAnnotations := make([]SurfaceAnnotation, len(annotations))
	for i, annotation := range annotations {
		if err := annotation.Encoding.Validate(); err != nil { return err }
		if annotation.NodeID != 0 && k.Registry.GetByID(annotation.NodeID) == nil {
			return fmt.Errorf("surface annotation %d has unknown node %d", i, annotation.NodeID)
		}
		for _, id := range annotation.Population {
			if id == 0 || k.Registry.GetByID(id) == nil {
				return fmt.Errorf("surface annotation %d has unknown population node %d", i, id)
			}
		}
		copyAnnotations[i] = annotation
		copyAnnotations[i].Population = append([]NodeID(nil), annotation.Population...)
	}
	k.annotationMu.Lock()
	defer k.annotationMu.Unlock()
	k.SurfaceAnnotations = copyAnnotations
	return nil
}
