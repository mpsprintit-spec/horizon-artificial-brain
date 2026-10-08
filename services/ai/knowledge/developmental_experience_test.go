package knowledge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCommunicationDevelopmentalBootstrapIsRevisable(t *testing.T) {
	brain := NewBootstrapBrain()
	if got := len(brain.BrainState.DevelopmentalExperiences); got != 6 {
		t.Fatalf("developmental experiences = %d, want 6", got)
	}
	for _, experience := range brain.BrainState.DevelopmentalExperiences {
		if experience.Provenance.Status != "initial_hypothesis" || experience.Provenance.DirectExperience {
			t.Fatalf("%s is not marked as a revisable bootstrap hypothesis: %+v", experience.ID, experience.Provenance)
		}
		if experience.LearningTrace.Confidence > 0.5 || experience.LearningTrace.Plasticity <= 0 {
			t.Fatalf("%s has invalid confidence/plasticity: %+v", experience.ID, experience.LearningTrace)
		}
		if len(experience.BeforeState.Representation) == 0 || len(experience.AfterState.Representation) == 0 {
			t.Fatalf("%s lacks numeric state representations", experience.ID)
		}
		if len(experience.Transition.ChangeRepresentation) == 0 || len(experience.Transition.TemporalDeltasNanos) == 0 {
			t.Fatalf("%s lacks numeric transition trace", experience.ID)
		}
		if experience.ID == "bootstrap-visual-face-89220a36" {
			if len(experience.VisualExposures) != 6 || experience.Provenance.Origin != "human_provided_visual_bootstrap" {
				t.Fatalf("%s lacks visual provenance/exposure trace", experience.ID)
			}
		} else {
			if len(experience.SymbolExposures) == 0 {
				t.Fatalf("%s lacks modality exposure trace", experience.ID)
			}
			for _, exposure := range experience.SymbolExposures {
				if exposure.Modality == "" || len(exposure.Representation) == 0 {
					t.Fatalf("%s contains incomplete symbol exposure: %+v", experience.ID, exposure)
				}
			}
		}
	}
}

func TestCommunicationDevelopmentalBootstrapPersistsThroughCanonicalJSON(t *testing.T) {
	brain := NewBootstrapBrain()
	dir := t.TempDir()
	path := filepath.Join(dir, "brain_memory.json")
	if err := brain.Save(path); err != nil { t.Fatal(err) }
	raw, err := os.ReadFile(path)
	if err != nil { t.Fatal(err) }
	var persisted struct {
		DevelopmentalExperiences []DevelopmentalExperience `json:"developmental_experiences"`
	}
	if err := json.Unmarshal(raw, &persisted); err != nil { t.Fatal(err) }
	if len(persisted.DevelopmentalExperiences) != 6 {
		t.Fatalf("persisted developmental experiences = %d, want 6", len(persisted.DevelopmentalExperiences))
	}
	loaded := NewBrain()
	if err := loaded.Load(path); err != nil { t.Fatal(err) }
	if len(loaded.BrainState.DevelopmentalExperiences) != 6 {
		t.Fatalf("reloaded developmental experiences = %d, want 6", len(loaded.BrainState.DevelopmentalExperiences))
	}
}

func TestCommunicationDevelopmentalBootstrapMaterializesNeuralPathways(t *testing.T) {
	brain := NewBootstrapBrain()
	pools := brain.ProjectionPopulations
	if len(pools) < 20 {
		t.Fatalf("bootstrap populations = %d, want at least 20", len(pools))
	}
	links := [][2]NodeID{
		{pools[8].Units[0].NodeID, pools[19].Units[0].NodeID},
		{pools[19].Units[0].NodeID, pools[19].Units[1].NodeID},
		{pools[19].Units[3].NodeID, pools[9].Units[0].NodeID},
		{pools[10].Units[0].NodeID, pools[9].Units[0].NodeID},
	}
	for _, link := range links {
		source := brain.Registry.GetByID(link[0])
		if source == nil {
			t.Fatalf("missing source node %d", link[0])
		}
		if len(source.Synapses[link[1]]) == 0 {
			t.Fatalf("missing communication developmental pathway %d -> %d", link[0], link[1])
		}
	}
}


func TestVisualFaceBootstrapMaterializesRevisablePerceptualState(t *testing.T) {
	brain := NewBootstrapBrain()
	var visual *DevelopmentalExperience
	for i := range brain.BrainState.DevelopmentalExperiences {
		if brain.BrainState.DevelopmentalExperiences[i].ID == "bootstrap-visual-face-89220a36" {
			visual = &brain.BrainState.DevelopmentalExperiences[i]
			break
		}
	}
	if visual == nil {
		t.Fatal("visual face bootstrap experience missing")
	}
	if visual.Provenance.DirectExperience || visual.Provenance.Status != "initial_hypothesis" {
		t.Fatalf("visual face bootstrap is not revisable: %+v", visual.Provenance)
	}
	if visual.Provenance.SourceHash != visualBootstrapSourceHash {
		t.Fatalf("visual source hash = %q, want %q", visual.Provenance.SourceHash, visualBootstrapSourceHash)
	}
	if len(visual.VisualExposures) != 6 {
		t.Fatalf("visual exposures = %d, want 6", len(visual.VisualExposures))
	}
	if len(brain.ProjectionPopulations) != 21 || brain.ProjectionPopulations[20].Domain != "visual_form" {
		t.Fatalf("visual population missing: %d populations", len(brain.ProjectionPopulations))
	}
	if len(brain.Registry.Nodes()) != 128 {
		t.Fatalf("visual neural units = %d, want 128", len(brain.Registry.Nodes()))
	}
	if len(brain.Registry.GetByID(123).Representation) != 32 {
		t.Fatalf("visual neural representation length = %d, want 32", len(brain.Registry.GetByID(123).Representation))
	}
	if len(brain.Registry.GetByID(123).Synapses[124]) == 0 {
		t.Fatal("visual neural pathway 123 -> 124 missing")
	}
	if len(brain.Registry.GetByID(123).Synapses[19]) == 0 {
		t.Fatal("visual -> identity continuity pathway missing")
	}
}
