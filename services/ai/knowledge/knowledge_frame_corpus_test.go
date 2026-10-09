package knowledge

import (
	"strings"
	"testing"
	"time"
	"encoding/json"
)

func TestDecodeKnowledgeFrameCorpusPreservesFramesThroughCompilation(t *testing.T) {
	input := `{
		"schema_version":"horizon.knowledge-frame.v1",
		"corpus_id":"test-corpus",
		"purpose":"parser test",
		"records":[{
			"schema_version":"horizon.knowledge-frame.v1",
			"id":"test-state-change",
			"domain":"language",
			"kind":"lexical_sense",
			"status":"model_synthesized_unverified",
			"content":{
				"summary":"A state-changing verb",
				"state_model":{
					"initial_state":["door closed"],
					"mechanism_or_transition":["agent moves door"],
					"resulting_state":["door open"]
				}
			},
			"epistemic":{"basis":"model_synthesis","confidence":0.6}
		}]
	}`
	corpus, err := DecodeKnowledgeFrameCorpus(strings.NewReader(input))
	if err != nil { t.Fatal(err) }
	documents, err := corpus.KnowledgeDocuments()
	if err != nil { t.Fatal(err) }
	if len(documents) != 1 { t.Fatalf("documents=%d, want 1", len(documents)) }
	compiled, err := CompileKnowledgeDocument(documents[0], time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC))
	if err != nil { t.Fatal(err) }
	if len(compiled.Experiences) != 1 { t.Fatalf("experiences=%d, want 1", len(compiled.Experiences)) }
	exp := compiled.Experiences[0]
	if len(exp.SymbolExposures) != 3 { t.Fatalf("exposures=%d, want 3", len(exp.SymbolExposures)) }
	for i, exposure := range exp.SymbolExposures {
		if exposure.SequencePosition != i { t.Fatalf("exposure %d sequence=%d, want %d", i, exposure.SequencePosition, i) }
		if !json.Valid(exposure.SemanticFrame) { t.Fatalf("exposure %d lost its semantic frame", i) }
		var frame map[string]any
		if err := json.Unmarshal(exposure.SemanticFrame, &frame); err != nil { t.Fatal(err) }
		if frame["id"] != "test-state-change" { t.Fatalf("exposure %d frame id=%v", i, frame["id"]) }
	}
	if exp.Provenance.Status != "model_synthesized_unverified" { t.Fatalf("status lost: %q", exp.Provenance.Status) }
}

func TestDecodeKnowledgeFrameCorpusRejectsDuplicateIDs(t *testing.T) {
	input := `{
		"schema_version":"horizon.knowledge-frame.v1",
		"corpus_id":"test-corpus",
		"purpose":"duplicate test",
		"records":[
			{"schema_version":"horizon.knowledge-frame.v1","id":"same","domain":"math","kind":"fact","content":{"summary":"one"},"epistemic":{"basis":"model_synthesis","confidence":0.5}},
			{"schema_version":"horizon.knowledge-frame.v1","id":"same","domain":"math","kind":"fact","content":{"summary":"two"},"epistemic":{"basis":"model_synthesis","confidence":0.5}}
		]
	}`
	if _, err := DecodeKnowledgeFrameCorpus(strings.NewReader(input)); err == nil { t.Fatal("expected duplicate ID error") }
}
