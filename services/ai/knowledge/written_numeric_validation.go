package knowledge

import (
	"bytes"
	"encoding/json"
	"errors"
	"math"
)

// HasNeuralWrittenProjection prevents an unauthenticated Brain-only graph from
// being silently attached to the compatibility runtime's serving contract.
func (k *KnowledgeBase) HasNeuralWrittenProjection() bool {
	if k == nil {
		return false
	}
	k.projectionMu.RLock()
	defer k.projectionMu.RUnlock()
	for _, p := range k.ProjectionPopulations {
		if p.Scope != nil && *p.Scope == writtenNumericScope {
			return true
		}
	}
	return false
}

// ValidateWrittenNumericState rejects malformed numerical routes before a
// checkpoint can mutate its target. Ordinary legacy files take the byte-scan
// fast path; the new mode may have no routes yet immediately after activation.
func ValidateWrittenNumericState(data []byte, enabled bool) error {
	// An equivalent field/value can use JSON Unicode escapes. Only skip the
	// semantic decode when neither the literal marker nor an escape occurs.
	if !bytes.Contains(data, []byte(writtenNumericScope.Field)) && bytes.IndexByte(data, '\\') < 0 {
		return nil
	}
	var state struct {
		Units []struct {
			ID NodeID `json:"id"`
		} `json:"neural_units"`
		Populations []ProjectionPopulation `json:"populations"`
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}
	known := make(map[NodeID]bool, len(state.Units))
	for _, unit := range state.Units {
		known[unit.ID] = true
	}
	prototypes := make(map[[16]uint64]bool)
	for _, p := range state.Populations {
		if p.Scope == nil || *p.Scope != writtenNumericScope {
			continue
		}
		if !enabled {
			return errors.New("numeric written projections require the authenticated serving contract")
		}
		if len(p.Prototype) != 16 || len(p.Units) == 0 {
			return errors.New("invalid numeric written receptive field")
		}
		var key [16]uint64
		for i, value := range p.Prototype {
			if math.IsNaN(value) || math.IsInf(value, 0) || value < -1 || value > 1 {
				return errors.New("invalid numeric written prototype")
			}
			key[i] = math.Float64bits(value)
		}
		if prototypes[key] {
			return errors.New("ambiguous numeric written receptive field")
		}
		prototypes[key] = true
		seen := make(map[NodeID]bool, len(p.Units))
		for _, unit := range p.Units {
			if unit.NodeID == 0 || !known[unit.NodeID] || seen[unit.NodeID] {
				return errors.New("invalid numeric written population member")
			}
			seen[unit.NodeID] = true
		}
	}
	return nil
}
