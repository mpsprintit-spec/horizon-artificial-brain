package knowledge

// Explicit opt-in experiment. Historical Joint rows and production selectors
// keep their original meaning. Newly allocated heads are canonical N synapses.
import (
	"errors"
	"math"
	"math/rand"
	"sync"
	"time"
)

const Phase2VersionV5 = "phase2-native-world-v1"

type Phase2UpdateTimesV5 struct{ ParameterReadNS, ForwardBackwardNS, OptimizerNS int64 }

type Phase2SpecV5 struct {
	Version              string `json:"version"`
	Representation       string `json:"representation"`
	LearnGRU             bool   `json:"learn_gru"`
	Optimizer            string `json:"optimizer"`
	ConditionalSelection bool   `json:"conditional_selection"`
	SelectionSupport     []bool `json:"selection_support,omitempty"`
	FeatureSeed          int64  `json:"feature_seed"`
}

func (c Phase2SpecV5) Validate() error {
	if c.Version != Phase2VersionV5 {
		return errors.New("unknown explicit experimental world version")
	}
	switch c.Representation {
	case "categorical", "pairwise", "independent", "shuffled-pairwise":
	default:
		return errors.New("unknown experimental representation")
	}
	if c.Optimizer != "historical-global" && c.Optimizer != "new-head-effective" {
		return errors.New("unknown experimental Adam clock")
	}
	if c.Optimizer == "new-head-effective" && (c.LearnGRU || c.Representation == "categorical") {
		return errors.New("historical effective counts are unknown; only a newly allocated frozen-GRU head has exact counters")
	}
	return nil
}

type Phase2StateV5 struct {
	Spec           Phase2SpecV5    `json:"spec"`
	Nodes          []NodeID        `json:"nodes,omitempty"`
	Weights        CognitiveTensor `json:"weights"`
	Bias           CognitiveTensor `json:"bias"`
	BaseParameters int             `json:"base_parameters"`
	HeadSteps      uint64          `json:"head_steps"`
}
type Phase2BatchV5 struct {
	FreezeHead bool                   `json:"freeze_head,omitempty"` // optimizer audit only; main fitting leaves false
	Batch      CognitiveTrainingBatch `json:"batch"`
}

func phase2DesignV5(n NeuralMetaConfig, spec Phase2SpecV5) ([][]float64, error) {
	if err := spec.Validate(); err != nil {
		return nil, err
	}
	d, err := pairwiseDesignV5(n)
	if err != nil {
		return nil, err
	}
	if spec.Representation == "independent" {
		count := n.Classes - 1 + n.Computations*(n.Outcomes-1)
		for i := range d {
			d[i] = d[i][:count]
		}
	}
	if spec.Representation == "shuffled-pairwise" {
		order := rand.New(rand.NewSource(spec.FeatureSeed)).Perm(len(d))
		out := make([][]float64, len(d))
		for i, j := range order {
			out[i] = d[j]
		}
		d = out
	}
	return d, nil
}
func (k *KnowledgeBase) ValidatePhase2InitV5(spec Phase2SpecV5) error {
	k.RLock()
	defer k.RUnlock()
	if err := spec.Validate(); err != nil {
		return err
	}
	s := k.BrainState.CognitiveModel
	if s == nil || s.NeuralMeta == nil || s.Config.NeuralMeta == nil || s.Phase2V5 != nil {
		return errors.New("phase2 requires one restored unextended native cognitive model")
	}
	if spec.ConditionalSelection {
		if len(spec.SelectionSupport) != s.Config.NeuralMeta.States() {
			return errors.New("experimental exposure support shape")
		}
		any := false
		for _, v := range spec.SelectionSupport {
			any = any || v
		}
		if !any {
			return errors.New("empty exposure support")
		}
	} else if len(spec.SelectionSupport) != 0 {
		return errors.New("unconditional objective cannot carry exposure support")
	}
	return nil
}
func (k *KnowledgeBase) InitializePhase2V5(spec Phase2SpecV5, at time.Time) error {
	if err := k.ValidatePhase2InitV5(spec); err != nil {
		return err
	}
	if at.IsZero() {
		return errors.New("experimental allocation requires event time")
	}
	k.Lock()
	defer k.Unlock()
	s := k.BrainState.CognitiveModel
	if s.Phase2V5 != nil {
		return errors.New("experimental circuit already allocated")
	}
	p := &Phase2StateV5{Spec: spec, BaseParameters: len(s.Layout.Edges)}
	if spec.Representation == "categorical" {
		p.Weights, p.Bias = s.NeuralMeta.Joint, s.NeuralMeta.JointBias
	} else {
		d, err := phase2DesignV5(*s.Config.NeuralMeta, spec)
		if err != nil {
			return err
		}
		for i := 0; i < len(d[0]); i++ {
			coordinate := float64(1_000_000 + i)
			node := k.createRepresentationPrototypeAt(NewNeuralVector([]float64{coordinate, 1}), at)
			node.Activation = .5
			node.Importance = 1
			p.Nodes = append(p.Nodes, node.ID)
			scope := cognitiveScope
			k.projectionMu.Lock()
			k.ProjectionPopulations = append(k.ProjectionPopulations, ProjectionPopulation{Scope: &scope, Prototype: []float64{coordinate, 1}, Units: []PopulationUnit{{NodeID: node.ID, Activation: .5}}})
			k.projectionMu.Unlock()
		}
		p.Weights = CognitiveTensor{Offset: len(s.Layout.Edges), Rows: len(p.Nodes), Cols: s.Config.Hidden}
		for _, to := range p.Nodes {
			for _, from := range s.Layout.Hidden {
				s.Layout.Edges = append(s.Layout.Edges, CognitiveEdge{from, to})
			}
		}
		p.Bias = CognitiveTensor{Offset: len(s.Layout.Edges), Rows: len(p.Nodes), Cols: 1}
		for _, to := range p.Nodes {
			s.Layout.Edges = append(s.Layout.Edges, CognitiveEdge{s.Layout.BiasInput, to})
		}
		for _, e := range s.Layout.Edges[p.BaseParameters:] {
			k.adjustCompositionParameter([]NodeID{e.From}, []NodeID{e.To}, 0, 1, at, true)
		}
		s.FirstMoment = append(s.FirstMoment, make([]float64, len(s.Layout.Edges)-p.BaseParameters)...)
		s.SecondMoment = append(s.SecondMoment, make([]float64, len(s.Layout.Edges)-p.BaseParameters)...)
	}
	s.Phase2V5 = p
	return nil
}
func phase2TensorsV5(s *CognitiveModelState) (CognitiveTensor, CognitiveTensor) {
	if s.Phase2V5 == nil {
		return s.NeuralMeta.Joint, s.NeuralMeta.JointBias
	}
	return s.Phase2V5.Weights, s.Phase2V5.Bias
}

type phase2StructureKeyV5 struct {
	Classes, Operations, Outcomes int
	Representation                string
	Seed                          int64
}
type phase2StructureV5 struct {
	Dense  [][]float64
	Active [][]int
}

var phase2StructureCacheV5 sync.Map // immutable incidence only, never N weights, observations or beliefs
func phase2StructureForV5(n NeuralMetaConfig, spec Phase2SpecV5) (*phase2StructureV5, error) {
	key := phase2StructureKeyV5{n.Classes, n.Computations, n.Outcomes, spec.Representation, spec.FeatureSeed}
	if v, ok := phase2StructureCacheV5.Load(key); ok {
		return v.(*phase2StructureV5), nil
	}
	d, err := phase2DesignV5(n, spec)
	if err != nil {
		return nil, err
	}
	v := &phase2StructureV5{Dense: d, Active: make([][]int, len(d))}
	for s, row := range d {
		for f, x := range row {
			if x != 0 {
				v.Active[s] = append(v.Active[s], f)
			}
		}
	}
	actual, _ := phase2StructureCacheV5.LoadOrStore(key, v)
	return actual.(*phase2StructureV5), nil
}
func phase2LogitsV5(s *CognitiveModelState, w, h []float64, dense bool) ([]float64, error) {
	if s.Phase2V5 == nil || s.Phase2V5.Spec.Representation == "categorical" {
		z := mat(w, s.NeuralMeta.Joint, h)
		addBias(z, w, s.NeuralMeta.JointBias)
		return z, nil
	}
	p := s.Phase2V5
	structure, err := phase2StructureForV5(*s.Config.NeuralMeta, p.Spec)
	if err != nil {
		return nil, err
	}
	v := mat(w, p.Weights, h)
	addBias(v, w, p.Bias)
	z := make([]float64, len(structure.Dense))
	if dense {
		for state, row := range structure.Dense {
			for j, x := range row {
				z[state] += x * v[j]
			}
		}
	} else {
		for state, active := range structure.Active {
			for _, j := range active {
				z[state] += v[j]
			}
		}
	}
	return z, nil
}

// The selection rule is an observed training protocol, never an inference law.
// Conditional NLL log Z_S-log Z_(event intersect S) is a proper likelihood for
// selection-truncated training; it does not label or assign unseen probability.
func phase2PartialV5(z []float64, event, support []bool) (float64, []float64, error) {
	if len(z) != len(event) || len(z) != len(support) {
		return 0, nil, errors.New("world likelihood mask shape")
	}
	ma, me := math.Inf(-1), math.Inf(-1)
	for i, v := range z {
		if support[i] {
			ma = math.Max(ma, v)
			if event[i] {
				me = math.Max(me, v)
			}
		}
	}
	if math.IsInf(ma, -1) || math.IsInf(me, -1) {
		return 0, nil, errors.New("empty selection or verified event")
	}
	a, b := 0., 0.
	for i, v := range z {
		if support[i] {
			a += math.Exp(v - ma)
			if event[i] {
				b += math.Exp(v - me)
			}
		}
	}
	g := make([]float64, len(z))
	for i, v := range z {
		if support[i] {
			g[i] = math.Exp(v-ma) / a
			if event[i] {
				g[i] -= math.Exp(v-me) / b
			}
		}
	}
	return ma + math.Log(a) - me - math.Log(b), g, nil
}
func phase2MutableV5(s *CognitiveModelState, i int) bool {
	if terminalReadoutParameterV3(s, i) {
		return false
	}
	p := s.Phase2V5
	in := func(t CognitiveTensor) bool { return i >= t.Offset && i < t.Offset+t.Rows*t.Cols }
	if in(p.Weights) || in(p.Bias) {
		return true
	}
	return p.Spec.LearnGRU && (in(s.Layout.WI) || in(s.Layout.WH) || in(s.Layout.BI) || in(s.Layout.BH))
}
func phase2ObjectiveV5(s *CognitiveModelState, w []float64, b CognitiveTrainingBatch) (CognitiveLearningResult, []float64, error) {
	stats := CognitiveLearningResult{Episodes: len(b.Episodes)}
	g := make([]float64, len(w))
	n := *s.Config.NeuralMeta
	support := make([]bool, n.States())
	for i := range support {
		support[i] = true
		if s.Phase2V5.Spec.ConditionalSelection {
			support[i] = s.Phase2V5.Spec.SelectionSupport[i]
		}
	}
	scale := n.JointWeight / float64(len(b.Episodes))
	j, bias := phase2TensorsV5(s)
	var design [][]float64
	if s.Phase2V5.Spec.Representation != "categorical" {
		structure, err := phase2StructureForV5(n, s.Phase2V5.Spec)
		if err != nil {
			return stats, nil, err
		}
		design = structure.Dense
	}
	for _, ep := range b.Episodes {
		allowed := metaAllowed(n, ep)
		frames := make([]gruFrame, len(ep.Frames))
		local := make([][]float64, len(frames))
		h := make([]float64, s.Config.Hidden)
		for t, f := range ep.Frames {
			if b.NoRecurrence {
				clear(h)
			}
			x := append([]float64(nil), f.Features...)
			if f.NeuralMeta.NoFeedback {
				clear(x[n.ResultOffset : n.ResultOffset+n.Outcomes])
				clear(x[n.OperationOffset : n.OperationOffset+n.Computations])
			}
			frames[t] = cognitiveGRU(s.Config, s.Layout, w, x, h)
			h = frames[t].h
			z, err := phase2LogitsV5(s, w, h, false)
			if err != nil {
				return stats, nil, err
			}
			loss, dz, err := phase2PartialV5(z, allowed, support)
			if err != nil {
				return stats, nil, err
			}
			stats.Joint += loss / float64(len(b.Episodes))
			df := dz
			if design != nil {
				df = make([]float64, j.Rows)
				for state, row := range design {
					for f, x := range row {
						if x != 0 {
							df[f] += dz[state] * x
						}
					}
				}
			}
			for i := range df {
				df[i] *= scale
			}
			local[t] = make([]float64, s.Config.Hidden)
			cognitiveLinearBackward(g, local[t], w, h, df, j, bias)
		}
		if s.Phase2V5.Spec.LearnGRU {
			next := make([]float64, s.Config.Hidden)
			for t := len(frames) - 1; t >= 0; t-- {
				for i := range next {
					local[t][i] += next[i]
				}
				next = cognitiveGRUBackward(s.Config, s.Layout, w, g, frames[t], local[t])
				if b.NoRecurrence {
					clear(next)
				}
			}
		}
	}
	stats.Loss = n.JointWeight * stats.Joint
	return stats, g, nil
}
func (k *KnowledgeBase) ValidatePhase2BatchV5(b Phase2BatchV5) error {
	k.RLock()
	defer k.RUnlock()
	s := k.BrainState.CognitiveModel
	if s == nil || s.Phase2V5 == nil {
		return errors.New("explicit experimental allocation required")
	}
	if b.Batch.LearnPolicy || b.Batch.TerminalReadoutOnly || b.Batch.ComponentV4 != "" || b.Batch.EntropyWeightV4 != nil {
		return errors.New("world experiment cannot update policy or mix objectives")
	}
	if err := b.Batch.Validate(s.Config); err != nil {
		return err
	}
	w, err := k.cognitiveWeights(s)
	if err != nil {
		return err
	}
	if CognitiveDigest(w) != b.Batch.ParametersHash {
		return errors.New("stale experimental synaptic digest")
	}
	if s.Phase2V5.Spec.ConditionalSelection {
		for _, e := range b.Batch.Episodes {
			mask := metaAllowed(*s.Config.NeuralMeta, e)
			any := false
			for state, v := range mask {
				if v && s.Phase2V5.Spec.SelectionSupport[state] {
					any = true
				}
			}
			if !any {
				return errors.New("outcome outside documented training exposure")
			}
		}
	}
	return nil
}
func (k *KnowledgeBase) LearnPhase2V5(batch Phase2BatchV5, at time.Time) (CognitiveLearningResult, error) {
	if err := k.ValidatePhase2BatchV5(batch); err != nil {
		return CognitiveLearningResult{}, err
	}
	k.Lock()
	defer k.Unlock()
	s := k.BrainState.CognitiveModel
	begin := time.Now()
	w, err := k.cognitiveWeights(s)
	if err != nil {
		return CognitiveLearningResult{}, err
	}
	if CognitiveDigest(w) != batch.Batch.ParametersHash {
		return CognitiveLearningResult{}, errors.New("experimental weights changed")
	}
	readNS := time.Since(begin).Nanoseconds()
	begin = time.Now()
	stats, g, err := phase2ObjectiveV5(s, w, batch.Batch)
	if err != nil {
		return stats, err
	}
	backNS := time.Since(begin).Nanoseconds()
	begin = time.Now()
	mutable := func(i int) bool {
		p := s.Phase2V5
		if batch.FreezeHead && ((i >= p.Weights.Offset && i < p.Weights.Offset+p.Weights.Rows*p.Weights.Cols) || (i >= p.Bias.Offset && i < p.Bias.Offset+p.Bias.Rows*p.Bias.Cols)) {
			return false
		}
		return phase2MutableV5(s, i)
	}
	ss := 0.
	for i, d := range g {
		if !finiteNumber(d) {
			return stats, errors.New("nonfinite world gradient")
		}
		if mutable(i) {
			ss += d * d
		}
	}
	stats.GradientNorm = math.Sqrt(ss)
	clip := 1.
	if stats.GradientNorm > 1 {
		clip = 1 / stats.GradientNorm
	}
	s.OptimizerStep++
	if !batch.FreezeHead {
		s.Phase2V5.HeadSteps++
	}
	for i, d := range g {
		if !mutable(i) {
			continue
		}
		d *= clip
		s.FirstMoment[i] = .9*s.FirstMoment[i] + .1*d
		s.SecondMoment[i] = .999*s.SecondMoment[i] + .001*d*d
		t := float64(s.OptimizerStep)
		if s.Phase2V5.Spec.Optimizer == "new-head-effective" {
			t = float64(s.Phase2V5.HeadSteps)
		}
		delta := s.Config.LearningRate * (s.FirstMoment[i] / (1 - math.Pow(.9, t))) / (math.Sqrt(s.SecondMoment[i]/(1-math.Pow(.999, t))) + 1e-8)
		v := w[i] - delta
		if math.Abs(v) > 7.9 {
			stats.ClippedParameters++
			v = math.Copysign(7.9, v)
		}
		e := s.Layout.Edges[i]
		k.adjustCompositionParameter([]NodeID{e.From}, []NodeID{e.To}, v, 1, at, false)
	}
	s.LearnedEpisodes += uint64(len(batch.Batch.Episodes))
	ev := batch.Batch.Evidence
	s.LastEvidence = &ev
	stats.OptimizerStep = s.OptimizerStep
	stats.Phase2Times = &Phase2UpdateTimesV5{ParameterReadNS: readNS, ForwardBackwardNS: backNS, OptimizerNS: time.Since(begin).Nanoseconds()}
	return stats, nil
}
func (k *KnowledgeBase) HasPhase2V5() bool {
	k.RLock()
	defer k.RUnlock()
	return k.BrainState.CognitiveModel != nil && k.BrainState.CognitiveModel.Phase2V5 != nil
}
