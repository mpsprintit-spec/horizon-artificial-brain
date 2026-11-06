package knowledge

import (
	"bytes"
	"encoding/json"
	"errors"
	"math"
)

func ValidateNativeControlState(data []byte, enabled bool) error {
	if !enabled && !bytes.Contains(data, []byte("native_control")) {
		return nil
	}
	var raw struct {
		Schema    int                   `json:"schema_version"`
		State     *NativeControlState   `json:"native_control"`
		Units     []canonicalNeuralUnit `json:"neural_units"`
		Edges     []canonicalSynapse    `json:"synapses"`
		Cognitive json.RawMessage       `json:"cognitive_model"`
	}
	if e := json.Unmarshal(data, &raw); e != nil {
		return e
	}
	s := raw.State
	if s == nil {
		if enabled || raw.Schema == 7 {
			return errors.New("missing native control")
		}
		return nil
	}
	if !enabled || raw.Schema != 7 || len(raw.Cognitive) > 0 && string(raw.Cognitive) != "null" {
		return errors.New("native state requires canonical7/checkpoint20 and isolated native controller")
	}
	if e := s.Config.Validate(); e != nil {
		return e
	}
	known := map[NodeID]bool{}
	for _, n := range raw.Units {
		if known[n.ID] {
			return errors.New("duplicate native neuron")
		}
		known[n.ID] = true
	}
	owned := map[NodeID]bool{}
	inputs := map[NodeID]bool{}
	for _, id := range s.Config.Sources {
		if !known[id] {
			return errors.New("missing native source")
		}
		inputs[id] = true
	}
	if !known[s.Bias] || inputs[s.Bias] {
		return errors.New("invalid native bias")
	}
	inputs[s.Bias] = true
	owned[s.Bias] = true
	if len(s.Outputs) != len(s.Config.Operations) {
		return errors.New("native operation readout mismatch")
	}
	targets := map[NodeID]bool{}
	for _, group := range s.Outputs {
		if len(group) != 4 {
			return errors.New("native effector population size")
		}
		for _, id := range group {
			if !known[id] || owned[id] || inputs[id] {
				return errors.New("invalid native effector")
			}
			owned[id] = true
			targets[id] = true
		}
	}
	type edge struct{ from, to NodeID }
	edges := map[edge]canonicalSynapse{}
	for _, e := range raw.Edges {
		if targets[e.SourceID] {
			return errors.New("native effectors cannot feed observations")
		}
		if !targets[e.TargetID] {
			if e.SourceID == s.Bias {
				return errors.New("native bias foreign edge")
			}
			continue
		}
		key := edge{e.SourceID, e.TargetID}
		if _, ok := edges[key]; ok {
			return errors.New("duplicate native parameter")
		}
		if !inputs[e.SourceID] || e.Kind != "" || !nativeFinite(e.Weight) || e.Weight < 0 || e.Weight > .9375 || e.Dynamic.Weight != e.Weight || e.Confidence != 1 || e.Dynamic.Confidence != 1 {
			return errors.New("invalid native canonical parameter")
		}
		edges[key] = e
	}
	if len(edges) != len(inputs)*len(targets) {
		return errors.New("incomplete native topology")
	}
	if ep := s.Episode; ep != nil {
		if ep.ID == "" || ep.Started.IsZero() || len(ep.Decisions) > s.Config.MaxSteps || ep.Truncated && (!ep.Stopped || len(ep.Decisions) != s.Config.MaxSteps) {
			return errors.New("invalid native episode")
		}
		previous := ep.Started
		for i, d := range ep.Decisions {
			if d.At.Before(previous) || d.Action < 0 || d.Action >= len(s.Outputs) || len(d.Values) != len(s.Outputs) || len(d.Probabilities) != len(s.Outputs) || d.Cost != s.Config.Operations[d.Action].Cost || d.Before != NativeStateDigest(d.State) || len(d.After) != 64 {
				return errors.New("invalid native decision")
			}
			if i > 0 && ep.Decisions[i-1].After != d.Before {
				return errors.New("broken native state continuity")
			}
			if s.Config.Operations[d.Action].Kind == "stop" && (i != len(ep.Decisions)-1 || !ep.Stopped) {
				return errors.New("decision after stop")
			}
			x := map[NodeID]float64{s.Bias: 1}
			for id, v := range d.State {
				if !known[id] || owned[id] || !nativeFinite(v) || v < 0 || v > 2 {
					return errors.New("invalid native decision state")
				}
				if inputs[id] {
					x[id] = v
				}
			}
			for a, group := range s.Outputs {
				q := 0.0
				for _, id := range nativeIDs(x) {
					for _, to := range group {
						e := edges[edge{id, to}]
						v := e.Weight * nativeReadoutScale
						if e.Inhibitory {
							v = -v
						}
						q += x[id] * v / 4
					}
				}
				if q != d.Values[a] || !nativeFinite(d.Probabilities[a]) || d.Probabilities[a] < 0 || d.Probabilities[a] > 1 {
					return errors.New("native logged value/propensity mismatch")
				}
			}
			best := d.Values[0]
			for _, q := range d.Values {
				best = math.Max(best, q)
			}
			sum := 0.0
			for _, q := range d.Values {
				sum += math.Exp((q - best) / s.Config.Temperature)
			}
			for a, q := range d.Values {
				if d.Probabilities[a] != math.Exp((q-best)/s.Config.Temperature)/sum {
					return errors.New("native propensity mismatch")
				}
			}
			previous = d.At
		}
		if len(ep.Decisions) > 0 {
			lastStop := s.Config.Operations[ep.Decisions[len(ep.Decisions)-1].Action].Kind == "stop"
			limit := len(ep.Decisions) == s.Config.MaxSteps
			if ep.Stopped != (lastStop || limit) || ep.Truncated != (limit && !lastStop) {
				return errors.New("native stop/budget mismatch")
			}
		}
		if ep.Stopped && len(ep.Decisions) == 0 {
			return errors.New("empty stopped episode")
		}
	}
	return nil
}
