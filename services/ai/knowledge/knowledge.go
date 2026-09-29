package knowledge

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type NeuralRegistry struct {
	mu       sync.RWMutex
	nextID   NodeID
	byID     map[NodeID]*ConceptNode
}

func NewNeuralRegistry() *NeuralRegistry { return &NeuralRegistry{nextID: 1, byID: make(map[NodeID]*ConceptNode)} }

// TokenRegistry is retained only as a source-compatibility alias during migration.
// It is not token-indexed and stores no lexical identity.
type TokenRegistry = NeuralRegistry

func NewTokenRegistry() *TokenRegistry { return NewNeuralRegistry() }
func canonicalToken(token string) string { return strings.ToLower(strings.TrimSpace(token)) }

func (r *NeuralRegistry) GetOrCreate(token string) (*ConceptNode, bool, error) {
	canonical := canonicalToken(token)
	if canonical == "" {
		return nil, false, errors.New("token is empty")
	}
	// Language is an input adapter only. The lexical form is immediately
	// converted into a modality-neutral numeric representation and is never
	// stored on the neural unit or used as its identity.
	return r.GetOrCreateRepresentation(observationVector(canonical, "language"), 0.999999)
}

func (r *TokenRegistry) Get(token string) *ConceptNode {
	canonical := canonicalToken(token)
	if canonical == "" {
		return nil
	}
	vector := observationVector(canonical, "language")
	r.mu.RLock()
	defer r.mu.RUnlock()
	best := (*ConceptNode)(nil)
	bestScore := 0.0
	for _, node := range r.byID {
		if len(node.Representation) != len(vector.Values) {
			continue
		}
		score := NewNeuralVector(node.Representation).Similarity(vector)
		if score > bestScore || (score == bestScore && (best == nil || node.ID < best.ID)) {
			best, bestScore = node, score
		}
	}
	if best != nil && bestScore >= 0.999999 {
		return best
	}
	return nil
}

func (r *TokenRegistry) GetByID(id NodeID) *ConceptNode {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.byID[id]
}

func (r *TokenRegistry) Nodes() []*ConceptNode {
	r.mu.RLock()
	defer r.mu.RUnlock()
	nodes := make([]*ConceptNode, 0, len(r.byID))
	for _, n := range r.byID {
		nodes = append(nodes, n)
	}
	// The registry is keyed by a map, so iteration order is undefined.
	// Checkpoints must serialize the same neural substrate identically across
	// replay/restart, independent of map iteration order.
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })
	return nodes
}

// GetOrCreateRepresentation reuses an existing numeric neural unit when its
// learned prototype is sufficiently similar. Token identity is not involved.
// A new unit is created only when no compatible prototype exists.
func (r *TokenRegistry) GetOrCreateRepresentation(vector NeuralVector, threshold float64) (*ConceptNode, bool, error) {
	if vector.Empty() { return nil, false, errors.New("neural vector is empty") }
	threshold = clamp(threshold, 0, 1)
	r.mu.Lock()
	defer r.mu.Unlock()

	best := (*ConceptNode)(nil)
	bestScore := 0.0
	for _, node := range r.byID {
		if len(node.Representation) != len(vector.Values) { continue }
		score := NewNeuralVector(node.Representation).Similarity(vector)
		if score > bestScore { best, bestScore = node, score }
	}
	if best != nil && bestScore >= threshold {
		best.Frequency++
		best.LastActivation = time.Now().UTC()
		return best, false, nil
	}

	n := newRepresentationNode(r.nextID, vector.Values)
	n.Frequency = 1
	r.nextID++
	r.byID[n.ID] = n
	return n, true, nil
}

// KnowledgeBase stores the persistent neural substrate. Semantic labels are
// optional compatibility metadata and are not required to form or traverse
// dynamic connections. ProjectionPopulations records distributed receptive
// fields so repeated experiences can recover the same population without
// turning similar experiences into one identical representation.
type BootstrapProvenance struct { Origin string `json:"origin"`; DirectExperience bool `json:"direct_experience"`; Status string `json:"status"` }

type BootstrapExperience struct { ID string `json:"id"`; Populations []NodeID `json:"populations"`; TemporalTrace []time.Time `json:"temporal_trace,omitempty"`; Activation float64 `json:"activation"`; Confidence float64 `json:"confidence"`; Provenance BootstrapProvenance `json:"provenance"` }

type LearningPolicyState struct {
	NoveltySensitivity float64 `json:"novelty_sensitivity"`
	UncertaintySensitivity float64 `json:"uncertainty_sensitivity"`
	MemoryRetention float64 `json:"memory_retention"`
	CuriosityPressure float64 `json:"curiosity_pressure"`
	ExplorationBias float64 `json:"exploration_bias"`
	RepeatObservationBias float64 `json:"repeat_observation_bias"`
	DeferConclusionBias float64 `json:"defer_conclusion_bias"`
}

type InquiryState struct {
	Sequence uint64 `json:"sequence"`
	LastUpdated time.Time `json:"last_updated"`
	Pending bool `json:"pending"`
	SelectedAction string `json:"selected_action,omitempty"`
	Uncertainty float64 `json:"uncertainty"`
	ExpectedInformationGain float64 `json:"expected_information_gain"`
	SelectedScore float64 `json:"selected_score"`
	ObservedInformationGain float64 `json:"observed_information_gain"`
	OutcomePredictionError float64 `json:"outcome_prediction_error"`
}

type BrainState struct {
	AttentionState map[string]float64 `json:"attention_state,omitempty"`
	CuriosityState map[string]float64 `json:"curiosity_state,omitempty"`
	PredictionState map[string]float64 `json:"prediction_state,omitempty"`
	PlasticityState map[string]float64 `json:"plasticity_state,omitempty"`
	MemoryState map[string]float64 `json:"memory_state,omitempty"`
	SelfModelState map[string]float64 `json:"self_model_state,omitempty"`
	SocialModelState map[string]float64 `json:"social_model_state,omitempty"`
	ValueState map[string]float64 `json:"value_state,omitempty"`
	LearningPolicyState LearningPolicyState `json:"learning_policy_state"`
	InquiryState InquiryState `json:"inquiry_state"`
	BootstrapExperiences []BootstrapExperience `json:"bootstrap_experiences,omitempty"`
	Episodes []BootstrapExperience `json:"episodes,omitempty"`
	ExperienceTraces []ExperienceTrace `json:"experience_traces,omitempty"`
}

type KnowledgeBase struct {
	Registry              *NeuralRegistry
	Patterns              *PatternIndex
	ProjectionPopulations []ProjectionPopulation `json:"projection_populations,omitempty"`
	BrainState            BrainState `json:"brain_state"`
	SurfaceAnnotations    []SurfaceAnnotation `json:"surface_annotations,omitempty"`
	projectionMu          sync.RWMutex
	annotationMu          sync.RWMutex
	mu                    sync.RWMutex
}

// Lock serializes mutations to the canonical neural substrate.
func (k *KnowledgeBase) Lock() { if k != nil { k.mu.Lock() } }
func (k *KnowledgeBase) Unlock() { if k != nil { k.mu.Unlock() } }
func (k *KnowledgeBase) RLock() { if k != nil { k.mu.RLock() } }
func (k *KnowledgeBase) RUnlock() { if k != nil { k.mu.RUnlock() } }

func NewKnowledgeBase() *KnowledgeBase {
	return &KnowledgeBase{
		Registry: NewTokenRegistry(),
		Patterns: NewPatternIndex(),
		BrainState: BrainState{
			AttentionState: map[string]float64{},
			CuriosityState: map[string]float64{},
			PredictionState: map[string]float64{},
			PlasticityState: map[string]float64{},
			MemoryState: map[string]float64{},
			SelfModelState: map[string]float64{},
			SocialModelState: map[string]float64{},
			ValueState: map[string]float64{},
			LearningPolicyState: LearningPolicyState{
				NoveltySensitivity: 0.70,
				UncertaintySensitivity: 0.80,
				MemoryRetention: 0.60,
				CuriosityPressure: 0.65,
				ExplorationBias: 0.30,
				RepeatObservationBias: 0.20,
				DeferConclusionBias: 0.20,
			},
		},
	}
}
func (k *KnowledgeBase) Fetch(token string) *ConceptNode {
	if k == nil || k.Registry == nil { return nil }
	if node := k.Registry.Get(token); node != nil {
		return node
	}
	canonical := canonicalToken(token)
	if canonical == "" {
		return nil
	}
	k.annotationMu.RLock()
	annotations := append([]SurfaceAnnotation(nil), k.SurfaceAnnotations...)
	k.annotationMu.RUnlock()
	// Compatibility lookup: prefer a language annotation, otherwise use the
	// most recently observed surface. Cognition should consume grounded NodeIDs
	// rather than call this lexical adapter.
	var fallback *SurfaceAnnotation
	for i := range annotations {
		if annotations[i].Surface != canonical || annotations[i].NodeID == 0 {
			continue
		}
		if annotations[i].Modality == "language" {
			fallback = &annotations[i]
			break
		}
		copy := annotations[i]
		if fallback == nil || copy.LastSeen.After(fallback.LastSeen) {
			fallback = &copy
		}
	}
	if fallback == nil {
		return nil
	}
	return k.Registry.GetByID(fallback.NodeID)
}

func (k *KnowledgeBase) recordSurfaceAnnotation(surface, modality string, nodeID NodeID, population []NodeID, now time.Time) {
	if k == nil || surface == "" || nodeID == 0 {
		return
	}
	k.annotationMu.Lock()
	defer k.annotationMu.Unlock()
	for i := range k.SurfaceAnnotations {
		annotation := &k.SurfaceAnnotations[i]
		if annotation.Surface == surface && annotation.Modality == modality {
			annotation.NodeID = nodeID
			annotation.Population = append(annotation.Population[:0], population...)
			annotation.LastSeen = now
			return
		}
	}
	k.SurfaceAnnotations = append(k.SurfaceAnnotations, SurfaceAnnotation{
		Surface: surface, Modality: modality, NodeID: nodeID,
		Population: append([]NodeID(nil), population...), LastSeen: now,
	})
}
// SurfaceForNode resolves adapter metadata for a neural node without making
// the surface form part of neural identity. A distributed population is
// represented by one annotation anchored at its first member, so resolution
// also checks population membership.
func (k *KnowledgeBase) SurfaceForNode(id NodeID, modality string) string {
	if k == nil || id == 0 { return "" }
	k.annotationMu.RLock()
	defer k.annotationMu.RUnlock()
	best := ""
	var bestSeen time.Time
	for _, annotation := range k.SurfaceAnnotations {
		if modality != "" && annotation.Modality != modality { continue }
		match := annotation.NodeID == id
		if !match {
			for _, member := range annotation.Population {
				if member == id { match = true; break }
			}
		}
		if !match || annotation.Surface == "" { continue }
		if best == "" || annotation.LastSeen.After(bestSeen) {
			best = annotation.Surface
			bestSeen = annotation.LastSeen
		}
	}
	return best
}

func (k *KnowledgeBase) Store(token string) *ConceptNode {
	if k == nil || k.Registry == nil {
		return nil
	}
	canonical := canonicalToken(token)
	if canonical == "" {
		return nil
	}
	nodeID, _, _, err := k.ProjectVector(observationVector(canonical, "language"), 0.999999)
	if err != nil {
		return nil
	}
	k.recordSurfaceAnnotation(canonical, "language", nodeID, []NodeID{nodeID}, time.Now().UTC())
	return k.Registry.GetByID(nodeID)
}

// ProjectVector presents numeric experience to the same neural substrate used
// by language and other sources. The result is an internal activation pattern,
// not a separate vector memory database.
func (k *KnowledgeBase) ProjectVector(vector NeuralVector, threshold float64) (NodeID, float64, bool, error) {
	if k == nil { return 0, 0, false, errors.New("brain is nil") }
	k.mu.Lock()
	defer k.mu.Unlock()
	node, created, err := k.Registry.GetOrCreateRepresentation(vector, threshold)
	if err != nil { return 0, 0, false, err }
	score := 1.0
	if !created { score = NewNeuralVector(node.Representation).Similarity(vector) }
	node.Activation = score
	node.LastActivation = time.Now().UTC()
	return node.ID, score, created, nil
}

func (k *KnowledgeBase) Connect(source, target *ConceptNode, weight, confidence float64, inhibitory bool) {
	if source == nil || target == nil { return }
	k.mu.Lock()
	defer k.mu.Unlock()
	now := time.Now().UTC(); if source.Synapses == nil { source.Synapses = make(map[NodeID]SynapseList) }
	list := source.Synapses[target.ID]; s := list.FindDynamic(inhibitory)
	if s == nil { s = &Synapse{TargetID: target.ID, Weight: clamp01(weight), Confidence: clamp01(confidence), Inhibitory: inhibitory}; source.Synapses[target.ID] = append(list, s) }
	s.Dynamic.Reinforce(weight, confidence, 1, now); syncSynapseLegacyState(s); source.LastActivation = now
}

func (k *KnowledgeBase) ConnectKind(source, target *ConceptNode, kind RelationKind, weight, confidence float64, inhibitory bool) {
	if source == nil || target == nil { return }
	k.mu.Lock()
	defer k.mu.Unlock()
	now := time.Now().UTC(); if source.Synapses == nil { source.Synapses = make(map[NodeID]SynapseList) }
	list := source.Synapses[target.ID]; s := list.Find(kind, inhibitory)
	if s == nil { s = &Synapse{TargetID: target.ID, Kind: kind, Weight: weight, Confidence: confidence, Inhibitory: inhibitory}; source.Synapses[target.ID] = append(list, s) }
	s.Dynamic.Reinforce(weight, confidence, 1, now); syncSynapseLegacyState(s); source.LastActivation = now
}

func syncSynapseLegacyState(s *Synapse) {
	if s == nil { return }; s.Weight = s.Dynamic.Weight; s.Confidence = s.Dynamic.Confidence; s.Frequency = s.Dynamic.Frequency; s.Activation = s.Dynamic.Activation; s.LastActivation = s.Dynamic.LastActivation
}
func hydrateSynapseDynamicState(s *Synapse) {
	if s == nil { return }; if s.Dynamic.Frequency == 0 && s.Frequency > 0 { s.Dynamic.Weight = clamp01(s.Weight); s.Dynamic.Confidence = clamp01(s.Confidence); s.Dynamic.Activation = clamp01(s.Activation); s.Dynamic.Frequency = s.Frequency; s.Dynamic.LastActivation = s.LastActivation; s.Dynamic.LastModification = s.LastActivation }; syncSynapseLegacyState(s)
}

type persistedGraph struct {
	SchemaVersion          int                    `json:"schema_version,omitempty"`
	BrainIdentity          string                 `json:"brain_identity,omitempty"`
	Nodes                  []*ConceptNode         `json:"nodes"`
	Patterns               []*PatternSynapse      `json:"patterns,omitempty"`
	ProjectionPopulations  []ProjectionPopulation `json:"projection_populations,omitempty"`
	BrainState              BrainState             `json:"brain_state"`
}

func (k *KnowledgeBase) Load(path string) error {
	if k == nil { return ErrNilBrain }
	k.mu.Lock()
	defer k.mu.Unlock()
	b, err := os.ReadFile(path); if err != nil { return err }
	nodes, patterns, populations, state, err := loadCanonicalBrain(b)
	if err != nil { return err }
	registry := NewTokenRegistry(); var maxID NodeID
	for _, node := range nodes { if node == nil { continue }; if node.Synapses == nil { node.Synapses = map[NodeID]SynapseList{} }; for _, synapses := range node.Synapses { for _, synapse := range synapses { hydrateSynapseDynamicState(synapse) } }; registry.byID[node.ID] = node; if node.ID > maxID { maxID = node.ID } }
	registry.nextID = maxID + 1; if registry.nextID < 1 { registry.nextID = 1 }; k.Registry = registry
	patternIndex := NewPatternIndex(); var maxPatternID PatternID; for _, ps := range patterns { if ps == nil { continue }; patternIndex.patterns[ps.ID] = ps; if ps.ID > maxPatternID { maxPatternID = ps.ID } }; patternIndex.nextID = maxPatternID + 1; if patternIndex.nextID < 1 { patternIndex.nextID = 1 }; k.Patterns = patternIndex
	k.projectionMu.Lock(); k.ProjectionPopulations = append([]ProjectionPopulation(nil), populations...); k.projectionMu.Unlock()
	k.BrainState = state
	k.normalizeBrainState()
	k.normalizeBootstrapState()

	return nil
}

// normalizeBootstrapState upgrades older persisted bootstrap populations to the
// current materialization schema without changing their neural identities.
// This is migration metadata only: it never invents semantic labels or world
// facts and never creates a second memory store.
func (k *KnowledgeBase) normalizeBootstrapState() {
	if k == nil || len(k.ProjectionPopulations) == 0 {
		return
	}
	limit := len(BootstrapNetworkDomains)
	if len(k.ProjectionPopulations) < limit {
		limit = len(k.ProjectionPopulations)
	}
	for i := 0; i < limit; i++ {
		population := &k.ProjectionPopulations[i]
		if population.Domain == "" {
			population.Domain = BootstrapNetworkDomains[i].Name
		}
		if len(population.LearningTarget) == 0 {
			for _, unit := range population.Units {
				if unit.NodeID != 0 {
					population.LearningTarget = append(population.LearningTarget, unit.NodeID)
				}
			}
		}
		if len(population.CounterEvidenceTargets) == 0 && len(k.ProjectionPopulations) > 10 {
			for _, unit := range k.ProjectionPopulations[10].Units {
				if unit.NodeID != 0 {
					population.CounterEvidenceTargets = append(population.CounterEvidenceTargets, unit.NodeID)
				}
			}
		}
		if population.TemporalScale == "" {
			population.TemporalScale = bootstrapTemporalScale(i)
		}
	}
}

func (k *KnowledgeBase) normalizeBrainState() {
	if k == nil {
		return
	}
	if k.BrainState.AttentionState == nil {
		k.BrainState.AttentionState = map[string]float64{}
	}
	if k.BrainState.CuriosityState == nil {
		k.BrainState.CuriosityState = map[string]float64{}
	}
	if k.BrainState.PredictionState == nil {
		k.BrainState.PredictionState = map[string]float64{}
	}
	if k.BrainState.PlasticityState == nil {
		k.BrainState.PlasticityState = map[string]float64{}
	}
	if k.BrainState.MemoryState == nil {
		k.BrainState.MemoryState = map[string]float64{}
	}
	if k.BrainState.SelfModelState == nil {
		k.BrainState.SelfModelState = map[string]float64{}
	}
	if k.BrainState.SocialModelState == nil {
		k.BrainState.SocialModelState = map[string]float64{}
	}
	if k.BrainState.ValueState == nil {
		k.BrainState.ValueState = map[string]float64{}
	}
	policy := &k.BrainState.LearningPolicyState
	if policy.NoveltySensitivity <= 0 {
		policy.NoveltySensitivity = 0.70
	}
	if policy.UncertaintySensitivity <= 0 {
		policy.UncertaintySensitivity = 0.80
	}
	if policy.MemoryRetention <= 0 {
		policy.MemoryRetention = 0.60
	}
	if policy.CuriosityPressure <= 0 {
		policy.CuriosityPressure = 0.65
	}
	if policy.ExplorationBias <= 0 {
		policy.ExplorationBias = 0.30
	}
	if policy.RepeatObservationBias <= 0 {
		policy.RepeatObservationBias = 0.20
	}
	if policy.DeferConclusionBias <= 0 {
		policy.DeferConclusionBias = 0.20
	}
	k.BrainState.InquiryState.Uncertainty = clamp01(k.BrainState.InquiryState.Uncertainty)
	k.BrainState.InquiryState.ExpectedInformationGain = clamp01(k.BrainState.InquiryState.ExpectedInformationGain)
	k.BrainState.InquiryState.SelectedScore = clamp01(k.BrainState.InquiryState.SelectedScore)
	k.BrainState.InquiryState.ObservedInformationGain = clamp01(k.BrainState.InquiryState.ObservedInformationGain)
	k.BrainState.InquiryState.OutcomePredictionError = clamp01(k.BrainState.InquiryState.OutcomePredictionError)
}

// RecordInquirySelection persists the internal inquiry decision without
// executing the selected action. Only the numeric trajectory is persisted.
func (k *KnowledgeBase) RecordInquirySelection(sequence uint64, action string, uncertainty, informationGain, score float64, now time.Time) {
	if k == nil || action == "" { return }
	if now.IsZero() { now = time.Now().UTC() }
	k.mu.Lock()
	defer k.mu.Unlock()
	k.BrainState.InquiryState = InquiryState{
		Sequence: sequence, LastUpdated: now.UTC(), Pending: true,
		SelectedAction: action, Uncertainty: clamp01(uncertainty),
		ExpectedInformationGain: clamp01(informationGain), SelectedScore: clamp01(score),
	}
}

// RecordInquiryOutcome closes the inquiry trajectory and adapts the persisted
// learning strategy from observed information gain and prediction error.
func (k *KnowledgeBase) RecordInquiryOutcome(observedInformationGain, predictionError float64, now time.Time) {
	if k == nil { return }
	if now.IsZero() { now = time.Now().UTC() }
	observedInformationGain = clamp01(observedInformationGain)
	predictionError = clamp01(predictionError)
	k.mu.Lock()
	defer k.mu.Unlock()
	state := &k.BrainState.InquiryState
	state.LastUpdated = now.UTC()
	state.Pending = false
	state.ObservedInformationGain = observedInformationGain
	state.OutcomePredictionError = predictionError
	policy := &k.BrainState.LearningPolicyState
	policy.ExplorationBias = clamp01(policy.ExplorationBias + 0.20*predictionError)
	policy.RepeatObservationBias = clamp01(policy.RepeatObservationBias + 0.15*predictionError)
	policy.DeferConclusionBias = clamp01(policy.DeferConclusionBias + 0.10*predictionError)
	policy.CuriosityPressure = clamp01(policy.CuriosityPressure + 0.10*predictionError)
	if observedInformationGain > 0.70 && predictionError < 0.25 {
		policy.ExplorationBias = clamp01(policy.ExplorationBias + 0.05*observedInformationGain)
	}
}

func (k *KnowledgeBase) Save(path string) error {
	if k == nil { return ErrNilBrain }
	k.mu.RLock()
	defer k.mu.RUnlock()
	b, e := marshalCanonicalBrain(k)
	if e != nil { return e }
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil { return err }
	tmp, err := os.CreateTemp(dir, ".brain-memory-*.tmp")
	if err != nil { return err }
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(b); err != nil { _ = tmp.Close(); return err }
	if err := tmp.Sync(); err != nil { _ = tmp.Close(); return err }
	if err := tmp.Close(); err != nil { return err }
	return os.Rename(tmpName, path)
}
func clamp01(v float64) float64 { if v < 0 { return 0 }; if v > 1 { return 1 }; return v }