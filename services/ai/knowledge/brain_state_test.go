package knowledge_test

import (
	"path/filepath"
	"time"
	"path/filepath"
	"testing"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestBrainStatePersistsBootstrapAndLearningState(t *testing.T) {
	brain := knowledge.NewBrain()
	brain.BrainState.BootstrapExperiences = []knowledge.BootstrapExperience{{
		ID: "bootstrap-test",
		Populations: []knowledge.NodeID{1, 2, 3, 4},
		Activation: 0.35,
		Confidence: 0.10,
		Provenance: knowledge.BootstrapProvenance{
			Origin: "bootstrap",
			DirectExperience: false,
			Status: "initial_hypothesis",
		},
	}}
	brain.BrainState.LearningPolicyState.MemoryRetention = 0.73
	brain.BrainState.CuriosityState["pressure"] = 0.65

	path := filepath.Join(t.TempDir(), "brain_memory.json")
	if err := brain.Save(path); err != nil {
		t.Fatalf("save brain: %v", err)
	}

	restored := knowledge.NewBrain()
	if err := restored.Load(path); err != nil {
		t.Fatalf("load brain: %v", err)
	}

	if len(restored.BrainState.BootstrapExperiences) != 1 {
		t.Fatalf("bootstrap experiences were not restored: %+v", restored.BrainState.BootstrapExperiences)
	}
	if restored.BrainState.BootstrapExperiences[0].ID != "bootstrap-test" {
		t.Fatalf("unexpected bootstrap experience: %+v", restored.BrainState.BootstrapExperiences[0])
	}
	if restored.BrainState.LearningPolicyState.MemoryRetention != 0.73 {
		t.Fatalf("learning policy state was not restored: %+v", restored.BrainState.LearningPolicyState)
	}
	if restored.BrainState.CuriosityState["pressure"] != 0.65 {
		t.Fatalf("curiosity state was not restored: %+v", restored.BrainState.CuriosityState)
	}
}


func TestInquiryInformationExperiencePersistsAcrossCanonicalSaveLoad(t *testing.T) {
	now := time.Date(2026, 10, 3, 15, 0, 0, 0, time.UTC)
	brain := knowledge.NewBrain()
	brain.RecordInquiryInformationExperience("focus", 0.8, 0.9, now)
	beforeYield, beforeReliability, beforeSamples, learned := brain.InquiryInformationExperience("focus")
	if !learned {
		t.Fatal("expected inquiry information experience to be learned")
	}
	path := filepath.Join(t.TempDir(), "brain_memory.json")
	if err := brain.Save(path); err != nil {
		t.Fatal(err)
	}
	restored := knowledge.NewBrain()
	if err := restored.Load(path); err != nil {
		t.Fatal(err)
	}
	afterYield, afterReliability, afterSamples, restoredLearned := restored.InquiryInformationExperience("focus")
	if !restoredLearned {
		t.Fatal("inquiry information experience was not persisted")
	}
	if beforeYield != afterYield || beforeReliability != afterReliability || beforeSamples != afterSamples {
		t.Fatalf("inquiry information experience changed across save/load: before=(%v,%v,%v) after=(%v,%v,%v)", beforeYield, beforeReliability, beforeSamples, afterYield, afterReliability, afterSamples)
	}
}
