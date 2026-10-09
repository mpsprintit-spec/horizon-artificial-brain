package knowledge

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// KnowledgeDocument is a domain-neutral envelope for source-described knowledge.
// It preserves supplied structure and provenance without inferring semantic facts.
type KnowledgeDocument struct {
	ID string `json:"id"`
	Domain string `json:"domain"`
	Source string `json:"source"`
	SourceType string `json:"source_type"`
	Status string `json:"status"`
	DirectExperience bool `json:"direct_experience"`
	Experiences []KnowledgeExperienceInput `json:"experiences"`
}

type KnowledgeExperienceInput struct {
	ID string `json:"id,omitempty"`
	Before []KnowledgeDatum `json:"before,omitempty"`
	Changes []KnowledgeDatum `json:"changes,omitempty"`
	After []KnowledgeDatum `json:"after,omitempty"`
	TemporalDeltasNanos []int64 `json:"temporal_deltas_nanos,omitempty"`
	Prediction []float64 `json:"prediction,omitempty"`
	Outcome []float64 `json:"outcome,omitempty"`
	PredictionError float64 `json:"prediction_error,omitempty"`
	Plasticity float64 `json:"plasticity,omitempty"`
	Confidence float64 `json:"confidence,omitempty"`
}

type KnowledgeDatum struct {
	Value string `json:"value"`
	Modality string `json:"modality"`
	RelativePosition string `json:"relative_position,omitempty"`
	Confidence float64 `json:"confidence,omitempty"`
}

type KnowledgeCompilation struct {
	DocumentID string `json:"document_id"`
	Domain string `json:"domain"`
	SourceHash string `json:"source_hash"`
	Experiences []DevelopmentalExperience `json:"experiences"`
}

// CompileKnowledgeDocument maps explicitly supplied before/change/after data
// into the canonical developmental schema. Compilation is not proof of truth
// and does not by itself constitute runtime learning.
func CompileKnowledgeDocument(document KnowledgeDocument, at time.Time) (KnowledgeCompilation, error) {
	document.ID = strings.TrimSpace(document.ID)
	document.Domain = strings.TrimSpace(document.Domain)
	document.Source = strings.TrimSpace(document.Source)
	document.SourceType = strings.TrimSpace(document.SourceType)
	if document.ID == "" || document.Domain == "" || document.Source == "" || document.SourceType == "" {
		return KnowledgeCompilation{}, errors.New("knowledge document requires id, domain, source, and source_type")
	}
	if len(document.Experiences) == 0 {
		return KnowledgeCompilation{}, errors.New("knowledge document requires at least one experience")
	}
	if document.Status == "" { document.Status = "unverified" }
	canonical, err := json.Marshal(document)
	if err != nil { return KnowledgeCompilation{}, err }
	sum := sha256.Sum256(canonical)
	hash := hex.EncodeToString(sum[:])
	if at.IsZero() { at = time.Unix(1767225600, 0).UTC() } else { at = at.UTC() }
	out := KnowledgeCompilation{DocumentID: document.ID, Domain: document.Domain, SourceHash: hash}
	seen := map[string]bool{}
	for i, in := range document.Experiences {
		id := strings.TrimSpace(in.ID)
		if id == "" { id = fmt.Sprintf("%s-experience-%03d", document.ID, i+1) }
		if seen[id] { return KnowledgeCompilation{}, fmt.Errorf("duplicate experience id %q", id) }
		seen[id] = true
		before, be, err := compileKnowledgeData(in.Before, "before", in.Confidence)
		if err != nil { return KnowledgeCompilation{}, fmt.Errorf("%s before: %w", id, err) }
		changes, ce, err := compileKnowledgeData(in.Changes, "transition", in.Confidence)
		if err != nil { return KnowledgeCompilation{}, fmt.Errorf("%s changes: %w", id, err) }
		after, ae, err := compileKnowledgeData(in.After, "after", in.Confidence)
		if err != nil { return KnowledgeCompilation{}, fmt.Errorf("%s after: %w", id, err) }
		if len(before)+len(changes)+len(after) == 0 { return KnowledgeCompilation{}, fmt.Errorf("%s has no experience data", id) }
		confidence := in.Confidence
		if confidence == 0 { confidence = 0.15 }
		plasticity := in.Plasticity
		if plasticity == 0 { plasticity = 0.1 }
		exposures := append(append(be, ce...), ae...)
		out.Experiences = append(out.Experiences, DevelopmentalExperience{
			ID: id,
			BeforeState: DevelopmentalStateFrame{Representation: averageKnowledgeVectors(before), Confidence: confidence},
			Transition: DevelopmentalTransition{ChangeRepresentation: averageKnowledgeVectors(changes), TemporalDeltasNanos: append([]int64(nil), in.TemporalDeltasNanos...), Confidence: confidence},
			AfterState: DevelopmentalStateFrame{Representation: averageKnowledgeVectors(after), Confidence: confidence},
			SymbolExposures: exposures,
			LearningTrace: DevelopmentalLearningTrace{TemporalTrace: []time.Time{at}, Prediction: append([]float64(nil), in.Prediction...), Outcome: append([]float64(nil), in.Outcome...), PredictionError: in.PredictionError, Plasticity: plasticity, Confidence: confidence},
			Provenance: BootstrapProvenance{Origin: document.SourceType + ":" + document.Source + ":domain=" + document.Domain, DirectExperience: document.DirectExperience, Status: document.Status, SourceHash: hash},
		})
	}
	return out, nil
}

// ImportKnowledgeDocument adds compiled experiences to the canonical store.
// Importing the same experience ID twice is idempotent; it does not create
// synapses or claim that the runtime has learned the imported content.
func (k *KnowledgeBase) ImportKnowledgeDocument(document KnowledgeDocument, at time.Time) (KnowledgeCompilation, error) {
	if k == nil { return KnowledgeCompilation{}, ErrNilBrain }
	compiled, err := CompileKnowledgeDocument(document, at)
	if err != nil { return KnowledgeCompilation{}, err }
	k.mu.Lock()
	defer k.mu.Unlock()
	existing := make(map[string]bool, len(k.BrainState.DevelopmentalExperiences))
	for _, exp := range k.BrainState.DevelopmentalExperiences { existing[exp.ID] = true }
	for _, exp := range compiled.Experiences {
		if !existing[exp.ID] {
			k.BrainState.DevelopmentalExperiences = append(k.BrainState.DevelopmentalExperiences, exp)
			existing[exp.ID] = true
		}
	}
	return compiled, nil
}

func compileKnowledgeData(data []KnowledgeDatum, position string, fallback float64) ([][]float64, []SymbolExposure, error) {
	vectors := make([][]float64, 0, len(data))
	exposures := make([]SymbolExposure, 0, len(data))
	for i, item := range data {
		value, modality := strings.TrimSpace(item.Value), strings.TrimSpace(item.Modality)
		if value == "" || modality == "" { return nil, nil, fmt.Errorf("datum %d requires value and modality", i) }
		confidence := item.Confidence
		if confidence == 0 { confidence = fallback }
		if confidence == 0 { confidence = 0.15 }
		if confidence < 0 || confidence > 1 { return nil, nil, fmt.Errorf("datum %d confidence outside [0,1]", i) }
		vector := EncodeObservation(value, modality).Values
		vectors = append(vectors, append([]float64(nil), vector...))
		relative := strings.TrimSpace(item.RelativePosition)
		if relative == "" { relative = position }
		exposures = append(exposures, SymbolExposure{Symbol: value, Modality: modality, Representation: append([]float64(nil), vector...), RelativePosition: relative, SequencePosition: i, Confidence: confidence})
	}
	return vectors, exposures, nil
}

func averageKnowledgeVectors(vectors [][]float64) []float64 {
	if len(vectors) == 0 { return nil }
	out := make([]float64, len(vectors[0]))
	for _, v := range vectors {
		if len(v) != len(out) { return nil }
		for i, x := range v { out[i] += x }
	}
	for i := range out { out[i] /= float64(len(vectors)) }
	return out
}
