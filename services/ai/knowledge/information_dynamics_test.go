package knowledge

import (
	"testing"
	"time"
)

func TestInformationDynamicsRisesFromNovelRepresentation(t *testing.T) {
	k := NewKnowledgeBase()
	baseID, _, _, err := k.ProjectVector(NewNeuralVector([]float64{1, 0}), 0.99)
	if err != nil {
		t.Fatal(err)
	}
	novelID, _, _, err := k.ProjectVector(NewNeuralVector([]float64{-1, 0}), 0.99)
	if err != nil {
		t.Fatal(err)
	}

	k.ProjectionPopulations = []ProjectionPopulation{
		{Prototype: []float64{1, 0}},
	}

	baseline := k.UpdateInformationDynamics(
		map[NodeID]float64{baseID: 1},
		map[NodeID]float64{baseID: 1},
		0,
		time.Unix(100, 0),
	)
	novel := k.UpdateInformationDynamics(
		map[NodeID]float64{novelID: 1},
		map[NodeID]float64{novelID: 1},
		0,
		time.Unix(101, 0),
	)

	if novel.Novelty <= baseline.Novelty {
		t.Fatalf("novelty did not increase: baseline=%v novel=%v", baseline.Novelty, novel.Novelty)
	}
	if novel.InformationValue <= baseline.InformationValue {
		t.Fatalf("information value did not increase: baseline=%v novel=%v", baseline.InformationValue, novel.InformationValue)
	}
}

func TestInformationDynamicsPersistentPredictionErrorRaisesUncertainty(t *testing.T) {
	k := NewKnowledgeBase()
	actualID, _, _, err := k.ProjectVector(NewNeuralVector([]float64{1, 0}), 0.99)
	if err != nil {
		t.Fatal(err)
	}
	predictedID, _, _, err := k.ProjectVector(NewNeuralVector([]float64{0, 1}), 0.99)
	if err != nil {
		t.Fatal(err)
	}
	k.ProjectionPopulations = []ProjectionPopulation{
		{Prototype: []float64{1, 0}},
	}

	start := k.UpdateInformationDynamics(
		map[NodeID]float64{actualID: 1},
		map[NodeID]float64{actualID: 1},
		0,
		time.Unix(200, 0),
	)
	current := start
	for i := 0; i < 6; i++ {
		current = k.UpdateInformationDynamics(
			map[NodeID]float64{predictedID: 1},
			map[NodeID]float64{actualID: 1},
			1,
			time.Unix(int64(201+i), 0),
		)
	}

	if current.ErrorEMA <= start.ErrorEMA {
		t.Fatalf("persistent mismatch did not raise error EMA: start=%v current=%v", start.ErrorEMA, current.ErrorEMA)
	}
	if current.Uncertainty <= start.Uncertainty {
		t.Fatalf("persistent mismatch did not raise uncertainty: start=%v current=%v", start.Uncertainty, current.Uncertainty)
	}
}

func TestInformationDynamicsPredictableExperienceRelaxesAfterMismatch(t *testing.T) {
	k := NewKnowledgeBase()
	actualID, _, _, err := k.ProjectVector(NewNeuralVector([]float64{1, 0}), 0.99)
	if err != nil {
		t.Fatal(err)
	}
	predictedID, _, _, err := k.ProjectVector(NewNeuralVector([]float64{0, 1}), 0.99)
	if err != nil {
		t.Fatal(err)
	}
	k.ProjectionPopulations = []ProjectionPopulation{
		{Prototype: []float64{1, 0}},
	}

	for i := 0; i < 5; i++ {
		k.UpdateInformationDynamics(
			map[NodeID]float64{predictedID: 1},
			map[NodeID]float64{actualID: 1},
			1,
			time.Unix(int64(300+i), 0),
		)
	}
	peak := k.InformationDynamics()

	var current InformationDynamics
	for i := 0; i < 12; i++ {
		current = k.UpdateInformationDynamics(
			map[NodeID]float64{actualID: 1},
			map[NodeID]float64{actualID: 1},
			0,
			time.Unix(int64(305+i), 0),
		)
	}

	if current.ErrorEMA >= peak.ErrorEMA {
		t.Fatalf("predictable experience did not reduce error EMA: peak=%v current=%v", peak.ErrorEMA, current.ErrorEMA)
	}
	if current.Uncertainty >= peak.Uncertainty {
		t.Fatalf("predictable experience did not reduce uncertainty: peak=%v current=%v", peak.Uncertainty, current.Uncertainty)
	}
}

func TestInformationDynamicsIsBounded(t *testing.T) {
	k := NewKnowledgeBase()
	id, _, _, err := k.ProjectVector(NewNeuralVector([]float64{1, 0}), 0.99)
	if err != nil {
		t.Fatal(err)
	}

	dynamics := k.UpdateInformationDynamics(
		map[NodeID]float64{id: 1},
		map[NodeID]float64{id: 1},
		100,
		time.Unix(400, 0),
	)
	if dynamics.Uncertainty < 0 || dynamics.Uncertainty > 1 ||
		dynamics.InformationValue < 0 || dynamics.InformationValue > 1 ||
		dynamics.Novelty < 0 || dynamics.Novelty > 1 {
		t.Fatalf("information dynamics escaped bounds: %+v", dynamics)
	}
}
