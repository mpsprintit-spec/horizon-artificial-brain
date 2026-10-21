package knowledge

import (
	"errors"
	"math"
	"reflect"
	"slices"
	"sort"
	"time"
)

// This scope names codec coordinates, never a word, concept or semantic role.
var writtenNumericScope = ProjectionScope{Modality: "language", Field: "written-neural-v2"}

func writtenPopulationIDs(p ProjectionPopulation) []NodeID {
	ids := make([]NodeID, len(p.Units))
	for i, unit := range p.Units {
		ids[i] = unit.NodeID
	}
	return ids
}

// Only numeric prototypes/units are read here. Adapter annotations are neither
// an identity source nor a fallback. The runtime owns this mutation boundary.
func (k *KnowledgeBase) projectWrittenNumeric(vector NeuralVector, threshold float64, size int, at time.Time, learning bool) (ProjectionPopulation, bool, error) {
	if k == nil || k.Registry == nil {
		return ProjectionPopulation{}, false, ErrNilBrain
	}
	if vector.Empty() {
		return ProjectionPopulation{}, false, ErrEmptyNeuralVector
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	at = at.UTC()
	k.Lock()
	defer k.Unlock()
	k.projectionMu.RLock()
	var selected ProjectionPopulation
	found := false
	for _, p := range k.ProjectionPopulations {
		if p.Scope != nil && *p.Scope == writtenNumericScope && slices.Equal(p.Prototype, vector.Values) {
			selected, found = p, true
			break
		}
	}
	k.projectionMu.RUnlock()
	if found {
		if len(selected.Units) == 0 {
			return ProjectionPopulation{}, false, errors.New("empty numeric written population")
		}
		for _, unit := range selected.Units {
			if k.Registry.GetByID(unit.NodeID) == nil {
				return ProjectionPopulation{}, false, errors.New("numeric written population has an unavailable unit")
			}
		}
		if learning {
			k.Registry.GetByID(selected.Units[0].NodeID).Frequency++
		}
		return k.activateProjectionPopulationAt(selected, vector, at), true, nil
	}
	candidates := k.matchRepresentationPopulation(vector, threshold)
	p := ProjectionPopulation{Scope: cloneProjectionScope(&writtenNumericScope), Prototype: slices.Clone(vector.Values)}
	if len(candidates) > 0 {
		p.Units = append(p.Units, PopulationUnit{NodeID: candidates[0].node.ID})
		if learning {
			candidates[0].node.Frequency++
		}
	}
	for slot := len(p.Units); slot < size; slot++ {
		node := k.createRepresentationPrototypeAt(projectionPrototype(vector, slot, size), at)
		p.Units = append(p.Units, PopulationUnit{NodeID: node.ID})
	}
	k.registerProjectionPopulation(p)
	return k.activateProjectionPopulationAt(p, vector, at), false, nil
}

// PlanWrittenNumericProjection is an offline compatibility boundary. It reads
// old labels ONCE to preserve their existing routes as numeric receptive fields.
// It creates no units and makes no claim that hashing spelling learns meaning.
// Conflicting same-priority bindings are rejected rather than silently merged.
func (k *KnowledgeBase) PlanWrittenNumericProjection() ([]ProjectionPopulation, error) {
	if k == nil || k.Registry == nil {
		return nil, ErrNilBrain
	}
	k.projectionMu.RLock()
	for _, p := range k.ProjectionPopulations {
		if p.Scope != nil && *p.Scope == writtenNumericScope {
			k.projectionMu.RUnlock()
			return nil, errors.New("numeric written projections already exist; recover their complete checkpoint")
		}
	}
	k.projectionMu.RUnlock()
	annotations := k.SurfaceAnnotationsSnapshot()
	selected := make(map[string]SurfaceAnnotation)
	for _, encoding := range []SurfaceEncoding{WrittenTextEncoding, LegacySurfaceEncoding} {
		for _, a := range annotations {
			if a.Encoding != encoding || a.Modality != "language" || len(a.Population) == 0 {
				continue
			}
			surface := canonicalToken(a.Surface)
			if surface == "" {
				return nil, errors.New("empty historical written surface")
			}
			if old, exists := selected[surface]; exists {
				if old.Encoding == encoding && !slices.Equal(old.Population, a.Population) {
					return nil, errors.New("ambiguous historical written population")
				}
				continue
			}
			if a.NodeID != a.Population[0] {
				return nil, errors.New("historical written anchor disagrees")
			}
			seen := make(map[NodeID]bool)
			for _, id := range a.Population {
				if id == 0 || seen[id] || k.Registry.GetByID(id) == nil {
					return nil, errors.New("invalid historical written population")
				}
				seen[id] = true
			}
			selected[surface] = a
		}
	}
	keys := make([]string, 0, len(selected))
	for key := range selected {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]ProjectionPopulation, 0, len(keys))
	seenPrototypes := make(map[[16]uint64]bool, len(keys))
	for _, key := range keys {
		p := ProjectionPopulation{Scope: cloneProjectionScope(&writtenNumericScope),
			Prototype: observationVectorWithEncoding(key, "language", NeuralWrittenEncoding).Values}
		for _, id := range selected[key].Population {
			p.Units = append(p.Units, PopulationUnit{NodeID: id})
		}
		// A codec collision must not pick one existing population arbitrarily.
		var key [16]uint64
		for i, value := range p.Prototype {
			key[i] = math.Float64bits(value)
		}
		if seenPrototypes[key] {
			return nil, errors.New("colliding numeric written prototypes")
		}
		seenPrototypes[key] = true
		out = append(out, p)
	}
	return out, nil
}

// ApplyWrittenNumericProjection validates the exact detached plan before adding
// numeric routing traces to the SAME canonical substrate. Old data is retained.
func (k *KnowledgeBase) ApplyWrittenNumericProjection(plan []ProjectionPopulation) error {
	expected, err := k.PlanWrittenNumericProjection()
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(plan, expected) {
		return errors.New("numeric written migration plan changed")
	}
	k.Lock()
	defer k.Unlock()
	k.projectionMu.RLock()
	for _, p := range k.ProjectionPopulations {
		if p.Scope != nil && *p.Scope == writtenNumericScope {
			k.projectionMu.RUnlock()
			return errors.New("numeric written projections already exist; migration cannot overwrite them")
		}
	}
	k.projectionMu.RUnlock()
	for _, p := range plan {
		k.registerProjectionPopulation(p)
	}
	return nil
}
