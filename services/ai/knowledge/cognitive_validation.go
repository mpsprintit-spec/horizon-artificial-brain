package knowledge

import (
	"bytes"
	"encoding/json"
	"errors"
	"math"
	"slices"
)

// Validate the complete topology without constructing a second Brain. This is
// called BEFORE restoring a checkpoint into a live substrate.
func ValidateCognitiveModelState(data []byte, enabled bool) error {
	if !enabled && !bytes.Contains(data, []byte("cognitive_model")) && !bytes.Contains(data, []byte(CognitiveModelVersion)) {
		return nil
	}
	var raw struct {
		SchemaVersion int                    `json:"schema_version"`
		Model         *CognitiveModelState   `json:"cognitive_model"`
		Units         []canonicalNeuralUnit  `json:"neural_units"`
		Populations   []ProjectionPopulation `json:"populations"`
		Synapses      []canonicalSynapse     `json:"synapses"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if raw.Model == nil {
		for _, p := range raw.Populations {
			if p.Scope != nil && *p.Scope == cognitiveScope {
				return errors.New("orphan cognitive population")
			}
		}
		if enabled {
			return errors.New("missing cognitive model")
		}
		return nil
	}
	if !enabled || (raw.SchemaVersion != 4 && raw.SchemaVersion != 5 && raw.SchemaVersion != 6) || (raw.SchemaVersion == 4 && raw.Model.Config.Version != CognitiveModelVersion) || (raw.SchemaVersion >= 5 && raw.Model.Config.Version != CognitiveModelV2) {
		return errors.New("cognitive model requires authenticated canonical schema 4 / checkpoint 15")
	}
	s := raw.Model
	coordinated := s.Phase4V5 != nil && s.Phase4V5.Contract != ""
	if coordinated != (raw.SchemaVersion == 6) || coordinated && s.Phase4V5.Contract != CoordinatedContractV5 {
		return errors.New("coordinated canonical schema6 required")
	}
	c, l := s.Config, s.Layout
	if err := c.Validate(); err != nil {
		return err
	}
	if len(l.Inputs) != c.Inputs || len(l.Hidden) != c.Hidden || len(l.Gates) != 3*c.Hidden || len(l.Actions) != c.Actions || len(l.Success) != 5 || len(l.Value) != 1 || len(l.World) != c.Inputs || len(l.BMPS) != 4 {
		return errors.New("cognitive layout dimension mismatch")
	}
	ids := []NodeID{}
	for _, group := range [][]NodeID{l.Inputs, l.Hidden, l.Gates, l.Actions, l.Success, l.Value, l.World, l.BMPS, {l.BiasInput, l.BiasHidden}} {
		ids = append(ids, group...)
	}
	if c.NeuralMeta != nil {
		m := s.NeuralMeta
		if m == nil || len(m.JointNodes) != c.NeuralMeta.States() || len(m.RewardNodes) != 1 || len(m.TemperatureNodes) != 6 {
			return errors.New("invalid native world topology")
		}
		ids = append(ids, m.JointNodes...)
		ids = append(ids, m.RewardNodes...)
		ids = append(ids, m.TemperatureNodes...)
	} else if s.NeuralMeta != nil {
		return errors.New("V1 carries unauthenticated V2 state")
	}
	if s.Phase2V5 != nil {
		p := s.Phase2V5
		if c.NeuralMeta == nil || p.Spec.Validate() != nil || p.HeadSteps > s.OptimizerStep {
			return errors.New("invalid versioned phase2 metadata")
		}
		if p.Spec.ConditionalSelection {
			if len(p.Spec.SelectionSupport) != c.NeuralMeta.States() {
				return errors.New("invalid phase2 exposure mask")
			}
			any := false
			for _, v := range p.Spec.SelectionSupport {
				any = any || v
			}
			if !any {
				return errors.New("empty phase2 exposure")
			}
		} else if len(p.Spec.SelectionSupport) != 0 {
			return errors.New("foreign phase2 exposure")
		}
		if p.Spec.Representation == "categorical" {
			if len(p.Nodes) != 0 || p.Weights != s.NeuralMeta.Joint || p.Bias != s.NeuralMeta.JointBias {
				return errors.New("categorical phase2 cannot reinterpret or allocate world rows")
			}
		} else {
			d, e := phase2DesignV5(*c.NeuralMeta, p.Spec)
			if e != nil || len(p.Nodes) != len(d[0]) {
				return errors.New("phase2 factor topology mismatch")
			}
		}
		ids = append(ids, p.Nodes...)
	}
	if p := s.Phase4V5; p != nil {
		if p.Spec.Validate() != nil || c.Hidden != 32 || c.Actions != 8 || s.Phase2V5 == nil || len(p.Features) != 58 || len(p.Hidden) != 16 || len(p.Actions) != 8 || len(p.Value) != 1 {
			return errors.New("invalid Phase IV N metadata")
		}
		ids = append(ids, p.Features...)
		ids = append(ids, p.Hidden...)
		ids = append(ids, p.Actions...)
		ids = append(ids, p.Value...)
		if p.LastEvidence != nil {
			if e := p.LastEvidence.Validate(); e != nil {
				return e
			}
		}
	}
	seen := map[NodeID]bool{}
	known := map[NodeID]canonicalNeuralUnit{}
	for _, u := range raw.Units {
		known[u.ID] = u
	}
	for _, id := range ids {
		u, ok := known[id]
		if id == 0 || !ok || seen[id] || !finiteNumber(u.Activation) || u.Activation < 0 || u.Activation > 1 {
			return errors.New("invalid or repeated cognitive neuron")
		}
		seen[id] = true
	}
	scoped := map[NodeID]bool{}
	for _, p := range raw.Populations {
		if p.Scope != nil && *p.Scope == cognitiveScope {
			if len(p.Units) != 1 || !seen[p.Units[0].NodeID] || scoped[p.Units[0].NodeID] {
				return errors.New("invalid cognitive population")
			}
			scoped[p.Units[0].NodeID] = true
		}
	}
	if len(scoped) != len(seen) {
		return errors.New("missing cognitive population")
	}
	expected := []CognitiveEdge{}
	tensor := func(from, to []NodeID, t CognitiveTensor) bool {
		if t.Offset != len(expected) || t.Rows != len(to) || t.Cols != len(from) {
			return false
		}
		for _, b := range to {
			for _, a := range from {
				expected = append(expected, CognitiveEdge{a, b})
			}
		}
		return true
	}
	if !tensor(l.Inputs, l.Gates, l.WI) || !tensor(l.Hidden, l.Gates, l.WH) || !tensor([]NodeID{l.BiasInput}, l.Gates, l.BI) || !tensor([]NodeID{l.BiasHidden}, l.Gates, l.BH) || !tensor(l.Hidden, l.Actions, l.Actor) || !tensor([]NodeID{l.BiasInput}, l.Actions, l.ActorBias) || !tensor(l.Hidden, l.Value, l.Critic) || !tensor([]NodeID{l.BiasInput}, l.Value, l.CriticBias) || !tensor(l.Hidden, l.Success, l.Assessment) || !tensor([]NodeID{l.BiasInput}, l.Success, l.AssessmentBias) || !tensor(append(append([]NodeID{}, l.Hidden...), l.Actions...), l.World, l.Decoder) || !tensor([]NodeID{l.BiasInput}, l.World, l.DecoderBias) {
		return errors.New("invalid cognitive tensor topology")
	}
	if c.NeuralMeta != nil {
		m := s.NeuralMeta
		if !tensor(l.Hidden, m.JointNodes, m.Joint) || !tensor([]NodeID{l.BiasInput}, m.JointNodes, m.JointBias) || !tensor(l.Hidden, m.RewardNodes, m.Reward) || !tensor([]NodeID{l.BiasInput}, m.RewardNodes, m.RewardBias) {
			return errors.New("invalid native world tensor")
		}
		if m.CalibrationEvidence != nil {
			if err := m.CalibrationEvidence.Validate(); err != nil {
				return err
			}
		}
	}
	if p := s.Phase2V5; p != nil {
		if p.BaseParameters != len(expected) {
			return errors.New("phase2 base tensor boundary differs")
		}
		if p.Spec.Representation != "categorical" && (!tensor(l.Hidden, p.Nodes, p.Weights) || !tensor([]NodeID{l.BiasInput}, p.Nodes, p.Bias)) {
			return errors.New("invalid separately allocated phase2 tensor")
		}
	}
	if p := s.Phase4V5; p != nil {
		if p.BaseParameters != len(expected) || !tensor(p.Features, p.Hidden, p.WI) || !tensor([]NodeID{l.BiasInput}, p.Hidden, p.BI) || !tensor(p.Hidden, p.Actions, p.Actor) || !tensor([]NodeID{l.BiasInput}, p.Actions, p.ActorBias) || !tensor(p.Hidden, p.Value, p.Critic) || !tensor([]NodeID{l.BiasInput}, p.Value, p.CriticBias) || len(expected)-p.BaseParameters != 1097 {
			return errors.New("invalid Phase IV complete tensor topology")
		}
	}
	if !slices.Equal(expected, l.Edges) {
		return errors.New("invalid complete cognitive parameter topology")
	}
	n := len(expected)
	if len(s.FirstMoment) != n || len(s.SecondMoment) != n {
		return errors.New("invalid cognitive optimizer shape")
	}
	for i, x := range s.FirstMoment {
		if !finiteNumber(x) || !finiteNumber(s.SecondMoment[i]) || s.SecondMoment[i] < 0 {
			return errors.New("invalid cognitive optimizer moment")
		}
	}
	type key struct {
		a, b NodeID
		inh  bool
	}
	edges := map[key]canonicalSynapse{}
	for _, e := range raw.Synapses {
		if !seen[e.SourceID] && !seen[e.TargetID] {
			continue
		}
		if !seen[e.SourceID] || !seen[e.TargetID] {
			return errors.New("unregistered cross-circuit cognitive connection")
		}
		q := key{e.SourceID, e.TargetID, e.Inhibitory}
		if _, ok := edges[q]; ok || e.Kind != "" || !finiteNumber(e.Dynamic.Weight) || e.Dynamic.Weight < 0 || e.Dynamic.Weight > 1 {
			return errors.New("invalid cognitive synapse")
		}
		edges[q] = e
	}
	expected = append(expected, []CognitiveEdge{{l.BiasInput, l.BMPS[0]}, {l.BiasInput, l.BMPS[1]}, {l.BiasInput, l.BMPS[2]}, {l.BiasInput, l.BMPS[3]}}...)
	if s.NeuralMeta != nil {
		for _, id := range s.NeuralMeta.TemperatureNodes {
			expected = append(expected, CognitiveEdge{l.BiasInput, id})
		}
	}
	for _, e := range expected {
		for _, inh := range []bool{false, true} {
			if _, ok := edges[key{e.From, e.To, inh}]; !ok {
				return errors.New("missing signed cognitive parameter")
			}
		}
	}
	if len(edges) != 2*len(expected) {
		return errors.New("unregistered cognitive parameter")
	}
	var weights [4]float64
	for i, id := range l.BMPS {
		weights[i] = compositionScale * (edges[key{l.BiasInput, id, false}].Dynamic.Weight - edges[key{l.BiasInput, id, true}].Dynamic.Weight)
	}
	weights[3] *= float64(c.Horizon)
	if err := ValidateBMPSWeights(weights, c.Horizon); err != nil {
		return err
	}
	if s.LastEvidence != nil {
		if err := s.LastEvidence.Validate(); err != nil {
			return err
		}
	}
	for _, e := range s.Search {
		if err := e.Validate(); err != nil {
			return err
		}
	}
	if math.IsInf(float64(s.OptimizerStep), 0) {
		return errors.New("invalid optimizer step")
	}
	return nil
}
