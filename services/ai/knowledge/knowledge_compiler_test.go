package knowledge

import (
	"testing"
	"time"
)

func sampleKnowledgeDocument() KnowledgeDocument {
	return KnowledgeDocument{
		ID: "physics-falling-body",
		Domain: "physics",
		Source: "research-note-001",
		SourceType: "human_supplied_hypothesis",
		Status: "initial_hypothesis",
		DirectExperience: false,
		Experiences: []KnowledgeExperienceInput{{
			ID: "falling-body-state-transition",
			Before: []KnowledgeDatum{{Value: "body at height", Modality: "text", Confidence: 0.3}},
			Changes: []KnowledgeDatum{{Value: "position changes over time", Modality: "text", Confidence: 0.3}},
			After: []KnowledgeDatum{{Value: "body closer to surface", Modality: "text", Confidence: 0.3}},
			TemporalDeltasNanos: []int64{25000000},
			Confidence: 0.3,
		}},
	}
}

func TestCompileKnowledgeDocumentProducesCanonicalExperience(t *testing.T) {
	doc := sampleKnowledgeDocument()
	at := time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC)
	got, err := CompileKnowledgeDocument(doc, at)
	if err != nil { t.Fatal(err) }
	if got.DocumentID != doc.ID || got.Domain != doc.Domain || len(got.SourceHash) != 64 {
		t.Fatalf("unexpected compilation metadata: %+v", got)
	}
	if len(got.Experiences) != 1 { t.Fatalf("experiences = %d, want 1", len(got.Experiences)) }
	exp := got.Experiences[0]
	if len(exp.BeforeState.Representation) == 0 || len(exp.Transition.ChangeRepresentation) == 0 || len(exp.AfterState.Representation) == 0 {
		t.Fatalf("compiled experience is missing state/transition representations: %+v", exp)
	}
	if len(exp.SymbolExposures) != 3 { t.Fatalf("symbol exposures = %d, want 3", len(exp.SymbolExposures)) }
	if exp.Provenance.DirectExperience || exp.Provenance.Status != "initial_hypothesis" || exp.Provenance.SourceHash != got.SourceHash {
		t.Fatalf("provenance not preserved: %+v", exp.Provenance)
	}
	if exp.SymbolExposures[0].Modality != "text" || len(exp.SymbolExposures[0].Representation) == 0 {
		t.Fatalf("symbol exposure was not numerically encoded: %+v", exp.SymbolExposures[0])
	}
	again, err := CompileKnowledgeDocument(doc, at)
	if err != nil { t.Fatal(err) }
	if again.SourceHash != got.SourceHash { t.Fatalf("source hash changed: %s != %s", again.SourceHash, got.SourceHash) }
}

func TestImportKnowledgeDocumentIsIdempotent(t *testing.T) {
	brain := NewBootstrapBrain()
	before := len(brain.BrainState.DevelopmentalExperiences)
	doc := sampleKnowledgeDocument()
	if _, err := brain.ImportKnowledgeDocument(doc, time.Time{}); err != nil { t.Fatal(err) }
	afterFirst := len(brain.BrainState.DevelopmentalExperiences)
	if afterFirst != before+1 { t.Fatalf("after first import = %d, want %d", afterFirst, before+1) }
	if _, err := brain.ImportKnowledgeDocument(doc, time.Time{}); err != nil { t.Fatal(err) }
	afterSecond := len(brain.BrainState.DevelopmentalExperiences)
	if afterSecond != afterFirst { t.Fatalf("duplicate import changed count: %d -> %d", afterFirst, afterSecond) }
}

func TestCompileKnowledgeDocumentRejectsMissingProvenanceAndEmptyExperience(t *testing.T) {
	if _, err := CompileKnowledgeDocument(KnowledgeDocument{}, time.Time{}); err == nil {
		t.Fatal("expected missing metadata error")
	}
	doc := sampleKnowledgeDocument()
	doc.Experiences = []KnowledgeExperienceInput{{ID: "empty"}}
	if _, err := CompileKnowledgeDocument(doc, time.Time{}); err == nil {
		t.Fatal("expected empty experience error")
	}
}
