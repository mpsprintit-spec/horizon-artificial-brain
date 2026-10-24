package knowledge

// This bounded numerical readout composes learned synaptic contributions.
// Its coordinates describe position and physical I/O, never word classes or
// object names. All trainable scalars live in ordinary Brain synapses. Arrays
// below are detached working values, rebuilt from those synapses on each call.
import (
	"errors"
	"fmt"
	"math"
	"slices"
	"time"
)

const (
	NeuralCompositionV1   = "synaptic-composition-v1"
	MaxCompositionWords   = 6
	MaxCompositionPatches = 7
	compositionHidden     = 12
	compositionOutputs    = 6
	compositionBanks      = 28
	compositionScale      = 8.0
)

var compositionScope = ProjectionScope{Modality: "internal-numeric", Field: NeuralCompositionV1}

type CompositionConfig struct {
	Version         string `json:"version"`
	Seed            uint64 `json:"seed"`
	Initialization  string `json:"initialization"`
	MaxLearnedWords int    `json:"max_learned_words"`
	MaxSceneAgeNS   int64  `json:"max_scene_age_ns"`
}

func DefaultCompositionConfig() CompositionConfig {
	return CompositionConfig{NeuralCompositionV1, 7, "separated", 256, int64(2 * time.Second)}
}

func (c CompositionConfig) Validate() error {
	if c.Version != NeuralCompositionV1 || c.Initialization != "separated" && c.Initialization != "zero" && c.Initialization != "random" || c.MaxLearnedWords < 1 || c.MaxLearnedWords > 4096 || c.MaxSceneAgeNS <= 0 || c.MaxSceneAgeNS > int64(time.Minute) {
		return errors.New("unsupported composition version, initialization or resource budget")
	}
	return nil
}

// Patch values are three measured appearance channels, physical state and a
// signal channel. Segmentation is supplied by the simulator, not learned here.
type CompositionPatch struct {
	Values []float64 `json:"values"`
	XY     []float64 `json:"xy"`
}

type CompositionScene struct {
	Patches    []CompositionPatch `json:"patches"`
	Point      []float64          `json:"point"`
	ObservedAt time.Time          `json:"observed_at"`
}

type CompositionInput struct {
	Words [][]float64      `json:"words"`
	Scene CompositionScene `json:"scene"`
}

type CompositionExposure struct {
	ID                string           `json:"id"`
	Source            string           `json:"source"`
	EpisodeID         string           `json:"episode_id"`
	IndependenceGroup string           `json:"independence_group"`
	CausalGroup       string           `json:"causal_group"`
	Reliability       float64          `json:"reliability"`
	Input             CompositionInput `json:"input"`
	Motor             []float64        `json:"motor"`
	After             CompositionScene `json:"after"`
}

func compositionNumbers(v []float64, n int) bool {
	if len(v) != n {
		return false
	}
	for _, x := range v {
		if math.IsNaN(x) || math.IsInf(x, 0) || x < -1 || x > 1 {
			return false
		}
	}
	return true
}

func (s CompositionScene) Validate() error {
	if s.ObservedAt.IsZero() || len(s.Patches) == 0 || len(s.Patches) > MaxCompositionPatches || !compositionNumbers(s.Point, 3) || s.Point[2] != -1 && s.Point[2] != 1 {
		return errors.New("composition scene requires bounded patches, event time and an explicit pointing presence")
	}
	for i, p := range s.Patches {
		if !compositionNumbers(p.Values, 5) || !compositionNumbers(p.XY, 2) || math.Abs(p.Values[3]) != 1 {
			return errors.New("invalid composition measurement coordinates or physical state")
		}
		for j := 0; j < i; j++ {
			if slices.Equal(p.XY, s.Patches[j].XY) {
				return errors.New("composition patches have duplicate spatial coordinates")
			}
		}
	}
	return nil
}

func (i CompositionInput) Validate() error {
	if err := ValidateCompositionWords(i.Words); err != nil {
		return err
	}
	return i.Scene.Validate()
}

func ValidateCompositionWords(words [][]float64) error {
	if len(words) > MaxCompositionWords {
		return errors.New("composition position window exceeded")
	}
	for _, v := range words {
		if !compositionNumbers(v, 16) {
			return errors.New("composition requires normalized 16D written observations")
		}
	}
	return nil
}

// Teaching supervision is derived solely from numerical contact and measured
// after-state. Future state and teaching motor are never available to readout.
func (e CompositionExposure) target() (int, float64, float64, error) {
	if !compositionNumbers(e.Motor, 4) || e.Motor[3] != 0 && e.Motor[3] != 1 {
		return -1, 0, 0, errors.New("invalid composition motor demonstration")
	}
	if e.Motor[3] == 0 {
		if !slices.Equal(e.Motor, []float64{0, 0, 0, 0}) {
			return -1, 0, 0, errors.New("no-op demonstration must have zero force and coordinates")
		}
		for _, p := range e.Input.Scene.Patches {
			found := false
			for _, q := range e.After.Patches {
				if slices.Equal(p.XY, q.XY) {
					found = true
					if !slices.Equal(p.Values, q.Values) {
						return -1, 0, 0, errors.New("no-op demonstration changed a measured patch")
					}
				}
			}
			if !found {
				return -1, 0, 0, errors.New("after-state omitted a patch")
			}
		}
		return -1, 0, 0, nil
	}
	if math.Abs(e.Motor[2]) != 1 {
		return -1, 0, 0, errors.New("unsupported composition setpoint")
	}
	for i, p := range e.Input.Scene.Patches {
		if p.XY[0] == e.Motor[0] && p.XY[1] == e.Motor[1] {
			for _, q := range e.After.Patches {
				if slices.Equal(p.XY, q.XY) && slices.Equal(p.Values[:3], q.Values[:3]) {
					return i, q.Values[3], 1, nil
				}
			}
		}
	}
	return -1, 0, 0, errors.New("motor contact has no corresponding before/after measurement")
}

func (e CompositionExposure) Validate() error {
	if e.ID == "" || len(e.ID) > 256 || e.Source == "" || len(e.Source) > 256 || e.EpisodeID == "" || len(e.EpisodeID) > 256 || e.IndependenceGroup == "" || len(e.IndependenceGroup) > 256 || e.CausalGroup == "" || len(e.CausalGroup) > 256 || math.IsNaN(e.Reliability) || e.Reliability <= 0 || e.Reliability > 1 {
		return errors.New("composition exposure requires bounded causal provenance and reliability in (0,1]")
	}
	if err := e.Input.Validate(); err != nil {
		return err
	}
	if err := e.After.Validate(); err != nil {
		return err
	}
	if e.After.ObservedAt.Before(e.Input.Scene.ObservedAt) || len(e.Input.Scene.Patches) != len(e.After.Patches) {
		return errors.New("composition before/after observations disagree")
	}
	_, _, _, err := e.target()
	return err
}

type CompositionReadout struct {
	Status          string    `json:"status"`
	Accepted        bool      `json:"accepted"`
	Goal            []float64 `json:"goal,omitempty"`
	Motor           []float64 `json:"motor,omitempty"`
	Confidence      float64   `json:"confidence"` // A bounded score, not a calibrated probability.
	ReferenceMargin float64   `json:"reference_margin"`
	LocalEffect     float64   `json:"predicted_local_effect"`
	SceneAt         time.Time `json:"scene_at"`
	Sequence        uint64    `json:"sequence"`
	BrainIdentity   string    `json:"brain_identity"`
	Coordinates     []float64 `json:"coordinates,omitempty"` // Current internal numerical state, never an authorization.
}

type compositionWeights struct {
	E [][]float64
	H [compositionOutputs][compositionHidden]float64
	P [compositionOutputs][MaxCompositionWords]float64
	B [compositionOutputs]float64
}

func compositionRandom(seed uint64, a, b int) float64 {
	v := seed ^ uint64(a+1)*0x9e3779b97f4a7c15 ^ uint64(b+17)*0xbf58476d1ce4e5b9
	v ^= v >> 30
	v *= 0xbf58476d1ce4e5b9
	v ^= v >> 27
	v *= 0x94d049bb133111eb
	v ^= v >> 31
	return 2*float64(v>>11)/float64(uint64(1)<<53) - 1
}

// Caller holds k.mu. Coordinates alone locate the shared numerical cells.
func (k *KnowledgeBase) compositionBankLocked() ([compositionBanks][]NodeID, error) {
	var bank [compositionBanks][]NodeID
	k.projectionMu.RLock()
	defer k.projectionMu.RUnlock()
	seen := map[NodeID]bool{}
	for _, p := range k.ProjectionPopulations {
		if p.Scope == nil || *p.Scope != compositionScope {
			continue
		}
		if len(p.Prototype) != 1 || len(p.Units) != 4 {
			return bank, errors.New("invalid composition coordinate bank")
		}
		x := p.Prototype[0] * 64
		if math.IsNaN(x) || x < 0 || x >= compositionBanks || math.Trunc(x) != x || len(bank[int(x)]) != 0 {
			return bank, errors.New("invalid or repeated composition coordinate")
		}
		for _, u := range p.Units {
			if seen[u.NodeID] || k.Registry.GetByID(u.NodeID) == nil {
				return bank, errors.New("invalid composition member")
			}
			seen[u.NodeID] = true
			bank[int(x)] = append(bank[int(x)], u.NodeID)
		}
	}
	return bank, nil
}

func (k *KnowledgeBase) HasNeuralComposition() bool {
	if k == nil {
		return false
	}
	k.projectionMu.RLock()
	defer k.projectionMu.RUnlock()
	for _, p := range k.ProjectionPopulations {
		if p.Scope != nil && *p.Scope == compositionScope {
			return true
		}
	}
	return false
}

func (k *KnowledgeBase) compositionParameter(from, to []NodeID) (float64, bool) {
	k.Registry.mu.RLock()
	defer k.Registry.mu.RUnlock()
	sum := 0.0
	count := 0
	for _, a := range from {
		n := k.Registry.byID[a]
		if n == nil {
			continue
		}
		for _, b := range to {
			list := n.Synapses[b]
			exc, inh := list.FindDynamic(false), list.FindDynamic(true)
			if exc == nil || inh == nil {
				continue
			}
			sum += compositionScale * (exc.Dynamic.Weight - inh.Dynamic.Weight)
			count++
		}
	}
	if count == 0 {
		return 0, false
	}
	return sum / float64(count), true
}

// Apply the bounded error-driven delta without discarding temporal history.
// Confidence is practice support capped below certainty; it is never promoted
// into independent causal evidence. This only touches learned coordinate links.
func (k *KnowledgeBase) adjustCompositionParameter(from, to []NodeID, value, reliability float64, at time.Time, initial bool) {
	value = max(-7.99, min(7.99, value))
	positive, negative := max(0, value)/compositionScale, max(0, -value)/compositionScale
	k.Registry.mu.Lock()
	defer k.Registry.mu.Unlock()
	for _, a := range from {
		n := k.Registry.byID[a]
		if n.Synapses == nil {
			n.Synapses = map[NodeID]SynapseList{}
		}
		for _, b := range to {
			list := n.Synapses[b]
			for _, inhibitory := range []bool{false, true} {
				s := list.FindDynamic(inhibitory)
				if s == nil {
					s = &Synapse{TargetID: b, Inhibitory: inhibitory}
					list = append(list, s)
				}
				target := positive
				if inhibitory {
					target = negative
				}
				delta := max(-.1, min(.1, target-s.Dynamic.Weight))
				if initial {
					s.Dynamic.Weight = target
					s.Dynamic.Confidence = .01
				} else {
					s.Dynamic.Weight = clamp01(s.Dynamic.Weight + delta)
					s.Dynamic.Eligibility = clamp01(.9*s.Dynamic.Eligibility + math.Abs(delta))
					s.Dynamic.Confidence = min(.95, s.Dynamic.Confidence+.02*reliability*(1-s.Dynamic.Confidence))
					s.Dynamic.Frequency++
					s.Dynamic.LastActivation = at
					s.Dynamic.LastEligibilityUpdate = at
				}
				s.Dynamic.LastModification = at
				syncSynapseLegacyState(s)
			}
			n.Synapses[b] = list
		}
	}
}

func (k *KnowledgeBase) InitializeCompositionAt(c CompositionConfig, at time.Time) error {
	if err := c.Validate(); err != nil {
		return err
	}
	if at.IsZero() {
		return errors.New("composition requires event time")
	}
	k.Lock()
	defer k.Unlock()
	bank, err := k.compositionBankLocked()
	if err != nil {
		return err
	}
	for _, ids := range bank {
		if len(ids) > 0 {
			return errors.New("composition is already initialized")
		}
	}
	for i := range bank {
		v := NewNeuralVector([]float64{float64(i) / 64})
		p := ProjectionPopulation{Scope: cloneProjectionScope(&compositionScope), Prototype: v.Values, Provenance: BootstrapProvenance{Origin: "numeric_coordinate_initialization", Status: "untrained_coordinates"}}
		for j := 0; j < 4; j++ {
			n := k.createRepresentationPrototypeAt(projectionPrototype(v, j, 4), at)
			bank[i] = append(bank[i], n.ID)
			p.Units = append(p.Units, PopulationUnit{NodeID: n.ID, Activation: 1})
		}
		k.registerProjectionPopulation(p)
	}
	norm := 0.0
	for out := 0; out < compositionOutputs; out++ {
		for h := 0; h < compositionHidden; h++ {
			v := compositionRandom(c.Seed, out, h)
			norm += v * v
		}
	}
	for out := 0; out < compositionOutputs; out++ {
		for h := 0; h < compositionHidden; h++ {
			v := 0.0
			if c.Initialization == "separated" && out == h {
				v = 1
			}
			if c.Initialization == "random" {
				v = compositionRandom(c.Seed, out, h) * math.Sqrt(6/norm)
			}
			k.adjustCompositionParameter(bank[h], bank[12+out], v, 0, at, true)
		}
		for pos := 0; pos < MaxCompositionWords; pos++ {
			k.adjustCompositionParameter(bank[18+pos], bank[12+out], 1, 0, at, true)
		}
		k.adjustCompositionParameter(bank[24], bank[12+out], 0, 0, at, true)
	}
	k.adjustCompositionParameter(bank[25], bank[27], 0, 0, at, true)
	k.adjustCompositionParameter(bank[26], bank[27], 0, 0, at, true)
	return nil
}

func (k *KnowledgeBase) compositionWordsLocked(words [][]float64) ([][]NodeID, int) {
	ids := make([][]NodeID, len(words))
	learned := 0
	k.projectionMu.RLock()
	defer k.projectionMu.RUnlock()
	for _, p := range k.ProjectionPopulations {
		if p.Scope == nil || *p.Scope != writtenNumericScope {
			continue
		}
		for i, v := range words {
			if slices.Equal(v, p.Prototype) {
				ids[i] = writtenPopulationIDs(p)
			}
		}
		learned++
	}
	return ids, learned
}

// Validation is read-only and precedes the runtime's durable append.
func (k *KnowledgeBase) ValidateCompositionExposure(c CompositionConfig, e CompositionExposure) error {
	if err := c.Validate(); err != nil {
		return err
	}
	if err := e.Validate(); err != nil {
		return err
	}
	k.RLock()
	defer k.RUnlock()
	bank, err := k.compositionBankLocked()
	if err != nil {
		return err
	}
	if len(bank[0]) == 0 {
		return errors.New("composition is not initialized")
	}
	ids, _ := k.compositionWordsLocked(e.Input.Words)
	count := 0
	k.projectionMu.RLock()
	for _, p := range k.ProjectionPopulations {
		if p.Scope != nil && *p.Scope == writtenNumericScope {
			if _, ok := k.compositionParameter(writtenPopulationIDs(p), bank[0]); ok {
				count++
			}
		}
	}
	k.projectionMu.RUnlock()
	for i, p := range ids {
		_, ok := k.compositionParameter(p, bank[0])
		if ok {
			continue
		}
		duplicate := false
		for j := 0; j < i; j++ {
			if slices.Equal(e.Input.Words[i], e.Input.Words[j]) {
				duplicate = true
			}
		}
		if !duplicate {
			count++
		}
	}
	if count > c.MaxLearnedWords {
		return errors.New("composition learned-word budget exceeded")
	}
	return nil
}

func (k *KnowledgeBase) loadComposition(bank [compositionBanks][]NodeID, ids [][]NodeID) compositionWeights {
	w := compositionWeights{E: make([][]float64, len(ids))}
	for i, p := range ids {
		w.E[i] = make([]float64, compositionHidden)
		for h := range w.E[i] {
			w.E[i][h], _ = k.compositionParameter(p, bank[h])
		}
	}
	for out := 0; out < compositionOutputs; out++ {
		for h := 0; h < compositionHidden; h++ {
			w.H[out][h], _ = k.compositionParameter(bank[h], bank[12+out])
		}
		for pos := 0; pos < MaxCompositionWords; pos++ {
			w.P[out][pos], _ = k.compositionParameter(bank[18+pos], bank[12+out])
		}
		w.B[out], _ = k.compositionParameter(bank[24], bank[12+out])
	}
	return w
}

func compositionPoint(s CompositionScene, p CompositionPatch) float64 {
	if s.Point[2] < 0 {
		return 0
	}
	dx, dy := s.Point[0]-p.XY[0], s.Point[1]-p.XY[1]
	return math.Exp(-(dx*dx + dy*dy) / (.22 * .22))
}

func compositionForward(input CompositionInput, w compositionWeights) ([compositionOutputs]float64, []float64) {
	z := w.B
	for i, e := range w.E {
		for out := range z {
			v := 0.0
			for h := range e {
				v += w.H[out][h] * e[h]
			}
			z[out] += w.P[out][i] * v
		}
	}
	scores := make([]float64, len(input.Scene.Patches))
	maximum := math.Inf(-1)
	for j, p := range input.Scene.Patches {
		for d := 0; d < 3; d++ {
			scores[j] += z[d] * p.Values[d]
		}
		scores[j] += z[3] * compositionPoint(input.Scene, p)
		maximum = max(maximum, scores[j])
	}
	sum := 0.0
	for j := range scores {
		scores[j] = math.Exp(scores[j] - maximum)
		sum += scores[j]
	}
	for j := range scores {
		scores[j] /= sum
	}
	return z, scores
}

func compositionSigmoid(x float64) float64 {
	if x >= 0 {
		return 1 / (1 + math.Exp(-x))
	}
	v := math.Exp(x)
	return v / (1 + v)
}
func compositionStep(v, g, reliability float64) float64 {
	return max(-7.99, min(7.99, v-.05*reliability*(max(-3, min(3, g))+.0001*v)))
}

func (k *KnowledgeBase) LearnCompositionAt(c CompositionConfig, e CompositionExposure, at time.Time) error {
	if err := k.ValidateCompositionExposure(c, e); err != nil {
		return err
	}
	ids := make([][]NodeID, len(e.Input.Words))
	for i, v := range e.Input.Words {
		p, _, err := k.projectWrittenNumeric(NewNeuralVector(v), GroundingProjectionThreshold(), 4, at, true)
		if err != nil {
			return err
		}
		ids[i] = writtenPopulationIDs(p)
	}
	k.Lock()
	defer k.Unlock()
	bank, err := k.compositionBankLocked()
	if err != nil {
		return err
	}
	target, state, force, err := e.target()
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		if force == 0 {
			return nil
		}
		a, _ := k.compositionParameter(bank[25], bank[27])
		b, _ := k.compositionParameter(bank[26], bank[27])
		predictionError := a*e.Motor[2] + b - state
		k.adjustCompositionParameter(bank[25], bank[27], a-.12*e.Reliability*predictionError*e.Motor[2], e.Reliability, at, false)
		k.adjustCompositionParameter(bank[26], bank[27], b-.12*e.Reliability*predictionError, e.Reliability, at, false)
		return nil
	}
	for _, p := range ids {
		for h := 0; h < compositionHidden; h++ {
			if _, ok := k.compositionParameter(p, bank[h]); !ok {
				k.adjustCompositionParameter(p, bank[h], .04*compositionRandom(c.Seed, int(p[0]), h), 0, at, true)
			}
		}
	}
	w := k.loadComposition(bank, ids)
	for micro := 0; micro < 5; micro++ {
		z, prob := compositionForward(e.Input, w)
		var dz [compositionOutputs]float64
		if force > 0 {
			for j, p := range e.Input.Scene.Patches {
				delta := prob[j]
				if j == target {
					delta--
				}
				for d := 0; d < 3; d++ {
					dz[d] += delta * p.Values[d]
				}
				dz[3] += delta * compositionPoint(e.Input.Scene, p)
			}
			dz[4] = compositionSigmoid(z[4]) - (state+1)/2
		}
		dz[5] = compositionSigmoid(z[5]) - force
		old := w
		old.E = make([][]float64, len(w.E))
		for i, e := range w.E {
			old.E[i] = slices.Clone(e)
		}
		for out := 0; out < compositionOutputs; out++ {
			w.B[out] = compositionStep(old.B[out], dz[out], e.Reliability)
			for h := 0; h < compositionHidden; h++ {
				g := 0.0
				for i, v := range old.E {
					g += dz[out] * old.P[out][i] * v[h]
				}
				w.H[out][h] = compositionStep(old.H[out][h], g, e.Reliability)
			}
			for i, v := range old.E {
				sum := 0.0
				for h := range v {
					sum += old.H[out][h] * v[h]
				}
				w.P[out][i] = compositionStep(old.P[out][i], dz[out]*sum, e.Reliability)
			}
		}
		// Repeated positions of one word share one parameter update. Aggregate
		// their gradients before writing, rather than letting the last slot win.
		for i, p := range ids {
			first := true
			for j := 0; j < i; j++ {
				if slices.Equal(p, ids[j]) {
					first = false
					break
				}
			}
			if !first {
				continue
			}
			for h, v := range old.E[i] {
				g := 0.0
				for j, q := range ids {
					if slices.Equal(p, q) {
						for out := range dz {
							g += dz[out] * old.P[out][j] * old.H[out][h]
						}
					}
				}
				next := compositionStep(v, g, e.Reliability)
				for j, q := range ids {
					if slices.Equal(p, q) {
						w.E[j][h] = next
					}
				}
			}
		}
	}
	for i, p := range ids {
		first := true
		for j := 0; j < i; j++ {
			if slices.Equal(p, ids[j]) {
				first = false
			}
		}
		if first {
			for h, v := range w.E[i] {
				k.adjustCompositionParameter(p, bank[h], v, e.Reliability, at, false)
			}
		}
	}
	for out := 0; out < compositionOutputs; out++ {
		for h := 0; h < compositionHidden; h++ {
			k.adjustCompositionParameter(bank[h], bank[12+out], w.H[out][h], e.Reliability, at, false)
		}
		for pos := 0; pos < MaxCompositionWords; pos++ {
			k.adjustCompositionParameter(bank[18+pos], bank[12+out], w.P[out][pos], e.Reliability, at, false)
		}
		k.adjustCompositionParameter(bank[24], bank[12+out], w.B[out], e.Reliability, at, false)
	}
	return nil
}

// Keep the native numerical written population threshold in one place for
// composition exposure. Exact receptive fields preserve subsequent identity.
func GroundingProjectionThreshold() float64 { return .90 }

// ReadComposition is strictly read-only: unknown inputs cannot recruit units,
// initialize embeddings, update timestamps, or change activation/parameters.
func (k *KnowledgeBase) ReadComposition(c CompositionConfig, input CompositionInput, at time.Time) (CompositionReadout, error) {
	out := CompositionReadout{Status: "untrained", SceneAt: input.Scene.ObservedAt}
	if err := c.Validate(); err != nil {
		return out, err
	}
	if err := input.Validate(); err != nil {
		return out, err
	}
	if at.Before(input.Scene.ObservedAt) || at.Sub(input.Scene.ObservedAt) > time.Duration(c.MaxSceneAgeNS) {
		out.Status = "stale-scene"
		return out, nil
	}
	if len(input.Words) == 0 {
		out.Status = "missing-written-input"
		return out, nil
	}
	k.RLock()
	defer k.RUnlock()
	bank, err := k.compositionBankLocked()
	if err != nil {
		return out, err
	}
	if len(bank[0]) == 0 {
		return out, nil
	}
	ids, _ := k.compositionWordsLocked(input.Words)
	for _, p := range ids {
		if len(p) == 0 {
			out.Status = "unknown-written-form"
			return out, nil
		}
		if _, ok := k.compositionParameter(p, bank[0]); !ok {
			out.Status = "unknown-written-form"
			return out, nil
		}
	}
	w := k.loadComposition(bank, ids)
	z, scores := compositionForward(input, w)
	out.Coordinates = slices.Clone(z[:])
	execute, state := compositionSigmoid(z[5]), compositionSigmoid(z[4])
	if execute < .5 {
		out.Status = "no-op"
		out.Confidence = 1 - execute
		out.Goal = make([]float64, 5)
		out.Motor = make([]float64, 4)
		out.Accepted = true
		return out, nil
	}
	best := 0
	for j := range scores {
		if scores[j] > scores[best] {
			best = j
		}
	}
	second := 0.0
	for j, v := range scores {
		if j != best {
			second = max(second, v)
		}
	}
	out.ReferenceMargin = scores[best] - second
	out.Confidence = min(scores[best], max(state, 1-state), execute)
	if input.Scene.Point[2] < 0 && z[3] > math.Abs(z[0])+math.Abs(z[1])+math.Abs(z[2])+.5 {
		out.Status = "missing-pointing"
		return out, nil
	}
	if out.Confidence < .5 || out.ReferenceMargin < .1 {
		out.Status = "ambiguous"
		return out, nil
	}
	a, _ := k.compositionParameter(bank[25], bank[27])
	b, _ := k.compositionParameter(bank[26], bank[27])
	if math.Abs(a) < .1 {
		out.Status = "untrained-controller"
		return out, nil
	}
	goal := 1.0
	if state < .5 {
		goal = -1
	}
	setpoint := 1.0
	if (goal-b)/a < 0 {
		setpoint = -1
	}
	p := input.Scene.Patches[best]
	out.Goal = append(slices.Clone(p.Values[:3]), goal, 1)
	out.Motor = []float64{p.XY[0], p.XY[1], setpoint, 1}
	out.LocalEffect = a*setpoint + b
	out.Accepted = true
	out.Status = "proposal"
	return out, nil
}

// Detached bounded context on the same output populations lets native
// recurrent processing use the composed coordinates. It grants no permission.
func (k *KnowledgeBase) CompositionActivation(out CompositionReadout) (map[NodeID]float64, error) {
	k.RLock()
	defer k.RUnlock()
	bank, err := k.compositionBankLocked()
	if err != nil {
		return nil, err
	}
	values := map[NodeID]float64{}
	for i, v := range out.Coordinates {
		if i >= compositionOutputs {
			break
		}
		for _, id := range bank[12+i] {
			values[id] = .05 * (1 + math.Tanh(v))
		}
	}
	return values, nil
}

// Validate the whole bank before a checkpoint can replace the target Brain.
func (k *KnowledgeBase) ValidateCompositionBank(enabled bool) error {
	k.RLock()
	defer k.RUnlock()
	bank, err := k.compositionBankLocked()
	if err != nil {
		return err
	}
	for i, p := range bank {
		if enabled && len(p) != 4 {
			return fmt.Errorf("composition bank coordinate %d is missing", i)
		}
		if !enabled && len(p) > 0 {
			return errors.New("composition coordinates require a complete authenticated composition checkpoint")
		}
	}
	return nil
}
