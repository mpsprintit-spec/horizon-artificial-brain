package knowledge

import (
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestSensoryProjectionReusesCompatibleCoordinatesAndSeparatesOthers(t *testing.T) {
	brain := NewBrain()
	at := time.Date(2026, 10, 18, 14, 0, 0, 0, time.UTC)
	vector := NewNeuralVector([]float64{0.2, 0.8})
	scope := ProjectionScope{Modality: "vision", Field: "light.rgb-v1"}
	first, err := brain.ProjectSensoryFieldAt(vector, scope, .9, 4, at)
	if err != nil {
		t.Fatal(err)
	}
	again, err := brain.ProjectSensoryFieldAt(vector, scope, .9, 4, at.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if !PopulationEquivalent(first, again) || len(brain.Registry.Nodes()) != 4 {
		t.Fatal("same factor did not reuse its population")
	}
	first.Scope.Field = "caller-mutation"
	if brain.ProjectionPopulations[0].Scope.Field != scope.Field {
		t.Fatal("returned scope aliases canonical population metadata")
	}
	other, err := brain.ProjectSensoryFieldAt(vector, ProjectionScope{Modality: "audio", Field: "audio.rms-v1"}, .9, 4, at)
	if err != nil {
		t.Fatal(err)
	}
	legacy, err := brain.ProjectVectorPopulationAt(vector, .9, 4, at)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range again.Units {
		for _, p := range []ProjectionPopulation{other, legacy} {
			for _, b := range p.Units {
				if a.NodeID == b.NodeID {
					t.Fatal("incompatible measurement coordinates shared anonymous units")
				}
			}
		}
	}
	path := filepath.Join(t.TempDir(), "brain.json")
	if err := brain.Save(path); err != nil {
		t.Fatal(err)
	}
	restored := NewBrain()
	if err := restored.Load(path); err != nil {
		t.Fatal(err)
	}
	continued, err := restored.ProjectSensoryFieldAt(vector, scope, .9, 4, at.Add(2*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(continued.Units, again.Units) || len(restored.Registry.Nodes()) != 12 {
		t.Fatal("save/load lost receptive-field identity")
	}
}
