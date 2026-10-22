package knowledge_test

import (
	"bytes"
	"path/filepath"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestCanonicalLoadDoesNotRelabelOrdinaryPopulationsAsBootstrap(t *testing.T) {
	brain := knowledge.NewBrain()
	at := time.Date(2026, 10, 19, 9, 0, 0, 0, time.UTC)
	if _, err := brain.GroundVectorAt(knowledge.NewNeuralVector([]float64{.2, .8}), at); err != nil {
		t.Fatal(err)
	}
	if _, err := brain.ProjectSensoryFieldAt(knowledge.NewNeuralVector([]float64{.7, -.2, .1}), knowledge.ProjectionScope{Modality: "vision", Field: "light.rgb-v1"}, .9, 4, at); err != nil {
		t.Fatal(err)
	}
	before, err := brain.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "canonical.json")
	if err := brain.Save(path); err != nil {
		t.Fatal(err)
	}
	restored := knowledge.NewBrain()
	if err := restored.Load(path); err != nil {
		t.Fatal(err)
	}
	after, err := restored.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("canonical load synthesized bootstrap metadata for observed populations")
	}
	if len(restored.BrainState.BootstrapExperiences) != 0 {
		t.Fatal("canonical load invented bootstrap experience")
	}
}
