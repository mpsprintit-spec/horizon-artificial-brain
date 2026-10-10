package knowledge

import (
	"bytes"
	"os"
	"testing"
	"time"
)

func TestSeedKnowledgeCorpusCompilesWithFrameMetadata(t *testing.T) {
	data, err := os.ReadFile("../../../knowledge/corpus/seed-v1.json")
	if err != nil { t.Fatal(err) }
	corpus, err := DecodeKnowledgeFrameCorpus(bytes.NewReader(data))
	if err != nil { t.Fatal(err) }
	if len(corpus.Records) != 44 { t.Fatalf("records=%d, want 44", len(corpus.Records)) }
	documents, err := corpus.KnowledgeDocuments()
	if err != nil { t.Fatal(err) }
	if len(documents) != len(corpus.Records) { t.Fatalf("documents=%d, records=%d", len(documents), len(corpus.Records)) }
	seen := map[string]bool{}
	domains := map[string]bool{}
	for _, document := range documents {
		if seen[document.ID] { t.Fatalf("duplicate document id %q", document.ID) }
		seen[document.ID] = true
		domains[document.Domain] = true
		compiled, err := CompileKnowledgeDocument(document, time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC))
		if err != nil { t.Fatalf("compile %q: %v", document.ID, err) }
		if len(compiled.Experiences) != 1 { t.Fatalf("%q experiences=%d, want 1", document.ID, len(compiled.Experiences)) }
		for _, exposure := range compiled.Experiences[0].SymbolExposures {
			if len(exposure.SemanticFrame) == 0 { t.Fatalf("%q exposure %q lost frame metadata", document.ID, exposure.Symbol) }
		}
	}
	for _, domain := range []string{"language", "mathematics", "physical-sciences", "biology-neuroscience", "computing-engineering", "empirical-reasoning"} {
		if !domains[domain] { t.Errorf("seed corpus is missing domain %q", domain) }
	}
}
