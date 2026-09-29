package knowledge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCanonicalBrainMemoryMaterializationSchema(t *testing.T) {
	brain := NewBootstrapBrain()
	path := filepath.Join(t.TempDir(), "brain_memory.json")
	if err := brain.Save(path); err != nil { t.Fatal(err) }
	data, err := os.ReadFile(path)
	if err != nil { t.Fatal(err) }

	var document map[string]json.RawMessage
	if err := json.Unmarshal(data, &document); err != nil { t.Fatal(err) }
	required := []string{"neural_units", "populations", "synapses", "temporal_patterns", "episodes", "bootstrap_experiences", "attention_state", "prediction_state", "error_state", "memory_state", "curiosity_state", "learning_policy_state", "self_model_state", "social_model_state", "value_state", "plasticity_state", "provenance"}
	for _, key := range required { if _, ok := document[key]; !ok { t.Fatalf("canonical brain memory is missing %q", key) } }
	assertNonEmptyArray := func(key string) { var values []json.RawMessage; if err := json.Unmarshal(document[key], &values); err != nil { t.Fatalf("%s is not an array: %v", key, err) }; if len(values) == 0 { t.Fatalf("%s is empty", key) } }
	for _, key := range []string{"neural_units", "populations", "synapses", "temporal_patterns", "episodes", "bootstrap_experiences", "provenance"} { assertNonEmptyArray(key) }

	var populations []ProjectionPopulation
	if err := json.Unmarshal(document["populations"], &populations); err != nil { t.Fatal(err) }
	for _, population := range populations {
		if len(population.Units) < 2 { t.Fatalf("population %q has too few units", population.Domain) }
		if len(population.PredictionTargets) == 0 || len(population.ErrorTargets) == 0 || len(population.PlasticityTargets) == 0 || len(population.CounterEvidenceTargets) == 0 { t.Fatalf("population %q is missing a developmental path", population.Domain) }
		if population.BootstrapExperienceID == "" || population.Provenance.Origin == "" { t.Fatalf("population %q is missing bootstrap provenance", population.Domain) }
	}
}

func TestCanonicalBrainMemoryArtifactInRepository(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok { t.Fatal("cannot locate test source") }
	root := filepath.Clean(filepath.Join(filepath.Dir(file), ".."))
	path := filepath.Join(root, "brain_memory.json")
	data, err := os.ReadFile(path)
	if err != nil { t.Fatal(err) }

	var document struct {
		SchemaVersion int `json:"schema_version"`
		NeuralUnits []json.RawMessage `json:"neural_units"`
		Populations []json.RawMessage `json:"populations"`
		Synapses []json.RawMessage `json:"synapses"`
		TemporalPatterns []json.RawMessage `json:"temporal_patterns"`
		Episodes []json.RawMessage `json:"episodes"`
		BootstrapExperiences []json.RawMessage `json:"bootstrap_experiences"`
		Provenance []json.RawMessage `json:"provenance"`
	}
	if err := json.Unmarshal(data, &document); err != nil { t.Fatal(err) }
	if document.SchemaVersion != 2 { t.Fatalf("schema_version = %d, want 2", document.SchemaVersion) }
	for name, values := range map[string][]json.RawMessage{"neural_units": document.NeuralUnits, "populations": document.Populations, "synapses": document.Synapses, "temporal_patterns": document.TemporalPatterns, "episodes": document.Episodes, "bootstrap_experiences": document.BootstrapExperiences, "provenance": document.Provenance} { if len(values) == 0 { t.Fatalf("repository brain_memory.json has empty %s", name) } }
}
