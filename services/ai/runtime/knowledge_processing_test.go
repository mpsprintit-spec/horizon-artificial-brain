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
	beforeSynapses := countTestSynapses(brain)
	at := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	result, err := runtime.ProcessKnowledgeDocument(doc, at)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Compilation.Experiences) != 1 {
		t.Fatalf("compiled experiences = %d, want 1", len(result.Compilation.Experiences))
	}
	if len(result.Outputs) != 3 {
		t.Fatalf("runtime outputs = %d, want 3 (before/change/after)", len(result.Outputs))
	}
	for i, output := range result.Outputs {
		if len(output.Activations) == 0 {
			t.Fatalf("runtime output %d has no neural activations", i)
		}
		if output.Sequence == 0 {
			t.Fatalf("runtime output %d has zero sequence", i)
		}
		if i > 0 && output.Sequence <= result.Outputs[i-1].Sequence {
			t.Fatalf("runtime sequences are not increasing: %d then %d", result.Outputs[i-1].Sequence, output.Sequence)
		}
	}
	afterSynapses := countTestSynapses(brain)
	if afterSynapses <= beforeSynapses {
		t.Fatalf("synapse count did not increase after sequential experience: before=%d after=%d", beforeSynapses, afterSynapses)
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

func countTestSynapses(brain *knowledge.Brain) int {
	if brain == nil || brain.Registry == nil {
		return 0
	}
	count := 0
	for _, node := range brain.Registry.Nodes() {
		count += len(node.OutboundAll())
	}
	return count
}
