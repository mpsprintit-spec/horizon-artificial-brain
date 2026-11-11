package knowledge

// Native control reads the current distributed activity through ordinary
// canonical synapses. It contains no V5 model, hidden GRU, world predictor or
// problem-class routing. Metadata names compute primitives, never skills.
import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"math/rand"
	"sort"
	"time"
)

const NativeControlVersion = "native-operation-control-v1"

func NativeDigest(v any) string {
	b, e := json.Marshal(v)
	if e != nil {
		return ""
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// Nil and empty activity are the same state across omitempty checkpoints.
func NativeStateDigest(state map[NodeID]float64) string {
	if len(state) == 0 {
		return NativeDigest(map[NodeID]float64{})
	}
	return NativeDigest(state)
}

const nativeReadoutScale = 8.0

type NativeOperation struct {
	Kind string  `json:"kind"` // propagate, focus, relax, stop
	Gain float64 `json:"gain"`
	Cost float64 `json:"cost"` // declared compute tariff, not wall-clock latency
}

func (o NativeOperation) Validate() error {
	if o.Kind != "propagate" && o.Kind != "focus" && o.Kind != "relax" && o.Kind != "stop" {
		return errors.New("unknown native operation")
	}
	if !nativeFinite(o.Gain) || o.Gain < 0 || o.Gain > 1 || !nativeFinite(o.Cost) || o.Cost < 0 || o.Cost > 1 {
		return errors.New("invalid native operation gain/cost")
	}
	if o.Kind == "stop" && (o.Cost != 0 || o.Gain != 0) {
		return errors.New("stop must have zero gain/cost")
	}
	return nil
}

type NativeControlConfig struct {
	MaxActiveNodes  int               `json:"max_active_nodes"`
	MaxSynapseReads int               `json:"max_synapse_reads"`
	Version         string            `json:"version"`
	Sources         []NodeID          `json:"sources"`
	Operations      []NativeOperation `json:"operations"`
	Seed            int64             `json:"seed"`
	LearningRate    float64           `json:"learning_rate"`
	Gamma           float64           `json:"gamma"`
	Lambda          float64           `json:"lambda"`
	Temperature     float64           `json:"temperature"`
	MaxSteps        int               `json:"max_steps"`
}

func nativeFinite(x float64) bool { return !math.IsNaN(x) && !math.IsInf(x, 0) }
func (c NativeControlConfig) Validate() error {
	if c.MaxActiveNodes < 1 || c.MaxActiveNodes > 1000000 || c.MaxSynapseReads < 1 || c.MaxSynapseReads > 10000000 {
		return errors.New("invalid native work budget")
	}
	if c.Version != NativeControlVersion || len(c.Sources) < 1 || len(c.Sources) > 4096 || len(c.Operations) < 2 || len(c.Operations) > 16 || c.MaxSteps < 1 || c.MaxSteps > 64 {
		return errors.New("invalid native control dimensions/version")
	}
	for _, v := range []float64{c.LearningRate, c.Gamma, c.Lambda, c.Temperature} {
		if !nativeFinite(v) {
			return errors.New("nonfinite native control configuration")
		}
	}
	if c.LearningRate <= 0 || c.LearningRate > 1 || c.Gamma < 0 || c.Gamma > 1 || c.Lambda < 0 || c.Lambda > 1 || c.Temperature <= 0 || c.Temperature > 10 {
		return errors.New("invalid native learning bounds")
	}
	seen := map[NodeID]bool{}
	for _, id := range c.Sources {
		if id <= 0 || seen[id] {
			return errors.New("duplicate/invalid native source")
		}
		seen[id] = true
	}
	stops := 0
	for _, o := range c.Operations {
		if e := o.Validate(); e != nil {
			return e
		}
		if o.Kind == "stop" {
			stops++
		}
	}
	if stops != 1 {
		return errors.New("native control requires one stop affordance")
	}
	return nil
}

// These are only references into N plus bounded transient experience. No
// learned weight is stored here: every value is read from canonical synapses.
type NativeControlState struct {
	Config  NativeControlConfig `json:"config"`
	Bias    NodeID              `json:"bias"`
	Outputs [][]NodeID          `json:"outputs"`
	Updates uint64              `json:"updates"`
	Episode *NativeEpisode      `json:"episode,omitempty"`
}
type NativeDecision struct {
	State         map[NodeID]float64 `json:"state"`
	Values        []float64          `json:"values"`
	Probabilities []float64          `json:"probabilities"`
	Action        int                `json:"action"`
	Cost          float64            `json:"cost"`
	Before        string             `json:"before"`
	After         string             `json:"after"`
	At            time.Time          `json:"at"`
}
type NativeEpisode struct {
	ID        string           `json:"id"`
	Training  bool             `json:"training"`
	Started   time.Time        `json:"started"`
	Decisions []NativeDecision `json:"decisions,omitempty"`
	Stopped   bool             `json:"stopped"`
	Truncated bool             `json:"truncated"`
}
type NativeReadout struct {
	Values        []float64 `json:"values"`
	Probabilities []float64 `json:"probabilities"`
	Action        int       `json:"action"`
}
type NativeLearning struct {
	Return          float64 `json:"return"`
	UpdatedSynapses int     `json:"updated_synapses"`
	MaxDelta        float64 `json:"max_delta"`
	TDError         float64 `json:"td_error"`
}

func (k *KnowledgeBase) HasNativeControl() bool {
	k.RLock()
	defer k.RUnlock()
	return k.BrainState.NativeControl != nil
}
func (k *KnowledgeBase) NativeControlSnapshot() *NativeControlState {
	k.RLock()
	defer k.RUnlock()
	if k.BrainState.NativeControl == nil {
		return nil
	}
	b, _ := json.Marshal(k.BrainState.NativeControl)
	var out NativeControlState
	_ = json.Unmarshal(b, &out)
	return &out
}
func (k *KnowledgeBase) ValidateNativeInit(c NativeControlConfig) error {
	if err := c.Validate(); err != nil {
		return err
	}
	k.RLock()
	defer k.RUnlock()
	if k.BrainState.NativeControl != nil || k.BrainState.CognitiveModel != nil {
		return errors.New("native control requires an unconfigured native Brain; V5 remains isolated")
	}
	for _, id := range c.Sources {
		if k.Registry.GetByID(id) == nil {
			return errors.New("native source missing from canonical N")
		}
	}
	return nil
}
func (k *KnowledgeBase) InitializeNativeControl(c NativeControlConfig, at time.Time) error {
	if at.IsZero() {
		return errors.New("native initialization needs time")
	}
	if e := k.ValidateNativeInit(c); e != nil {
		return e
	}
	// Caller owns the runtime; the Brain lock serializes persistence/inspection.
	k.Lock()
	defer k.Unlock()
	if k.BrainState.NativeControl != nil || k.BrainState.CognitiveModel != nil {
		return errors.New("native control already initialized")
	}
	c.Sources = append([]NodeID(nil), c.Sources...)
	sort.Slice(c.Sources, func(i, j int) bool { return c.Sources[i] < c.Sources[j] })
	c.Operations = append([]NativeOperation(nil), c.Operations...)
	s := &NativeControlState{Config: c}
	coordinate := 0
	alloc := func() NodeID {
		coordinate++
		n := k.createRepresentationPrototypeAt(NewNeuralVector([]float64{math.Sin(float64(coordinate)), math.Cos(float64(coordinate)), .37}), at)
		n.Importance = 1
		return n.ID
	}
	s.Bias = alloc()
	s.Outputs = make([][]NodeID, len(c.Operations))
	rng := rand.New(rand.NewSource(c.Seed))
	for a := range s.Outputs {
		for j := 0; j < 4; j++ {
			id := alloc()
			s.Outputs[a] = append(s.Outputs[a], id)
			for _, from := range append(append([]NodeID(nil), c.Sources...), s.Bias) {
				w := (rng.Float64() - .5) * .02
				syn := &Synapse{TargetID: id, Weight: math.Abs(w), Confidence: 1, Inhibitory: w < 0, Dynamic: DynamicState{Weight: math.Abs(w), Confidence: 1, LastModification: at}}
				n := k.Registry.GetByID(from)
				n.Synapses[id] = SynapseList{syn}
			}
		}
	}
	k.BrainState.NativeControl = s
	return nil
}
func nativeIDs(state map[NodeID]float64) []NodeID {
	ids := make([]NodeID, 0, len(state))
	for id := range state {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}
func nativeWeight(s *Synapse) float64 {
	w := s.Weight * nativeReadoutScale
	if s.Inhibitory {
		return -w
	}
	return w
}
func (k *KnowledgeBase) nativeFeatures(s *NativeControlState, state map[NodeID]float64) map[NodeID]float64 {
	x := make(map[NodeID]float64)
	x[s.Bias] = 1
	for _, id := range s.Config.Sources {
		if v := state[id]; v != 0 {
			x[id] = v
		}
	}
	return x
}
func (k *KnowledgeBase) nativeReadout(s *NativeControlState, state map[NodeID]float64, greedy bool, u float64) (NativeReadout, error) {
	out := NativeReadout{Values: make([]float64, len(s.Outputs)), Probabilities: make([]float64, len(s.Outputs))}
	for id, v := range state {
		if k.Registry.GetByID(id) == nil || !nativeFinite(v) || v < 0 || v > 2 {
			return out, errors.New("invalid native state")
		}
	}
	x := k.nativeFeatures(s, state)
	ids := nativeIDs(x)
	for a, targets := range s.Outputs {
		for _, id := range ids {
			n := k.Registry.GetByID(id)
			for _, target := range targets {
				list := n.Synapses[target]
				if len(list) != 1 || list[0] == nil {
					return out, errors.New("native parameter missing")
				}
				out.Values[a] += x[id] * nativeWeight(list[0]) / float64(len(targets))
			}
		}
	}
	best := 0
	for a := range out.Values {
		if out.Values[a] > out.Values[best] {
			best = a
		}
	}
	sum := 0.0
	for a, q := range out.Values {
		out.Probabilities[a] = math.Exp((q - out.Values[best]) / s.Config.Temperature)
		sum += out.Probabilities[a]
	}
	for a := range out.Values {
		out.Probabilities[a] /= sum
	}
	out.Action = best
	if !greedy {
		if !nativeFinite(u) || u < 0 || u >= 1 {
			return out, errors.New("sample must be in [0,1)")
		}
		out.Action = len(out.Values) - 1
		for a, p := range out.Probabilities {
			u -= p
			if u < 0 {
				out.Action = a
				break
			}
		}
	}
	return out, nil
}
func (k *KnowledgeBase) NativePolicy(state map[NodeID]float64, greedy bool, u float64) (NativeReadout, error) {
	k.RLock()
	defer k.RUnlock()
	s := k.BrainState.NativeControl
	if s == nil {
		return NativeReadout{}, errors.New("native control not initialized")
	}
	return k.nativeReadout(s, state, greedy, u)
}
func (k *KnowledgeBase) BeginNativeEpisode(id string, training bool, at time.Time) error {
	k.Lock()
	defer k.Unlock()
	s := k.BrainState.NativeControl
	if s == nil || s.Episode != nil || id == "" || at.IsZero() {
		return errors.New("invalid native episode boundary")
	}
	s.Episode = &NativeEpisode{ID: id, Training: training, Started: at}
	return nil
}
func (k *KnowledgeBase) AppendNativeDecision(d NativeDecision) error {
	k.Lock()
	defer k.Unlock()
	s := k.BrainState.NativeControl
	if s == nil || s.Episode == nil || s.Episode.Stopped || len(s.Episode.Decisions) >= s.Config.MaxSteps {
		return errors.New("native episode not open")
	}
	ep := s.Episode
	ep.Decisions = append(ep.Decisions, d)
	ep.Stopped = s.Config.Operations[d.Action].Kind == "stop"
	if len(ep.Decisions) == s.Config.MaxSteps && !ep.Stopped {
		ep.Stopped = true
		ep.Truncated = true
	}
	return nil
}

// LearnNativeOutcome is offline on-policy Sarsa(lambda) with frozen episode
// values (forward lambda returns). lambda=1 is Monte Carlo; lambda=0 TD(0).
// The active eligibility index below contains canonical addresses, not weights.
// Only executed actions and their nonzero source features get credit. This is
// temporal statistical credit, not a claim of identified causal world structure.
func (k *KnowledgeBase) LearnNativeOutcome(reward float64, learn bool, at time.Time) (NativeLearning, error) {
	k.Lock()
	defer k.Unlock()
	s := k.BrainState.NativeControl
	out := NativeLearning{}
	if s == nil || s.Episode == nil || !s.Episode.Stopped || len(s.Episode.Decisions) == 0 || !nativeFinite(reward) || reward < -1 || reward > 1 {
		return out, errors.New("native outcome has no completed episode")
	}
	ep := s.Episode
	out.Return = reward
	for _, d := range ep.Decisions {
		out.Return -= d.Cost
	}
	if !learn || !ep.Training {
		s.Episode = nil
		return out, nil
	}
	type edge struct{ from, to NodeID }
	grad := map[edge]float64{}
	g := reward
	for t := len(ep.Decisions) - 1; t >= 0; t-- {
		d := ep.Decisions[t]
		if t == len(ep.Decisions)-1 {
			g = reward - d.Cost
		} else {
			next := ep.Decisions[t+1]
			g = -d.Cost + s.Config.Gamma*((1-s.Config.Lambda)*next.Values[next.Action]+s.Config.Lambda*g)
		}
		delta := g - d.Values[d.Action]
		out.TDError += math.Abs(delta)
		x := k.nativeFeatures(s, d.State)
		norm := 0.0
		for _, id := range nativeIDs(x) {
			norm += x[id] * x[id]
		}
		if norm < 1 {
			norm = 1
		}
		for _, id := range nativeIDs(x) {
			for _, to := range s.Outputs[d.Action] {
				grad[edge{id, to}] += delta * x[id] / norm
			}
		}
	}
	keys := make([]edge, 0, len(grad))
	for key := range grad {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].from == keys[j].from {
			return keys[i].to < keys[j].to
		}
		return keys[i].from < keys[j].from
	})
	for _, key := range keys {
		syn := k.Registry.GetByID(key.from).Synapses[key.to][0]
		old := nativeWeight(syn)
		next := math.Max(-7.5, math.Min(7.5, old+s.Config.LearningRate*grad[key]))
		if next == old {
			continue
		}
		syn.Weight = math.Abs(next) / nativeReadoutScale
		syn.Inhibitory = next < 0
		syn.Dynamic.Weight = syn.Weight
		syn.Dynamic.LastModification = at
		syn.Dynamic.Frequency++
		out.UpdatedSynapses++
		out.MaxDelta = math.Max(out.MaxDelta, math.Abs(next-old))
	}
	s.Updates++
	s.Episode = nil
	return out, nil
}
