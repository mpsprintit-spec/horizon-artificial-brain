package knowledge

import (
	"bytes"
	"encoding/json"
	"errors"
	"math"
)

// ValidateCompositionState checks the canonical coordinate topology without
// constructing another mutable Brain or touching the restore target.
func ValidateCompositionState(data []byte, enabled bool) error {
	if !enabled && !bytes.Contains(data, []byte(NeuralCompositionV1)) && bytes.IndexByte(data, '\\') < 0 {
		return nil
	}
	var raw struct {
		Units       []canonicalNeuralUnit  `json:"neural_units"`
		Populations []ProjectionPopulation `json:"populations"`
		Synapses    []canonicalSynapse     `json:"synapses"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	known := map[NodeID]bool{}
	for _, u := range raw.Units {
		known[u.ID] = true
	}
	var banks [compositionBanks][]NodeID
	members := map[NodeID]bool{}
	for _, p := range raw.Populations {
		if p.Scope == nil || *p.Scope != compositionScope {
			continue
		}
		if !enabled {
			return errors.New("composition populations lack an authenticated serving contract")
		}
		if len(p.Prototype) != 1 || len(p.Units) != 4 {
			return errors.New("invalid composition coordinate shape")
		}
		x := p.Prototype[0] * 64
		if math.IsNaN(x) || math.IsInf(x, 0) || x < 0 || x >= compositionBanks || math.Trunc(x) != x || len(banks[int(x)]) > 0 {
			return errors.New("invalid composition coordinate index")
		}
		for _, u := range p.Units {
			if !known[u.NodeID] || members[u.NodeID] {
				return errors.New("invalid composition population member")
			}
			members[u.NodeID] = true
			banks[int(x)] = append(banks[int(x)], u.NodeID)
		}
	}
	if !enabled {
		return nil
	}
	for _, p := range banks {
		if len(p) != 4 {
			return errors.New("incomplete composition coordinate bank")
		}
	}
	type key struct {
		from, to   NodeID
		inhibitory bool
	}
	edges := map[key]bool{}
	for _, s := range raw.Synapses {
		if !members[s.TargetID] {
			continue
		}
		k := key{s.SourceID, s.TargetID, s.Inhibitory}
		if edges[k] || s.Kind != "" || !known[s.SourceID] || math.IsNaN(s.Dynamic.Weight) || math.IsInf(s.Dynamic.Weight, 0) || s.Dynamic.Weight < 0 || s.Dynamic.Weight > 1 || s.Dynamic.Confidence < 0 || s.Dynamic.Confidence > 1 {
			return errors.New("invalid composition connection")
		}
		edges[k] = true
	}
	check := func(from, to []NodeID) bool {
		for _, a := range from {
			for _, b := range to {
				if !edges[key{a, b, false}] || !edges[key{a, b, true}] {
					return false
				}
			}
		}
		return true
	}
	for out := 0; out < compositionOutputs; out++ {
		for h := 0; h < compositionHidden; h++ {
			if !check(banks[h], banks[12+out]) {
				return errors.New("incomplete composition readout")
			}
		}
		for pos := 0; pos < MaxCompositionWords; pos++ {
			if !check(banks[18+pos], banks[12+out]) {
				return errors.New("incomplete composition position connections")
			}
		}
		if !check(banks[24], banks[12+out]) {
			return errors.New("incomplete composition bias")
		}
	}
	if !check(banks[25], banks[27]) || !check(banks[26], banks[27]) {
		return errors.New("incomplete composition controller")
	}
	return nil
}
