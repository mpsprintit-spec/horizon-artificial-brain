package knowledge

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"
	"time"
)

// TokenRegistry is the only authority allowed to create token nodes.
type TokenRegistry struct {
	mu      sync.RWMutex
	nextID  NodeID
	byID    map[NodeID]*ConceptNode
	byToken map[string]NodeID
}

func NewTokenRegistry() *TokenRegistry {
	return &TokenRegistry{nextID: 1, byID: make(map[NodeID]*ConceptNode), byToken: make(map[string]NodeID)}
}

func canonicalToken(token string) string { return strings.ToLower(strings.TrimSpace(token)) }

// GetOrCreate returns the single canonical node for token, creating it only if absent.
func (r *TokenRegistry) GetOrCreate(token string) (*ConceptNode, bool, error) {
	canonical := canonicalToken(token)
	if canonical == "" {
		return nil, false, errors.New("token is empty")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if id, ok := r.byToken[canonical]; ok {
		n := r.byID[id]
		n.Frequency++
		n.LastActivation = time.Now().UTC()
		return n, false, nil
	}
	n := newConceptNode(r.nextID, canonical)
	r.nextID++
	n.Frequency = 1
	r.byToken[canonical] = n.ID
	r.byID[n.ID] = n
	return n, true, nil
}

func (r *TokenRegistry) Get(token string) *ConceptNode {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.byID[r.byToken[canonicalToken(token)]]
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

// KnowledgeBase is Horizon's neural semantic memory graph.
type KnowledgeBase struct {
	Registry *TokenRegistry
	Patterns *PatternIndex
}

func NewKnowledgeBase() *KnowledgeBase {
	return &KnowledgeBase{Registry: NewTokenRegistry(), Patterns: NewPatternIndex()}
}
func (k *KnowledgeBase) Fetch(token string) *ConceptNode { return k.Registry.Get(token) }
func (k *KnowledgeBase) Store(token string) *ConceptNode {
	n, _, _ := k.Registry.GetOrCreate(token)
	return n
}

func (k *KnowledgeBase) Connect(source, target *ConceptNode, weight, confidence float64, inhibitory bool) {
	k.ConnectKind(source, target, RelationAssociation, weight, confidence, inhibitory)
}

func (k *KnowledgeBase) ConnectKind(source, target *ConceptNode, kind RelationKind, weight, confidence float64, inhibitory bool) {
	if source == nil || target == nil {
		return
	}
	now := time.Now().UTC()
	if source.Synapses == nil {
		source.Synapses = make(map[NodeID]SynapseList)
	}
	list := source.Synapses[target.ID]
	s := list.Find(kind, inhibitory)
	if s == nil {
		s = &Synapse{TargetID: target.ID, Kind: kind, Weight: weight, Confidence: confidence, Inhibitory: inhibitory}
		source.Synapses[target.ID] = append(list, s)
	} else {
		s.Weight = clamp01((s.Weight*float64(s.Frequency) + weight) / float64(s.Frequency+1))
		s.Confidence = clamp01(1 - (1-s.Confidence)*(1-confidence))
	}
	if s.Kind == "" {
		s.Kind = kind
	}
	// Do not overwrite Kind/Inhibitory of a different channel — each channel is independent.
	s.Frequency++
	s.LastActivation = now
	source.LastActivation = now
}


type persistedGraph struct {
	Nodes    []*ConceptNode  `json:"nodes"`
	Patterns []*PatternSynapse `json:"patterns,omitempty"`
}

func (k *KnowledgeBase) Load(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var graph persistedGraph
	if err := json.Unmarshal(b, &graph); err != nil {
		return err
	}
	registry := NewTokenRegistry()
	var maxID NodeID
	for _, node := range graph.Nodes {
		if node == nil || canonicalToken(node.Token) == "" {
			continue
		}
		node.Token = canonicalToken(node.Token)
		if node.Synapses == nil {
			node.Synapses = map[NodeID]SynapseList{}
		}
		registry.byID[node.ID] = node
		registry.byToken[node.Token] = node.ID
		if node.ID > maxID {
			maxID = node.ID
		}
	}
	registry.nextID = maxID + 1
	if registry.nextID < 1 {
		registry.nextID = 1
	}
	k.Registry = registry

	patternIndex := NewPatternIndex()
	var maxPatternID PatternID
	for _, ps := range graph.Patterns {
		if ps == nil {
			continue
		}
		patternIndex.patterns[ps.ID] = ps
		if ps.ID > maxPatternID {
			maxPatternID = ps.ID
		}
	}
	patternIndex.nextID = maxPatternID + 1
	if patternIndex.nextID < 1 {
		patternIndex.nextID = 1
	}
	k.Patterns = patternIndex
	return nil
}

func (k *KnowledgeBase) Save(path string) error {
	b, e := json.MarshalIndent(persistedGraph{Nodes: k.Registry.Nodes(), Patterns: k.Patterns.All()}, "", "  ")
	if e != nil {
		return e
	}
	return os.WriteFile(path, b, 0644)
}
func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
