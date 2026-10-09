package runtime

import (
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestProcessKnowledgeDocumentUsesCanonicalRuntime(t *testing.T) {
	brain := knowledge.NewBootstrapBrain()
	runtime := NewBrainRuntime(brain)
	if runtime == nil {
		t.Fatal("runtime initialization failed")
	}
	doc := knowledge.KnowledgeDocument{
		ID: "runtime-knowledge-smoke",
		Domain: "physics",
		Source: "controlled-test",
		SourceType: "test_input",
		Status: "initial_hypothesis",
		Experiences: []knowledge.KnowledgeExperienceInput{
			{
				ID: "state-before",
				Before: []knowledge.KnowledgeDatum{{Value: "body at height", Modality: "text", Confidence: 0.2}},
				Changes: []knowledge.KnowledgeDatum{{Value: "position changes", Modality: "text", Confidence: 0.2}},
				After: []knowledge.KnowledgeDatum{{Value: "body lower", Modality: "text", Confidence: 0.2}},
				Confidence: 0.2,
			},
		},
	}
	at := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	result, err := runtime.ProcessKnowledgeDocument(doc, at)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Compilation.Experiences) != 1 {
		t.Fatalf("compiled experiences = %d, want 1", len(result.Compilation.Experiences))
	}
	if len(result.Outputs) != 1 {
		t.Fatalf("runtime outputs = %d, want 1", len(result.Outputs))
	}
	if result.Outputs[0].Sequence == 0 {
		t.Fatal("runtime did not advance its sequence")
	}
	found := false
	for _, experience := range brain.BrainState.DevelopmentalExperiences {
		if experience.ID == "state-before" {
			found = true
			if experience.Provenance.SourceHash != result.Compilation.SourceHash {
				t.Fatal("persisted provenance hash does not match compilation")
			}
			break
		}
	}
	if !found {
		t.Fatal("compiled experience was not imported into canonical brain state")
	}
}
