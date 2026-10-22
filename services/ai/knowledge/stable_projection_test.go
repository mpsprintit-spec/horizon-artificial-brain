package knowledge

import (
	"bytes"
	"math"
	"reflect"
	"slices"
	"testing"
	"time"
)

func stableProject(t *testing.T, b *Brain, v []float64, tolerance []float64) ProjectionPopulation {
	t.Helper()
	p, err := b.ProjectVectorPopulationWithDynamicsAt(NewNeuralVector(v), .9, 4, time.Unix(100, 0), StableProjectionDynamics, tolerance)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestStableRepresentationProtectsPrototypeWithoutFreezingLearning(t *testing.T) {
	for _, dynamics := range []DynamicsEncoding{EvidenceUncertaintyDynamics, StableProjectionDynamics} {
		b := NewBrain()
		v := []float64{.9, .1, 0, 0}
		p := stableProject(t, b, v, nil)
		// Redundant membership must not multiply one observation's update.
		b.registerProjectionPopulation(p)
		active, inactive := p.Units[0].NodeID, p.Units[1].NodeID
		beforeActive := slices.Clone(b.Registry.GetByID(active).Representation)
		beforeInactive := slices.Clone(b.Registry.GetByID(inactive).Representation)
		frequency := b.Registry.GetByID(active).Frequency
		b.ApplyPredictionErrorRepresentationPlasticityWithDynamics(map[NodeID]float64{inactive: 1}, map[NodeID]float64{active: 1}, 1, .4, time.Unix(101, 0), dynamics)
		if dynamics == StableProjectionDynamics {
			if !slices.Equal(b.ProjectionPopulations[0].Prototype, v) || !slices.Equal(b.Registry.GetByID(inactive).Representation, beforeInactive) {
				t.Fatal("prediction error changed sensory prototype or an unobserved member")
			}
			if slices.Equal(b.Registry.GetByID(active).Representation, beforeActive) || b.Registry.GetByID(active).Frequency != frequency+1 {
				t.Fatal("learning stopped or duplicate memberships multiplied credit")
			}
			count := len(b.Registry.Nodes())
			again := stableProject(t, b, v, nil)
			if !PopulationEquivalent(p, again) || len(b.Registry.Nodes()) != count {
				t.Fatal("learned member drift prevented sensory recovery")
			}
		} else if slices.Equal(b.ProjectionPopulations[0].Prototype, v) || slices.Equal(b.Registry.GetByID(inactive).Representation, beforeInactive) {
			t.Fatal("historical mechanism ablation did not reproduce drift/interference")
		}
	}
}

func TestStableProjectionToleranceKeepsDistinctAndAmbiguousExperiencesSeparate(t *testing.T) {
	b := NewBrain()
	a := stableProject(t, b, []float64{.2, .4}, nil)
	count := len(b.Registry.Nodes())
	if p := stableProject(t, b, []float64{.21, .4}, []float64{.02, 0}); !PopulationEquivalent(a, p) || len(b.Registry.Nodes()) != count {
		t.Fatal("one compatible measurement basin did not recover")
	}
	other := stableProject(t, b, []float64{.215, .4}, nil)
	if PopulationEquivalent(a, other) {
		t.Fatal("undeclared proximity merged distinct measurements")
	}
	ambiguous := stableProject(t, b, []float64{.2075, .4}, []float64{.02, 0})
	if PopulationEquivalent(a, ambiguous) || PopulationEquivalent(other, ambiguous) {
		t.Fatal("overlapping noise regions arbitrarily merged populations")
	}
	if p := stableProject(t, b, []float64{.2, .4}, []float64{.02, 0}); !PopulationEquivalent(a, p) {
		t.Fatal("noise tolerance displaced exact prior evidence")
	}
	if p := stableProject(t, b, []float64{.2, .40001}, []float64{.02, 0}); PopulationEquivalent(a, p) {
		t.Fatal("an exact measurement coordinate was blurred by another coordinate's tolerance")
	}
}

func TestStableProjectionInvalidToleranceCannotMutateBrain(t *testing.T) {
	b := NewBrain()
	stableProject(t, b, []float64{.2, .4}, nil)
	before, _ := b.CanonicalJSON()
	for _, tolerance := range [][]float64{{.1}, {-.1, 0}, {math.NaN(), 0}, {math.Inf(1), 0}, {2, 0}} {
		if _, err := b.ProjectVectorPopulationWithDynamicsAt(NewNeuralVector([]float64{.21, .4}), .9, 4, time.Unix(101, 0), StableProjectionDynamics, tolerance); err == nil {
			t.Fatal("invalid tolerance accepted")
		}
		after, _ := b.CanonicalJSON()
		if !bytes.Equal(before, after) {
			t.Fatal("invalid projection changed the Brain")
		}
	}
}

func TestStableProjectionPopulationMetadataIsDetached(t *testing.T) {
	b := NewBrain()
	p := stableProject(t, b, []float64{.2, .4}, nil)
	want := p.Prototype[0]
	p.Prototype[0] = .9
	p.Units[0].NodeID = 9999
	if b.ProjectionPopulations[0].Prototype[0] != want || reflect.DeepEqual(p, b.ProjectionPopulations[0]) {
		t.Fatal("projection response aliases canonical population")
	}
}
