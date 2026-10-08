package knowledge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCommunicationDevelopmentalBootstrapIsRevisable(t *testing.T) {
	brain := NewBootstrapBrain()
	if got := len(brain.BrainState.DevelopmentalExperiences); got != 5 {
		t.Fatalf("developmental communication experiences = %d, want 5", got)
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
	if len(persisted.DevelopmentalExperiences) != 5 {
		t.Fatalf("persisted developmental experiences = %d, want 5", len(persisted.DevelopmentalExperiences))
	}
	loaded := NewBrain()
	if err := loaded.Load(path); err != nil { t.Fatal(err) }
	if len(loaded.BrainState.DevelopmentalExperiences) != 5 {
		t.Fatalf("reloaded developmental experiences = %d, want 5", len(loaded.BrainState.DevelopmentalExperiences))
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
