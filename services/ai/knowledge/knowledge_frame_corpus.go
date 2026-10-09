package knowledge

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

// KnowledgeFrameCorpus is the authored cross-domain corpus envelope. It keeps
// the full source record as raw JSON so fields not yet mapped into runtime
// semantics are not silently discarded.
type KnowledgeFrameCorpus struct {
	SchemaVersion string                  `json:"schema_version"`
	CorpusID      string                  `json:"corpus_id"`
	Purpose       string                  `json:"purpose"`
	Records       []KnowledgeFrameRecord  `json:"records"`
}

// KnowledgeFrameRecord contains the fields needed to create a developmental
// experience. Raw retains the complete record, including domain-specific data.
type KnowledgeFrameRecord struct {
	SchemaVersion string                     `json:"schema_version"`
	ID            string                     `json:"id"`
	Domain        string                     `json:"domain"`
	Subdomain     string                     `json:"subdomain,omitempty"`
	Kind          string                     `json:"kind"`
	Status        string                     `json:"status,omitempty"`
	Content       KnowledgeFrameContent      `json:"content"`
	Epistemic     KnowledgeFrameEpistemic    `json:"epistemic"`
	Raw           json.RawMessage            `json:"-"`
}

type KnowledgeFrameContent struct {
	Summary    string                  `json:"summary"`
	StateModel KnowledgeFrameStateModel `json:"state_model"`
}

type KnowledgeFrameStateModel struct {
	InitialState         []string `json:"initial_state,omitempty"`
	TriggerOrConditions  []string `json:"trigger_or_conditions,omitempty"`
	MechanismOrTransition []string `json:"mechanism_or_transition,omitempty"`
	ResultingState       []string `json:"resulting_state,omitempty"`
	TemporalOrder        []string `json:"temporal_order,omitempty"`
	Invariants           []string `json:"invariants,omitempty"`
	Unknowns             []string `json:"unknowns,omitempty"`
}

type KnowledgeFrameEpistemic struct {
	Basis           string    `json:"basis"`
	Confidence      float64   `json:"confidence"`
	EvidenceSummary string    `json:"evidence_summary,omitempty"`
	SourceIDs       []string  `json:"source_ids,omitempty"`
	SourceURLs      []string  `json:"source_urls,omitempty"`
	Prediction      []float64 `json:"prediction,omitempty"`
	Outcome         []float64 `json:"outcome,omitempty"`
	PredictionError float64   `json:"prediction_error,omitempty"`
}

// DecodeKnowledgeFrameCorpus parses and checks the minimum structural
// requirements of a corpus. It does not replace full JSON Schema validation
// and does not verify the truth of the records.
func DecodeKnowledgeFrameCorpus(reader io.Reader) (KnowledgeFrameCorpus, error) {
	if reader == nil {
		return KnowledgeFrameCorpus{}, errors.New("knowledge frame corpus reader is nil")
	}
	var envelope struct {
		SchemaVersion string            `json:"schema_version"`
		CorpusID      string            `json:"corpus_id"`
		Purpose       string            `json:"purpose"`
		Records       []json.RawMessage `json:"records"`
	}
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&envelope); err != nil {
		return KnowledgeFrameCorpus{}, fmt.Errorf("decode knowledge frame corpus: %w", err)
	}
	if envelope.SchemaVersion != "horizon.knowledge-frame.v1" {
		return KnowledgeFrameCorpus{}, fmt.Errorf("unsupported knowledge frame schema_version %q", envelope.SchemaVersion)
	}
	if strings.TrimSpace(envelope.CorpusID) == "" || strings.TrimSpace(envelope.Purpose) == "" {
		return KnowledgeFrameCorpus{}, errors.New("knowledge frame corpus requires corpus_id and purpose")
	}
	if len(envelope.Records) == 0 {
		return KnowledgeFrameCorpus{}, errors.New("knowledge frame corpus requires at least one record")
	}
	corpus := KnowledgeFrameCorpus{SchemaVersion: envelope.SchemaVersion, CorpusID: envelope.CorpusID, Purpose: envelope.Purpose}
	seen := make(map[string]bool, len(envelope.Records))
	for i, raw := range envelope.Records {
		var record KnowledgeFrameRecord
		if err := json.Unmarshal(raw, &record); err != nil {
			return KnowledgeFrameCorpus{}, fmt.Errorf("decode knowledge frame record %d: %w", i, err)
		}
		record.ID = strings.TrimSpace(record.ID)
		record.Domain = strings.TrimSpace(record.Domain)
		record.Content.Summary = strings.TrimSpace(record.Content.Summary)
		if record.SchemaVersion != envelope.SchemaVersion || record.ID == "" || record.Domain == "" || record.Kind == "" || record.Content.Summary == "" {
			return KnowledgeFrameCorpus{}, fmt.Errorf("record %d requires matching schema_version, id, domain, kind, and content.summary", i)
		}
		if seen[record.ID] {
			return KnowledgeFrameCorpus{}, fmt.Errorf("duplicate knowledge frame id %q", record.ID)
		}
		if record.Epistemic.Confidence < 0 || record.Epistemic.Confidence > 1 {
			return KnowledgeFrameCorpus{}, fmt.Errorf("record %q confidence outside [0,1]", record.ID)
		}
		seen[record.ID] = true
		record.Raw = append(json.RawMessage(nil), raw...)
		corpus.Records = append(corpus.Records, record)
	}
	return corpus, nil
}

// KnowledgeDocuments converts each frame to a source-described document while
// preserving the complete frame on each generated datum. The runtime can learn
// ordered state summaries; preserving raw metadata does not mean the neural
// substrate has inferred or validated every semantic field.
func (c KnowledgeFrameCorpus) KnowledgeDocuments() ([]KnowledgeDocument, error) {
	if c.SchemaVersion != "horizon.knowledge-frame.v1" || strings.TrimSpace(c.CorpusID) == "" || len(c.Records) == 0 {
		return nil, errors.New("knowledge frame corpus is not initialized")
	}
	documents := make([]KnowledgeDocument, 0, len(c.Records))
	for _, record := range c.Records {
		raw := append(json.RawMessage(nil), record.Raw...)
		if len(raw) == 0 {
			var err error
			raw, err = json.Marshal(record)
			if err != nil { return nil, fmt.Errorf("marshal knowledge frame %q: %w", record.ID, err) }
		}
		makeDatum := func(parts []string, relative string) []KnowledgeDatum {
			value := strings.TrimSpace(strings.Join(parts, "; "))
			if value == "" { return nil }
			return []KnowledgeDatum{{Value: value, Modality: "knowledge-frame", RelativePosition: relative, Confidence: record.Epistemic.Confidence, SemanticFrame: raw}}
		}
		before := makeDatum(record.Content.StateModel.InitialState, "before")
		changes := makeDatum(record.Content.StateModel.MechanismOrTransition, "transition")
		after := makeDatum(record.Content.StateModel.ResultingState, "after")
		if len(before)+len(changes)+len(after) == 0 {
			changes = makeDatum([]string{record.Content.Summary}, "description")
		}
		if len(before)+len(changes)+len(after) == 0 {
			return nil, fmt.Errorf("knowledge frame %q has no usable content", record.ID)
		}
		status := record.Status
		if status == "" { status = "needs_review" }
		basis := record.Epistemic.Basis
		if basis == "" { basis = "unknown" }
		documents = append(documents, KnowledgeDocument{
			ID: record.ID,
			Domain: record.Domain,
			Source: c.CorpusID + "/" + record.ID,
			SourceType: "knowledge-frame-v1:" + basis,
			Status: status,
			DirectExperience: false,
			Experiences: []KnowledgeExperienceInput{{
				ID: record.ID,
				Before: before,
				Changes: changes,
				After: after,
				Prediction: append([]float64(nil), record.Epistemic.Prediction...),
				Outcome: append([]float64(nil), record.Epistemic.Outcome...),
				PredictionError: record.Epistemic.PredictionError,
				Confidence: record.Epistemic.Confidence,
			}},
		})
	}
	return documents, nil
}
