package knowledge

// Internal research adaptation. All persistent parameters below are ordinary
// dynamic synapses in the owning Brain. Layouts and optimizer moments are not
// another predictor, policy, lexical table, or Brain.
// Sources: Valiente & Pilly, 10.1016/j.neunet.2025.108131 (SA eq.4,
// regulation option c); Callaway et al., arXiv:1711.06892 (eqs.2-5);
// Chen et al., 10.64898/2026.04.14.718499 (shared GRU, policy gradient).
import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"
)

const CognitiveModelVersion = "internal-shared-gru-muse-bmps-v1"

var cognitiveScope = ProjectionScope{Modality: "internal-numeric", Field: CognitiveModelVersion}

type CognitiveModelConfig struct {
	NeuralMeta       *NeuralMetaConfig `json:"neural_meta,omitempty"`
	Version          string            `json:"version"`
	Inputs           int               `json:"inputs"`
	Hidden           int               `json:"hidden"`
	Actions          int               `json:"actions"`
	Horizon          int               `json:"horizon"`
	Seed             int64             `json:"seed"`
	LearningRate     float64           `json:"learning_rate"`
	Gamma            float64           `json:"gamma"`
	ValueWeight      float64           `json:"value_weight"`
	EntropyWeight    float64           `json:"entropy_weight"`
	AssessmentWeight float64           `json:"assessment_weight"`
	WorldWeight      float64           `json:"world_weight"`
	RegulationSteps  int               `json:"regulation_steps"`
	RegulationRate   float64           `json:"regulation_rate"`
}

func (c CognitiveModelConfig) Validate() error {
	if (c.Version != CognitiveModelVersion && c.Version != CognitiveModelV2) || c.Inputs < 1 || c.Hidden < 1 || c.Actions < 2 || c.Horizon < 1 || c.RegulationSteps < 0 {
		return errors.New("invalid cognitive model dimensions or version")
	}
	for _, x := range []float64{c.LearningRate, c.Gamma, c.ValueWeight, c.EntropyWeight, c.AssessmentWeight, c.WorldWeight, c.RegulationRate} {
		if !finiteNumber(x) || x < 0 {
			return errors.New("invalid cognitive learning parameter")
		}
	}
	if c.LearningRate <= 0 || c.LearningRate > .1 || c.Gamma > 1 {
		return errors.New("invalid cognitive learning rate/discount")
	}
	// An explicit resource budget is a caller configuration, not a global N cap.
	if c.Inputs > 1_000_000/c.Hidden || c.Hidden > 1_000_000/c.Hidden || c.Actions > 1_000_000/c.Hidden {
		return errors.New("cognitive circuit exceeds single-command allocation budget")
	}
	if c.Version == CognitiveModelV2 {
		if c.NeuralMeta == nil {
			return errors.New("V2 requires its native learned world decoder")
		}
		return c.NeuralMeta.Validate(c)
	}
	if c.NeuralMeta != nil {
		return errors.New("V1 cannot silently acquire V2 metadata")
	}
	return nil
}
func finiteNumber(x float64) bool { return !math.IsNaN(x) && !math.IsInf(x, 0) }

type CognitiveEdge struct {
	From NodeID `json:"from"`
	To   NodeID `json:"to"`
}
type CognitiveTensor struct {
	Offset int `json:"offset"`
	Rows   int `json:"rows"`
	Cols   int `json:"cols"`
}
type CognitiveLayout struct {
	Inputs         []NodeID        `json:"inputs"`
	Hidden         []NodeID        `json:"hidden"`
	Gates          []NodeID        `json:"gates"`
	Actions        []NodeID        `json:"actions"`
	Success        []NodeID        `json:"success"`
	Value          []NodeID        `json:"value"`
	World          []NodeID        `json:"world"`
	BiasInput      NodeID          `json:"bias_input"`
	BiasHidden     NodeID          `json:"bias_hidden"`
	BMPS           []NodeID        `json:"bmps"`
	Edges          []CognitiveEdge `json:"edges"`
	WI             CognitiveTensor `json:"wi"`
	WH             CognitiveTensor `json:"wh"`
	BI             CognitiveTensor `json:"bi"`
	BH             CognitiveTensor `json:"bh"`
	Actor          CognitiveTensor `json:"actor"`
	ActorBias      CognitiveTensor `json:"actor_bias"`
	Critic         CognitiveTensor `json:"critic"`
	CriticBias     CognitiveTensor `json:"critic_bias"`
	Assessment     CognitiveTensor `json:"assessment"`
	AssessmentBias CognitiveTensor `json:"assessment_bias"`
	Decoder        CognitiveTensor `json:"decoder"`
	DecoderBias    CognitiveTensor `json:"decoder_bias"`
}
type CognitiveModelState struct {
	Phase4V5        *Phase4StateV5       `json:"phase4_v5,omitempty"`
	Phase2V5        *Phase2StateV5       `json:"phase2_v5,omitempty"`
	NeuralMeta      *NeuralMetaState     `json:"neural_meta,omitempty"`
	Config          CognitiveModelConfig `json:"config"`
	Layout          CognitiveLayout      `json:"layout"`
	OptimizerStep   uint64               `json:"optimizer_step"`
	FirstMoment     []float64            `json:"first_moment"`
	SecondMoment    []float64            `json:"second_moment"`
	LearnedEpisodes uint64               `json:"learned_episodes"`
	ModelSteps      uint64               `json:"model_steps"`
	Search          []BMPSEvaluation     `json:"search,omitempty"`
	LastEvidence    *CognitiveEvidence   `json:"last_evidence,omitempty"`
}
type CognitiveEvidence struct {
	ID       string `json:"id"`
	Source   string `json:"source"`
	Verifier string `json:"verifier"`
	Digest   string `json:"digest"`
}

func (e CognitiveEvidence) Validate() error {
	b, err := hex.DecodeString(e.Digest)
	if e.ID == "" || e.Source == "" || e.Verifier == "" || err != nil || len(b) != 32 {
		return errors.New("verified cognitive feedback requires identity, source, verifier and SHA256 digest")
	}
	return nil
}
func CognitiveDigest(v any) string {
	b, _ := json.Marshal(v)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func (k *KnowledgeBase) HasCognitiveModel() bool {
	k.RLock()
	defer k.RUnlock()
	return k.BrainState.CognitiveModel != nil
}
func (k *KnowledgeBase) InitializeCognitiveModel(c CognitiveModelConfig, at time.Time) error {
	if err := c.Validate(); err != nil {
		return err
	}
	if at.IsZero() {
		return errors.New("cognitive initialization needs event time")
	}
	k.Lock()
	defer k.Unlock()
	if k.BrainState.CognitiveModel != nil {
		return errors.New("cognitive model already initialized")
	}
	l := CognitiveLayout{}
	coordinate := 0
	alloc := func(n int) []NodeID {
		out := make([]NodeID, n)
		for i := range out {
			v := NewNeuralVector([]float64{float64(coordinate), 1})
			node := k.createRepresentationPrototypeAt(v, at)
			node.Activation = .5
			node.Importance = 1
			out[i] = node.ID
			scope := cognitiveScope
			k.projectionMu.Lock()
			k.ProjectionPopulations = append(k.ProjectionPopulations, ProjectionPopulation{Scope: &scope, Prototype: []float64{float64(coordinate), 1}, Units: []PopulationUnit{{NodeID: node.ID, Activation: .5}}})
			k.projectionMu.Unlock()
			coordinate++
		}
		return out
	}
	l.Inputs = alloc(c.Inputs)
	l.Hidden = alloc(c.Hidden)
	l.Gates = alloc(3 * c.Hidden)
	l.Actions = alloc(c.Actions)
	l.Success = alloc(5)
	l.Value = alloc(1)
	l.World = alloc(c.Inputs)
	l.BiasInput = alloc(1)[0]
	l.BiasHidden = alloc(1)[0]
	l.BMPS = alloc(4)
	tensor := func(from, to []NodeID) CognitiveTensor {
		t := CognitiveTensor{Offset: len(l.Edges), Rows: len(to), Cols: len(from)}
		for _, b := range to {
			for _, a := range from {
				l.Edges = append(l.Edges, CognitiveEdge{a, b})
			}
		}
		return t
	}
	l.WI = tensor(l.Inputs, l.Gates)
	l.WH = tensor(l.Hidden, l.Gates)
	l.BI = tensor([]NodeID{l.BiasInput}, l.Gates)
	l.BH = tensor([]NodeID{l.BiasHidden}, l.Gates)
	l.Actor = tensor(l.Hidden, l.Actions)
	l.ActorBias = tensor([]NodeID{l.BiasInput}, l.Actions)
	l.Critic = tensor(l.Hidden, l.Value)
	l.CriticBias = tensor([]NodeID{l.BiasInput}, l.Value)
	l.Assessment = tensor(l.Hidden, l.Success)
	l.AssessmentBias = tensor([]NodeID{l.BiasInput}, l.Success)
	l.Decoder = tensor(append(append([]NodeID{}, l.Hidden...), l.Actions...), l.World)
	l.DecoderBias = tensor([]NodeID{l.BiasInput}, l.World)
	state := &CognitiveModelState{Config: c, Layout: l}
	if c.NeuralMeta != nil {
		k.InitializeNeuralMetaExtension(state, at)
		l = state.Layout
	}
	rng := rand.New(rand.NewSource(c.Seed))
	bound := 1 / math.Sqrt(float64(c.Hidden))
	for _, e := range l.Edges {
		k.adjustCompositionParameter([]NodeID{e.From}, []NodeID{e.To}, (2*rng.Float64()-1)*bound, 1, at, true)
	}
	// BMPS constraints: simplex mixture; cost multiplier in [1,h]. Store the
	// latter divided by h to reuse Horizon's existing bounded signed codec.
	for i, id := range l.BMPS {
		x := 1.0 / 3
		if i == 3 {
			x = 1 / float64(c.Horizon)
		}
		k.adjustCompositionParameter([]NodeID{l.BiasInput}, []NodeID{id}, x, 1, at, true)
	}
	state.FirstMoment = make([]float64, len(l.Edges))
	state.SecondMoment = make([]float64, len(l.Edges))
	k.BrainState.CognitiveModel = state
	return nil
}
func (k *KnowledgeBase) cognitiveWeights(s *CognitiveModelState) ([]float64, error) {
	w := make([]float64, len(s.Layout.Edges))
	if s.Config.Version == CognitiveModelV2 {
		k.Registry.mu.RLock()
		defer k.Registry.mu.RUnlock()
		for i, e := range s.Layout.Edges {
			node := k.Registry.byID[e.From]
			if node == nil {
				return nil, errors.New("missing canonical source")
			}
			list := node.Synapses[e.To]
			a, b := list.FindDynamic(false), list.FindDynamic(true)
			if a == nil || b == nil {
				return nil, errors.New("missing signed canonical parameter")
			}
			w[i] = compositionScale * (a.Dynamic.Weight - b.Dynamic.Weight)
			if !finiteNumber(w[i]) {
				return nil, errors.New("nonfinite canonical parameter")
			}
		}
		return w, nil
	}
	for i, e := range s.Layout.Edges {
		v, ok := k.compositionParameter([]NodeID{e.From}, []NodeID{e.To})
		if !ok || !finiteNumber(v) {
			return nil, fmt.Errorf("missing cognitive synapse %d", i)
		}
		w[i] = v
	}
	return w, nil
}
func (k *KnowledgeBase) cognitiveHidden(s *CognitiveModelState) []float64 {
	h := make([]float64, s.Config.Hidden)
	for i, id := range s.Layout.Hidden {
		h[i] = 2*k.Registry.GetByID(id).Activation - 1
	}
	return h
}
func (k *KnowledgeBase) setCognitiveHidden(s *CognitiveModelState, h []float64, at time.Time) {
	for i, id := range s.Layout.Hidden {
		n := k.Registry.GetByID(id)
		n.Activation = (h[i] + 1) / 2
		n.LastActivation = at
	}
}
func (k *KnowledgeBase) CognitiveParameters() ([]float64, error) {
	k.RLock()
	defer k.RUnlock()
	s := k.BrainState.CognitiveModel
	if s == nil {
		return nil, errors.New("cognitive model not initialized")
	}
	return k.cognitiveWeights(s)
}
func (k *KnowledgeBase) CognitiveConfig() (CognitiveModelConfig, error) {
	k.RLock()
	defer k.RUnlock()
	if k.BrainState.CognitiveModel == nil {
		return CognitiveModelConfig{}, errors.New("cognitive model not initialized")
	}
	c := k.BrainState.CognitiveModel.Config
	if c.NeuralMeta != nil {
		copy := *c.NeuralMeta
		c.NeuralMeta = &copy
	}
	return c, nil
}

type CognitiveInput struct {
	V1RegulationAblation string           `json:"v1_regulation_ablation,omitempty"` // diagnostic only; empty preserves V1 exactly
	NeuralMeta           *NeuralMetaInput `json:"neural_meta,omitempty"`
	Features             []float64        `json:"features"`
	Mask                 []bool           `json:"mask"`
	Reset                bool             `json:"reset"`
	Draw                 float64          `json:"draw"`                    // frozen uniform variate; replay never draws randomness
	NoRecurrence         bool             `json:"no_recurrence,omitempty"` // explicit ablation only
	Regulate             bool             `json:"regulate,omitempty"`
	Belief               *MetaBelief      `json:"belief,omitempty"`         // explicit BMPS metalevel model
	AssessUtility        bool             `json:"assess_utility,omitempty"` // expected-success utility model assumption
}

func (x CognitiveInput) Validate(c CognitiveModelConfig) error {
	if x.V1RegulationAblation != "" {
		if c.Version != CognitiveModelVersion || !x.Regulate {
			return errors.New("V1 counterfactual requires legacy regulation")
		}
		switch x.V1RegulationAblation {
		case "decoder-only", "gradient-only", "full-audit":
		default:
			return errors.New("unknown V1 counterfactual")
		}
	}
	if len(x.Features) != c.Inputs || len(x.Mask) != c.Actions || !finiteNumber(x.Draw) || x.Draw < 0 || x.Draw >= 1 {
		return errors.New("invalid cognitive input shape/draw")
	}
	valid := false
	for _, b := range x.Mask {
		valid = valid || b
	}
	if !valid {
		return errors.New("all cognitive actions masked")
	}
	for _, v := range x.Features {
		if !finiteNumber(v) || math.Abs(v) > 1 {
			return errors.New("cognitive inputs require normalized finite measurements")
		}
	}
	if c.Version == CognitiveModelV2 {
		if x.NeuralMeta == nil {
			return errors.New("V2 requires explicit native metalevel contract")
		}
		if err := x.NeuralMeta.Validate(c); err != nil {
			return err
		}
		if x.AssessUtility || x.Regulate {
			return errors.New("V1 utility scaling/latent ascent is not a V2 integration")
		}
		if x.Belief != nil && x.NeuralMeta.Mode != "adapter-bmps" {
			return errors.New("adapter belief prohibited in learned neural mode")
		}
		for i := 0; i < c.NeuralMeta.Classes; i++ {
			if x.Features[c.NeuralMeta.HypothesisOffset+i] != 0 {
				return errors.New("hidden target hypothesis cannot enter observed input")
			}
		}
		if x.Features[c.NeuralMeta.PlanMarker] != 0 {
			return errors.New("hypothetical plan cannot masquerade as observation")
		}
		return nil
	}
	if x.NeuralMeta != nil {
		return errors.New("V2 input cannot enter V1")
	}
	if x.Belief != nil {
		if err := x.Belief.Validate(); err != nil {
			return err
		}
		if len(x.Belief.Computations)+len(x.Belief.Utilities[0]) != c.Actions {
			return errors.New("BMPS action contract disagrees with neural actions")
		}
	}
	return nil
}

type CognitiveReadout struct {
	TerminalCompetence     [][]float64       `json:"terminal_competence,omitempty"`
	TerminalReward         [][]float64       `json:"terminal_reward,omitempty"`
	V1DecodedObservation   []float64         `json:"v1_decoded_observation,omitempty"`
	ClassBelief            []float64         `json:"class_belief,omitempty"`
	JointBelief            []float64         `json:"joint_belief,omitempty"`
	NativeLikelihood       []MetaComputation `json:"native_likelihood,omitempty"`
	AssessmentMeaning      string            `json:"assessment_meaning,omitempty"`
	Action                 int               `json:"action"`
	Policy                 []float64         `json:"policy"`
	Success                []float64         `json:"success"`
	RegulatedSuccess       []float64         `json:"regulated_success,omitempty"`
	Value                  float64           `json:"value"`
	Hidden                 []float64         `json:"hidden"`
	ParametersHash         string            `json:"parameters_hash"`
	BMPS                   *BMPSDecision     `json:"bmps,omitempty"`
	RegulationDisplacement float64           `json:"regulation_displacement"`
}

func sigmoid(x float64) float64 {
	if x >= 0 {
		return 1 / (1 + math.Exp(-x))
	}
	z := math.Exp(x)
	return z / (1 + z)
}
func mat(w []float64, t CognitiveTensor, x []float64) []float64 {
	y := make([]float64, t.Rows)
	for i := range y {
		for j, v := range x {
			y[i] += w[t.Offset+i*t.Cols+j] * v
		}
	}
	return y
}
func addBias(y, w []float64, t CognitiveTensor) {
	for i := range y {
		y[i] += w[t.Offset+i]
	}
}

type gruFrame struct{ x, hp, r, z, n, u, h []float64 }

func cognitiveGRU(c CognitiveModelConfig, l CognitiveLayout, w, x, hp []float64) gruFrame {
	a := mat(w, l.WI, x)
	addBias(a, w, l.BI)
	u := mat(w, l.WH, hp)
	addBias(u, w, l.BH)
	f := gruFrame{x: append([]float64{}, x...), hp: append([]float64{}, hp...), r: make([]float64, c.Hidden), z: make([]float64, c.Hidden), n: make([]float64, c.Hidden), u: make([]float64, c.Hidden), h: make([]float64, c.Hidden)}
	for i := range f.h {
		f.r[i] = sigmoid(a[i] + u[i])
		f.z[i] = sigmoid(a[c.Hidden+i] + u[c.Hidden+i])
		f.u[i] = u[2*c.Hidden+i]
		f.n[i] = math.Tanh(a[2*c.Hidden+i] + f.r[i]*f.u[i])
		f.h[i] = (1-f.z[i])*f.n[i] + f.z[i]*hp[i]
	}
	return f
}
func cognitiveHeads(l CognitiveLayout, w, h []float64, mask []bool) (p, sa []float64, value float64) {
	p = mat(w, l.Actor, h)
	addBias(p, w, l.ActorBias)
	m := math.Inf(-1)
	for i := range p {
		if mask[i] {
			m = math.Max(m, p[i])
		} else {
			p[i] = math.Inf(-1)
		}
	}
	z := 0.0
	for i := range p {
		if mask[i] {
			p[i] = math.Exp(p[i] - m)
			z += p[i]
		} else {
			p[i] = 0
		}
	}
	for i := range p {
		p[i] /= z
	}
	sa = mat(w, l.Assessment, h)
	addBias(sa, w, l.AssessmentBias)
	for i := range sa {
		sa[i] = sigmoid(sa[i])
	}
	v := mat(w, l.Critic, h)
	addBias(v, w, l.CriticBias)
	return p, sa, v[0]
}
func cognitiveDecode(l CognitiveLayout, w, h []float64, action, actions int) []float64 {
	x := append(append([]float64{}, h...), make([]float64, actions)...)
	x[len(h)+action] = 1
	y := mat(w, l.Decoder, x)
	addBias(y, w, l.DecoderBias)
	for i := range y {
		y[i] = math.Tanh(y[i])
	}
	return y
}
func samplePolicy(p []float64, u float64) int {
	s := 0.0
	last := 0
	for i, v := range p {
		if v > 0 {
			last = i
		}
		s += v
		if u < s {
			return i
		}
	}
	return last
}
func maxIndex(x []float64) int {
	j := 0
	for i := 1; i < len(x); i++ {
		if x[i] > x[j] {
			j = i
		}
	}
	return j
}

// Preview does not mutate recurrent state or weights. Advance commits the same
// calculation to canonical N activation; the runtime journals its input first.
func (k *KnowledgeBase) CognitivePreview(x CognitiveInput) (CognitiveReadout, error) {
	k.RLock()
	defer k.RUnlock()
	return k.cognitiveForward(x)
}
func (k *KnowledgeBase) cognitiveForward(x CognitiveInput) (CognitiveReadout, error) {
	s := k.BrainState.CognitiveModel
	if s == nil {
		return CognitiveReadout{}, errors.New("cognitive model not initialized")
	}
	c, l := s.Config, s.Layout
	if err := x.Validate(c); err != nil {
		return CognitiveReadout{}, err
	}
	w, err := k.cognitiveWeights(s)
	if err != nil {
		return CognitiveReadout{}, err
	}
	if c.Version == CognitiveModelV2 {
		return k.cognitiveForwardV2(x, s, w)
	}
	hp := k.cognitiveHidden(s)
	if x.Reset || x.NoRecurrence {
		hp = make([]float64, c.Hidden)
	}
	h := cognitiveGRU(c, l, w, x.Features, hp).h
	p, sa, v := cognitiveHeads(l, w, h, x.Mask)
	out := CognitiveReadout{Policy: p, Success: sa, Value: v, Hidden: h, ParametersHash: CognitiveDigest(w)}
	if x.Regulate {
		// MUSE option c adapted to a deterministic GRU world model, NOT a Dreamer
		// RSSM reproduction. Imagined decoder outputs remain hypothetical; they
		// never become verified feedback or alter the observed continuation h.
		hs := append([]float64{}, h...)
		for step := 0; step < c.RegulationSteps; step++ {
			pp, ss, _ := cognitiveHeads(l, w, hs, x.Mask)
			a := maxIndex(pp)
			xx := cognitiveDecode(l, w, hs, a, c.Actions)
			if x.V1RegulationAblation != "" {
				out.V1DecodedObservation = xx
			}
			if x.V1RegulationAblation != "gradient-only" {
				hs = cognitiveGRU(c, l, w, xx, hs).h
			}
			_, ss, _ = cognitiveHeads(l, w, hs, x.Mask)
			for j := range hs {
				if x.V1RegulationAblation == "decoder-only" {
					continue
				}
				d := 0.0
				for i, q := range ss {
					d += q * (1 - q) * w[l.Assessment.Offset+i*c.Hidden+j]
				}
				hs[j] = math.Max(-1, math.Min(1, hs[j]+c.RegulationRate*d))
			}
		}
		p, out.RegulatedSuccess, _ = cognitiveHeads(l, w, hs, x.Mask)
		out.Policy = p
		for i := range h {
			out.RegulationDisplacement += math.Pow(hs[i]-h[i], 2)
		}
		out.RegulationDisplacement = math.Sqrt(out.RegulationDisplacement)
	}
	out.Action = samplePolicy(out.Policy, x.Draw)
	if x.Belief != nil {
		b := *x.Belief
		// Optional model: task utility is realized only if the agent can execute
		// its proposed plan. This uses a learned probability, not a confidence
		// threshold. It is an explicit integration assumption, evaluated by ablation.
		if x.AssessUtility {
			q := sa[len(sa)-1]
			if len(out.RegulatedSuccess) > 0 {
				q = out.RegulatedSuccess[4]
			}
			b.Utilities = make([][]float64, len(b.Probabilities))
			for i, row := range x.Belief.Utilities {
				b.Utilities[i] = make([]float64, len(row))
				for j, u := range row {
					b.Utilities[i][j] = q * u
				}
			}
		}
		weights := k.cognitiveBMPSWeights(s)
		d, err := SelectBMPS(b, weights)
		if err != nil {
			return CognitiveReadout{}, err
		}
		out.BMPS = &d
		a := d.Computation
		if a < 0 {
			a = len(b.Computations) + d.TerminalAction
		}
		if !x.Mask[a] {
			return CognitiveReadout{}, errors.New("BMPS selected an operation disallowed by the adapter; model/mask mismatch")
		}
		out.Action = a
	}
	return out, nil
}
func (k *KnowledgeBase) AdvanceCognitiveModel(x CognitiveInput, at time.Time) (CognitiveReadout, error) {
	k.Lock()
	defer k.Unlock()
	out, err := k.cognitiveForward(x)
	if err != nil {
		return out, err
	}
	s := k.BrainState.CognitiveModel
	k.setCognitiveHidden(s, out.Hidden, at)
	for i, id := range s.Layout.Inputs {
		n := k.Registry.GetByID(id)
		value := x.Features[i]
		if s.NeuralMeta != nil && x.NeuralMeta.NoFeedback {
			m := s.Config.NeuralMeta
			if (i >= m.ResultOffset && i < m.ResultOffset+m.Outcomes) || (i >= m.OperationOffset && i < m.OperationOffset+m.Computations) {
				value = 0
			}
		}
		n.Activation = (value + 1) / 2
		n.LastActivation = at
	}
	if s.NeuralMeta != nil {
		for i, id := range s.NeuralMeta.JointNodes {
			n := k.Registry.GetByID(id)
			n.Activation = out.JointBelief[i]
			n.LastActivation = at
		}
		for i, id := range s.Layout.Success {
			n := k.Registry.GetByID(id)
			n.Activation = out.Success[i]
			n.LastActivation = at
		}
		for i, id := range s.Layout.Actions {
			n := k.Registry.GetByID(id)
			n.Activation = out.Policy[i]
			n.LastActivation = at
		}
	}
	s.ModelSteps++
	return out, nil
}
