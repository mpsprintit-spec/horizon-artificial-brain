package knowledge

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"
	"time"
)

type TokenRegistry struct {
	mu       sync.RWMutex
	nextID   NodeID
	byID     map[NodeID]*ConceptNode
}

func NewTokenRegistry() *TokenRegistry { return &TokenRegistry{nextID: 1, byID: make(map[NodeID]*ConceptNode)} }
func canonicalToken(token string) string { return strings.ToLower(strings.TrimSpace(token)) }

func (r *TokenRegistry) GetOrCreate(token string) (*ConceptNode, bool, error) {
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
		if score > bestScore {
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
type KnowledgeBase struct {
	Registry             *TokenRegistry
	Patterns             *PatternIndex
	ProjectionPopulations []ProjectionPopulation `json:"projection_populations,omitempty"`
	projectionMu         sync.RWMutex
	mu                   sync.RWMutex
}

// Lock serializes mutations to the canonical neural substrate.
func (k *KnowledgeBase) Lock() { if k != nil { k.mu.Lock() } }
func (k *KnowledgeBase) Unlock() { if k != nil { k.mu.Unlock() } }
func (k *KnowledgeBase) RLock() { if k != nil { k.mu.RLock() } }
func (k *KnowledgeBase) RUnlock() { if k != nil { k.mu.RUnlock() } }

func NewKnowledgeBase() *KnowledgeBase { return &KnowledgeBase{Registry: NewTokenRegistry(), Patterns: NewPatternIndex()} }
func (k *KnowledgeBase) Fetch(token string) *ConceptNode {
	if k == nil || k.Registry == nil { return nil }
	return k.Registry.Get(token)
}
func (k *KnowledgeBase) Store(token string) *ConceptNode {
	if k == nil || k.Registry == nil { return nil }
	k.mu.Lock()
	defer k.mu.Unlock()
	n, _, _ := k.Registry.GetOrCreate(token)
	return n
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
	Nodes                 []*ConceptNode      `json:"nodes"`
	Patterns              []*PatternSynapse   `json:"patterns,omitempty"`
	ProjectionPopulations []ProjectionPopulation `json:"projection_populations,omitempty"`
}

func (k *KnowledgeBase) Load(path string) error {
	if k == nil { return ErrNilBrain }
	k.mu.Lock()
	defer k.mu.Unlock()
	b, err := os.ReadFile(path); if err != nil { return err }; var graph persistedGraph; if err := json.Unmarshal(b, &graph); err != nil { return err }
	registry := NewTokenRegistry(); var maxID NodeID
	for _, node := range graph.Nodes { if node == nil { continue }; if node.Synapses == nil { node.Synapses = map[NodeID]SynapseList{} }; for _, synapses := range node.Synapses { for _, synapse := range synapses { hydrateSynapseDynamicState(synapse) } }; registry.byID[node.ID] = node; if node.ID > maxID { maxID = node.ID } }
	registry.nextID = maxID + 1; if registry.nextID < 1 { registry.nextID = 1 }; k.Registry = registry
	patternIndex := NewPatternIndex(); var maxPatternID PatternID; for _, ps := range graph.Patterns { if ps == nil { continue }; patternIndex.patterns[ps.ID] = ps; if ps.ID > maxPatternID { maxPatternID = ps.ID } }; patternIndex.nextID = maxPatternID + 1; if patternIndex.nextID < 1 { patternIndex.nextID = 1 }; k.Patterns = patternIndex
	k.projectionMu.Lock(); k.ProjectionPopulations = append([]ProjectionPopulation(nil), graph.ProjectionPopulations...); k.projectionMu.Unlock()
	return nil
}
func (k *KnowledgeBase) Save(path string) error {
	if k == nil { return ErrNilBrain }
	k.mu.RLock()
	defer k.mu.RUnlock()
	k.projectionMu.RLock()
	populations := append([]ProjectionPopulation(nil), k.ProjectionPopulations...)
	k.projectionMu.RUnlock()
	var nodes []*ConceptNode
	if k.Registry != nil { nodes = k.Registry.Nodes() }
	var patterns []*PatternSynapse
	if k.Patterns != nil { patterns = k.Patterns.All() }
	b, e := json.MarshalIndent(persistedGraph{Nodes: nodes, Patterns: patterns, ProjectionPopulations: populations}, "", "  ")
	if e != nil { return e }
	return os.WriteFile(path, b, 0644)
}
func clamp01(v float64) float64 { if v < 0 { return 0 }; if v > 1 { return 1 }; return v }