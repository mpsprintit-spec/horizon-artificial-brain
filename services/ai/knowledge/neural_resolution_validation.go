package knowledge

import (
	"encoding/json"
	"errors"
	"math"
	"slices"
)

// Reject malformed, detached or unauthenticated N topology before restore.
func validateNResolutionState(data []byte, enabled bool) error {
	var raw struct {
		State       *NResolutionState      `json:"n_resolution"`
		Units       []canonicalNeuralUnit  `json:"neural_units"`
		Populations []ProjectionPopulation `json:"populations"`
		Synapses    []canonicalSynapse     `json:"synapses"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	s := raw.State
	known := map[NodeID]bool{}
	for _, u := range raw.Units {
		known[u.ID] = true
	}
	nUnits := map[NodeID][]float64{}
	var precision, bias, errorBank []NodeID
	queries := [][]NodeID{}
	for _, p := range raw.Populations {
		if p.Scope == nil {
			continue
		}
		if p.Scope.Field == NeuralCompositionResolutionV5 {
			if !enabled || s == nil || p.Scope.Modality != "internal-numeric" || len(p.Units) != 1 || len(p.Prototype) != 2 || nUnits[p.Units[0].NodeID] != nil {
				return errors.New("invalid or unauthenticated N population")
			}
			nUnits[p.Units[0].NodeID] = p.Prototype
		}
		if *p.Scope == compositionScope && len(p.Prototype) == 1 {
			switch int(math.Round(p.Prototype[0] * 64)) {
			case 24:
				bias = writtenPopulationIDs(p)
			case 25:
				precision = writtenPopulationIDs(p)
			case 26:
				errorBank = writtenPopulationIDs(p)
			}
		}
		if *p.Scope == writtenNumericScope {
			queries = append(queries, writtenPopulationIDs(p))
		}
	}
	if s == nil {
		return nil
	}
	if !enabled || len(s.Sources) != nDimensions || len(s.Features) > maxNFeatures || len(s.Witnesses) > maxNWitnesses {
		return errors.New("invalid N contract or resource limits")
	}
	type edge struct {
		a, b NodeID
		inh  bool
	}
	edges := map[edge]canonicalSynapse{}
	for _, v := range raw.Synapses {
		if nUnits[v.SourceID] != nil || nUnits[v.TargetID] != nil {
			if v.Kind != "" || !known[v.SourceID] || !known[v.TargetID] || math.IsNaN(v.Dynamic.Weight) || math.IsInf(v.Dynamic.Weight, 0) || v.Dynamic.Weight < 0 || v.Dynamic.Weight > 1 {
				return errors.New("invalid N synapse")
			}
		}
		key := edge{v.SourceID, v.TargetID, v.Inhibitory}
		if _, exists := edges[key]; exists && (nUnits[v.SourceID] != nil || nUnits[v.TargetID] != nil) {
			return errors.New("duplicate N parameter edge")
		}
		edges[key] = v
	}
	param := func(a, b []NodeID) (float64, bool) {
		v := 0.
		for _, x := range a {
			for _, y := range b {
				p, ok := edges[edge{x, y, false}]
				n, nok := edges[edge{x, y, true}]
				if !ok || !nok {
					return 0, false
				}
				v += compositionScale * (p.Dynamic.Weight - n.Dynamic.Weight)
			}
		}
		if len(a)*len(b) == 0 {
			return 0, false
		}
		return v / float64(len(a)*len(b)), true
	}
	used := map[NodeID]bool{}
	features := map[NodeID]bool{}
	for j, id := range s.Sources {
		if id == 0 {
			continue
		}
		p := nUnits[id]
		if !known[id] || used[id] || len(p) != 2 || p[0] != .125 || p[1] != float64(j)/512 {
			return errors.New("invalid N source coordinate")
		}
		used[id] = true
	}
	for i, f := range s.Features {
		p := nUnits[f.Node]
		if !known[f.Node] || used[f.Node] || len(p) != 2 || p[0] != .25 || p[1] != float64(i)/512 {
			return errors.New("invalid N feature topology")
		}
		used[f.Node] = true
		d, ok := param(precision, []NodeID{f.Node})
		if !ok || math.IsNaN(d) || d < 1e-9 || d > 7.99 {
			return errors.New("invalid N learned resolution")
		}
		center, centerOK := param(bias, []NodeID{f.Node})
		if !centerOK || math.IsNaN(center) || math.IsInf(center, 0) || math.Abs(center) > 7.99 {
			return errors.New("invalid N learned center")
		}
		if f.Kind == 1 {
			if len(f.Parents) != 0 {
				return errors.New("projection N has parents")
			}
			norm := 0.
			for _, id := range s.Sources {
				if id != 0 {
					v, ok := param([]NodeID{id}, []NodeID{f.Node})
					if ok {
						norm += v * v
					}
				}
			}
			if math.Abs(norm-1) > 1e-7 {
				return errors.New("invalid N learned direction")
			}
		} else if f.Kind == 2 {
			if len(f.Parents) != 2 {
				return errors.New("invalid N relation arity")
			}
			for _, id := range f.Parents {
				v, ok := param([]NodeID{id}, []NodeID{f.Node})
				if !features[id] || !ok || v != 1 {
					return errors.New("N relation is cyclic or lacks learned parents")
				}
			}
		} else {
			return errors.New("unknown N circuit")
		}
		features[f.Node] = true
	}
	if len(used) != len(nUnits) {
		return errors.New("orphan N population")
	}
	headKnown := func(q []NodeID) bool {
		return slices.ContainsFunc(queries, func(p []NodeID) bool { return slices.Equal(q, p) })
	}
	headSeen := map[NodeID]bool{}
	for _, h := range s.Heads {
		if len(h.Query) != 4 || !headKnown(h.Query) || headSeen[h.Query[0]] {
			return errors.New("invalid N query topology")
		}
		headSeen[h.Query[0]] = true
		if _, ok := param(bias, h.Query); !ok {
			return errors.New("missing N bias")
		}
		e, ok := param(errorBank, h.Query)
		if !ok || e < 0 || e > 7.99 {
			return errors.New("invalid N empirical error")
		}
		if len(h.Features) > len(s.Features) {
			return errors.New("invalid N head extent")
		}
		for i, id := range h.Features {
			if s.Features[i].Node != id {
				return errors.New("invalid N readout order")
			}
			if _, ok := param([]NodeID{id}, h.Query); !ok {
				return errors.New("missing N readout weight")
			}
		}
	}
	for _, w := range s.Witnesses {
		if len(w.X) != nDimensions || len(w.Query) != 4 || !headSeen[w.Query[0]] || !headKnown(w.Query) || w.EpisodeID == "" || math.IsNaN(w.Outcome) || math.IsInf(w.Outcome, 0) || math.Abs(w.Outcome) > 1 {
			return errors.New("invalid N retention witness")
		}
		for _, v := range w.X {
			if math.IsNaN(v) || math.IsInf(v, 0) || math.Abs(v) > 1 {
				return errors.New("invalid N witness measurement")
			}
		}
	}
	return nil
}
