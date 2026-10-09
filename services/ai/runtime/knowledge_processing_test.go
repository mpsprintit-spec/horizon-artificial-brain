package runtime

import (
	"fmt"
	"strings"
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
	// Replay the same ordered surface experience under a new event/document
	// identity. Existing representations should be reused and at least one
	// previously established transition should be reinforced, not only added.
	weightsBeforeRepeat := testSynapseWeights(brain)
	repeatDoc := doc
	repeatDoc.ID = "runtime-knowledge-smoke-repeat"
	repeat, err := runtime.ProcessKnowledgeDocument(repeatDoc, at.Add(time.Second))
	if err != nil {
		t.Fatalf("repeat experience: %v", err)
	}
	if len(repeat.Outputs) != 3 {
		t.Fatalf("repeat runtime outputs = %d, want 3", len(repeat.Outputs))
	}
	weightsAfterRepeat := testSynapseWeights(brain)
	changedExistingWeight := false
	for key, beforeWeight := range weightsBeforeRepeat {
		if afterWeight, ok := weightsAfterRepeat[key]; ok && afterWeight != beforeWeight {
			changedExistingWeight = true
			break
		}
	}
	if !changedExistingWeight {
		t.Fatal("repeated experience did not change any existing synapse weight")
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

func testSynapseWeights(brain *knowledge.Brain) map[string]float64 {
	weights := make(map[string]float64)
	if brain == nil || brain.Registry == nil {
		return weights
	}
	for _, source := range brain.Registry.Nodes() {
		for _, synapse := range source.OutboundAll() {
			if synapse == nil {
				continue
			}
			key := fmt.Sprintf("%d:%d:%t", source.ID, synapse.TargetID, synapse.Inhibitory)
			weights[key] = synapse.Dynamic.Weight
		}
	}
	return weights
}


func TestProcessKnowledgeFrameCorpusUsesCanonicalRuntime(t *testing.T) {
	input := `{
		"schema_version":"horizon.knowledge-frame.v1",
		"corpus_id":"runtime-frame-test",
		"purpose":"runtime integration test",
		"records":[{
			"schema_version":"horizon.knowledge-frame.v1",
			"id":"door-state-transition",
			"domain":"language",
			"kind":"lexical_sense",
			"status":"model_synthesized_unverified",
			"content":{
				"summary":"A door changes from closed to open.",
				"state_model":{
					"initial_state":["door is closed"],
					"mechanism_or_transition":["agent moves the door"],
					"resulting_state":["door is open"]
				}
			},
			"epistemic":{"basis":"model_synthesis","confidence":0.6}
		}]
	}`
	runtime := NewBrainRuntime(knowledge.NewBootstrapBrain())
	at := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	results, err := runtime.ProcessKnowledgeFrameCorpus(strings.NewReader(input), at)
	if err != nil { t.Fatal(err) }
	if len(results) != 1 { t.Fatalf("results=%d, want 1", len(results)) }
	if len(results[0].Compilation.Experiences) != 1 { t.Fatalf("compiled experiences=%d, want 1", len(results[0].Compilation.Experiences)) }
	if len(results[0].Outputs) != 3 { t.Fatalf("runtime outputs=%d, want 3", len(results[0].Outputs)) }
	for i, output := range results[0].Outputs {
		if len(output.Activations) == 0 { t.Fatalf("output %d has no activations", i) }
	}
}
