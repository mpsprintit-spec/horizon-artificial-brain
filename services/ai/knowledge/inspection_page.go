package knowledge

import (
	"container/heap"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strings"
	"time"
)

const MaxInspectionLimit = 512
const MaxInspectionBytes = 512 * 1024

var ErrInspectionTooLarge = errors.New("inspection record exceeds page budget; use complete persistence")

type InspectionRequest struct {
	Section string
	After   string
	Order   string
	Limit   int
}
type InspectionData struct {
	Section           string                        `json:"section"`
	Order             string                        `json:"order"`
	Items             []json.RawMessage             `json:"items"`
	Next              string                        `json:"next_key,omitempty"`
	Total             int                           `json:"total"`
	Counts            map[string]int                `json:"counts"`
	RelatedSynapses   []json.RawMessage             `json:"related_synapses,omitempty"`
	RelatedTotal      int                           `json:"related_total,omitempty"`
	StatePreview      map[string]map[string]float64 `json:"state_preview,omitempty"`
	StatePreviewLimit int                           `json:"state_preview_limit,omitempty"`
}
type inspectionKey struct {
	Source     NodeID  `json:"s,omitempty"`
	Target     NodeID  `json:"t,omitempty"`
	Bucket     NodeID  `json:"b,omitempty"`
	Ordinal    int     `json:"i,omitempty"`
	Kind       string  `json:"k,omitempty"`
	Inhibitory bool    `json:"h,omitempty"`
	Name       string  `json:"n,omitempty"`
	Score      float64 `json:"a,omitempty"`
}

func keyLess(a, b inspectionKey, activity bool) bool {
	if activity && a.Score != b.Score {
		return a.Score > b.Score
	}
	if a.Source != b.Source {
		return a.Source < b.Source
	}
	if a.Target != b.Target {
		return a.Target < b.Target
	}
	if a.Inhibitory != b.Inhibitory {
		return !a.Inhibitory
	}
	if a.Kind != b.Kind {
		return a.Kind < b.Kind
	}
	if a.Bucket != b.Bucket {
		return a.Bucket < b.Bucket
	}
	if a.Ordinal != b.Ordinal {
		return a.Ordinal < b.Ordinal
	}
	return a.Name < b.Name
}

type inspectionHeap struct {
	keys     []inspectionKey
	activity bool
	limit    int
	total    int
	after    *inspectionKey
}

func (h inspectionHeap) Len() int           { return len(h.keys) }
func (h inspectionHeap) Less(i, j int) bool { return keyLess(h.keys[j], h.keys[i], h.activity) }
func (h inspectionHeap) Swap(i, j int)      { h.keys[i], h.keys[j] = h.keys[j], h.keys[i] }
func (h *inspectionHeap) Push(v any)        { h.keys = append(h.keys, v.(inspectionKey)) }
func (h *inspectionHeap) Pop() any {
	last := h.keys[len(h.keys)-1]
	h.keys = h.keys[:len(h.keys)-1]
	return last
}
func (h *inspectionHeap) add(key inspectionKey) {
	h.total++
	if h.after != nil && !keyLess(*h.after, key, h.activity) {
		return
	}
	if h.Len() < h.limit {
		heap.Push(h, key)
	} else if keyLess(key, h.keys[0], h.activity) {
		h.keys[0] = key
		heap.Fix(h, 0)
	}
}
func (h *inspectionHeap) sorted() []inspectionKey {
	sort.Slice(h.keys, func(i, j int) bool { return keyLess(h.keys[i], h.keys[j], h.activity) })
	return h.keys
}

// Conservative preflight bounds a single record before encoding/json allocates
// its buffer. Huge vectors/histories/strings fail explicitly instead of defeating
// the page budget. Canonical records contain no arbitrary custom marshalers.
func inspectionFits(v reflect.Value, budget *int) bool {
	if !v.IsValid() {
		*budget -= 4
		return *budget >= 0
	}
	if v.Kind() == reflect.Interface || v.Kind() == reflect.Pointer {
		if v.IsNil() {
			*budget -= 4
			return *budget >= 0
		}
		return inspectionFits(v.Elem(), budget)
	}
	if v.Type() == reflect.TypeOf(time.Time{}) {
		*budget -= 64
		return *budget >= 0
	}
	switch v.Kind() {
	case reflect.String:
		*budget -= 6*v.Len() + 2
	case reflect.Array, reflect.Slice:
		*budget -= 2 + v.Len()
		if *budget < 0 {
			return false
		}
		for i := 0; i < v.Len(); i++ {
			if !inspectionFits(v.Index(i), budget) {
				return false
			}
		}
	case reflect.Map:
		*budget -= 2 + v.Len()
		if *budget < 0 {
			return false
		}
		iter := v.MapRange()
		for iter.Next() {
			if !inspectionFits(iter.Key(), budget) || !inspectionFits(iter.Value(), budget) {
				return false
			}
		}
	case reflect.Struct:
		*budget -= 2
		for i := 0; i < v.NumField(); i++ {
			f := v.Type().Field(i)
			if f.PkgPath != "" || f.Tag.Get("json") == "-" {
				continue
			}
			*budget -= len(f.Name)*6 + 4
			if *budget < 0 || !inspectionFits(v.Field(i), budget) {
				return false
			}
		}
	default:
		*budget -= 32
	}
	return *budget >= 0
}
func inspectionJSON(value any) (json.RawMessage, error) {
	budget := MaxInspectionBytes - 16*1024
	if !inspectionFits(reflect.ValueOf(value), &budget) {
		return nil, ErrInspectionTooLarge
	}
	data, err := json.Marshal(value)
	if len(data) > MaxInspectionBytes-16*1024 {
		return nil, ErrInspectionTooLarge
	}
	return data, err
}
func inspectionToken(key inspectionKey) (string, error) {
	data, err := json.Marshal(key)
	if err != nil {
		return "", err
	}
	if len(data) > 2048 {
		return "", ErrInspectionTooLarge
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

var InspectionSections = []string{"neural_units", "synapses", "populations", "temporal_patterns", "sensory_traces", "experience_traces", "episodes", "bootstrap_experiences", "developmental_experiences", "provenance", "attention_state", "prediction_state", "error_state", "memory_state", "memory_decay", "curiosity_state", "learning_policy_state", "self_model_state", "social_model_state", "value_state", "plasticity_state", "inquiry_state", "schema_version", "brain_identity"}

// ReadInspectionPage scans indexes, retaining at most limit+1 row keys. It never
// invokes CanonicalJSON, Nodes(), All(), or creates a full graph clone. The owner
// must fence revisions around this read; returned JSON never aliases the Brain.
func (k *KnowledgeBase) ReadInspectionPage(request InspectionRequest) (InspectionData, error) {
	if k == nil {
		return InspectionData{}, ErrNilBrain
	}
	if request.Limit < 1 || request.Limit > MaxInspectionLimit {
		return InspectionData{}, fmt.Errorf("inspection limit must be 1..%d", MaxInspectionLimit)
	}
	if request.Order == "" {
		request.Order = "id"
	}
	if request.Order != "id" && !(request.Section == "neural_units" && request.Order == "activity") {
		return InspectionData{}, errors.New("unsupported inspection order")
	}
	h := &inspectionHeap{limit: request.Limit + 1, activity: request.Order == "activity"}
	if request.After != "" {
		if len(request.After) > 4096 {
			return InspectionData{}, errors.New("inspection key too long")
		}
		data, err := base64.RawURLEncoding.DecodeString(request.After)
		if err != nil {
			return InspectionData{}, errors.New("invalid inspection key")
		}
		var key inspectionKey
		if err = json.Unmarshal(data, &key); err != nil {
			return InspectionData{}, errors.New("invalid inspection key")
		}
		h.after = &key
	}
	k.RLock()
	defer k.RUnlock()
	k.projectionMu.RLock()
	defer k.projectionMu.RUnlock()
	k.Registry.mu.RLock()
	defer k.Registry.mu.RUnlock()
	k.Patterns.mu.RLock()
	defer k.Patterns.mu.RUnlock()
	counts := map[string]int{"neural_units": len(k.Registry.byID), "populations": len(k.ProjectionPopulations), "temporal_patterns": len(k.Patterns.patterns), "experience_traces": len(k.BrainState.ExperienceTraces), "sensory_traces": len(k.BrainState.SensoryTraces)}
	edgeCount := 0
	for id, node := range k.Registry.byID {
		if request.Section == "neural_units" {
			if math.IsNaN(node.Activation) || math.IsInf(node.Activation, 0) {
				return InspectionData{}, errors.New("nonfinite neural activation")
			}
			score := 0.0
			if h.activity {
				score = node.Activation
			}
			h.add(inspectionKey{Source: id, Score: score})
		}
		for bucket, list := range node.Synapses {
			for index, edge := range list {
				if edge == nil {
					continue
				}
				edgeCount++
				if request.Section == "synapses" {
					h.add(inspectionKey{Source: id, Target: edge.TargetID, Bucket: bucket, Ordinal: index, Kind: string(edge.Kind), Inhibitory: edge.Inhibitory})
				}
			}
		}
	}
	counts["synapses"] = edgeCount
	var sequence reflect.Value
	var values map[string]float64
	var single any
	var materialize func(inspectionKey) any
	s := k.BrainState
	switch request.Section {
	case "neural_units":
		materialize = func(key inspectionKey) any { return canonicalUnit(k.Registry.byID[key.Source]) }
	case "synapses":
		materialize = func(key inspectionKey) any {
			return canonicalEdge(key.Source, k.Registry.byID[key.Source].Synapses[key.Bucket][key.Ordinal])
		}
	case "temporal_patterns":
		for id := range k.Patterns.patterns {
			h.add(inspectionKey{Source: NodeID(id)})
		}
		materialize = func(key inspectionKey) any { return k.Patterns.patterns[PatternID(key.Source)] }
	case "populations", "provenance":
		sequence = reflect.ValueOf(k.ProjectionPopulations)
	case "sensory_traces":
		sequence = reflect.ValueOf(s.SensoryTraces)
	case "experience_traces":
		sequence = reflect.ValueOf(s.ExperienceTraces)
	case "episodes":
		sequence = reflect.ValueOf(s.Episodes)
	case "bootstrap_experiences":
		sequence = reflect.ValueOf(s.BootstrapExperiences)
	case "developmental_experiences":
		sequence = reflect.ValueOf(s.DevelopmentalExperiences)
	case "attention_state":
		values = s.AttentionState
	case "prediction_state", "error_state":
		values = s.PredictionState
	case "memory_state":
		values = s.MemoryState
	case "curiosity_state":
		values = s.CuriosityState
	case "self_model_state":
		values = s.SelfModelState
	case "social_model_state":
		values = s.SocialModelState
	case "value_state":
		values = s.ValueState
	case "plasticity_state":
		values = s.PlasticityState
	case "learning_policy_state":
		single = s.LearningPolicyState
	case "inquiry_state":
		single = s.InquiryState
	case "memory_decay":
		if s.MemoryDecay != nil {
			single = s.MemoryDecay
		}
	case "schema_version":
		single = 2
		if s.MemoryDecay != nil {
			single = 3
		}
	case "brain_identity":
		single = "horizon"
	default:
		return InspectionData{}, errors.New("unknown inspection section")
	}
	if sequence.IsValid() {
		for i := 0; i < sequence.Len(); i++ {
			h.add(inspectionKey{Ordinal: i})
		}
		materialize = func(key inspectionKey) any {
			if request.Section == "provenance" {
				return canonicalProvenanceAt(key.Ordinal, k.ProjectionPopulations[key.Ordinal])
			}
			return sequence.Index(key.Ordinal).Interface()
		}
	}
	if values != nil {
		for name := range values {
			if request.Section == "error_state" {
				if !strings.HasPrefix(name, "error_") {
					continue
				}
				name = strings.TrimPrefix(name, "error_")
			}
			h.add(inspectionKey{Name: name})
		}
		materialize = func(key inspectionKey) any {
			name := key.Name
			if request.Section == "error_state" {
				name = "error_" + name
			}
			return struct {
				Key   string  `json:"key"`
				Value float64 `json:"value"`
			}{key.Name, values[name]}
		}
	}
	if single != nil {
		h.add(inspectionKey{})
		materialize = func(inspectionKey) any { return single }
	}
	result := InspectionData{Section: request.Section, Order: request.Order, Items: []json.RawMessage{}, Total: h.total, Counts: counts}
	keys := h.sorted()
	budget := MaxInspectionBytes - 16*1024
	accepted := 0
	for _, key := range keys {
		if accepted == request.Limit {
			break
		}
		row, err := inspectionJSON(materialize(key))
		if err != nil {
			return InspectionData{}, err
		}
		if len(row) > budget {
			if accepted == 0 {
				return InspectionData{}, ErrInspectionTooLarge
			}
			break
		}
		result.Items = append(result.Items, row)
		budget -= len(row) + 1
		accepted++
	}
	if accepted < len(keys) {
		if accepted == 0 {
			return InspectionData{}, ErrInspectionTooLarge
		}
		next, err := inspectionToken(keys[accepted-1])
		if err != nil {
			return InspectionData{}, err
		}
		result.Next = next
	}
	if request.Section == "neural_units" {
		selected := make(map[NodeID]bool, accepted)
		for _, key := range keys[:accepted] {
			selected[key.Source] = true
		}
		edges := &inspectionHeap{limit: 1024}
		for _, key := range keys[:accepted] {
			for bucket, list := range k.Registry.byID[key.Source].Synapses {
				for index, edge := range list {
					if edge != nil && selected[edge.TargetID] {
						edges.add(inspectionKey{Source: key.Source, Target: edge.TargetID, Bucket: bucket, Ordinal: index, Kind: string(edge.Kind), Inhibitory: edge.Inhibitory})
					}
				}
			}
		}
		result.RelatedTotal = edges.total
		for _, key := range edges.sorted() {
			row, err := inspectionJSON(canonicalEdge(key.Source, k.Registry.byID[key.Source].Synapses[key.Bucket][key.Ordinal]))
			if err != nil || len(row) > budget {
				break
			}
			result.RelatedSynapses = append(result.RelatedSynapses, row)
			budget -= len(row) + 1
		}
		result.StatePreviewLimit = 16
		result.StatePreview = map[string]map[string]float64{}
		previewBytes := 0
		for _, name := range []string{"prediction", "error", "memory", "plasticity", "curiosity", "self_model"} {
			values := map[string]map[string]float64{"prediction": s.PredictionState, "error": s.PredictionState, "memory": s.MemoryState, "plasticity": s.PlasticityState, "curiosity": s.CuriosityState, "self_model": s.SelfModelState}[name]
			preview := map[string]float64{}
			selectKeys := &inspectionHeap{limit: 16}
			for key := range values {
				if name == "error" && !strings.HasPrefix(key, "error_") {
					continue
				}
				if len(key) > 128 {
					continue
				}
				selectKeys.add(inspectionKey{Name: key})
			}
			for _, key := range selectKeys.sorted() {
				label := key.Name
				if name == "error" {
					label = strings.TrimPrefix(label, "error_")
				}
				encoded, err := json.Marshal(map[string]float64{label: values[key.Name]})
				if err != nil {
					return InspectionData{}, err
				}
				if previewBytes+len(encoded) > 8192 {
					continue
				}
				previewBytes += len(encoded)
				preview[label] = values[key.Name]
			}
			result.StatePreview[name] = preview
		}
	}
	return result, nil
}
