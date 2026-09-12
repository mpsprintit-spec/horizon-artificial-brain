package knowledge

import "testing"

func TestNeuralVectorBoundsAndCopiesInput(t *testing.T) {
	input := []float64{-2, -0.5, 0.25, 2}
	vector := NewNeuralVector(input)
	input[2] = 1

	if vector.Values[0] != -1 || vector.Values[3] != 1 {
		t.Fatalf("expected vector values to be bounded: %#v", vector.Values)
	}
	if vector.Values[2] != 0.25 {
		t.Fatal("expected constructor to copy the input slice")
	}
}

func TestNeuralVectorSimilarityIsModalityNeutral(t *testing.T) {
	a := NewNeuralVector([]float64{-1, 0, 1})
	b := NewNeuralVector([]float64{-1, 0, 1})
	c := NewNeuralVector([]float64{1, 0, -1})

	if a.Similarity(b) != 1 {
		t.Fatal("identical internal patterns should have maximal similarity")
	}
	if a.Similarity(c) >= 0.5 {
		t.Fatal("opposed internal patterns should not be treated as similar")
	}
}

func TestProjectVectorReusesCompatibleNeuralUnit(t *testing.T) {
	brain := NewBrain()
	first := NewNeuralVector([]float64{0.8, 0.1, -0.2})
	second := NewNeuralVector([]float64{0.79, 0.11, -0.19})

	id1, score1, created1, err := brain.ProjectVector(first, 0.95)
	if err != nil { t.Fatal(err) }
	if !created1 || score1 != 1 { t.Fatalf("expected first vector to create a unit, created=%v score=%v", created1, score1) }

	id2, score2, created2, err := brain.ProjectVector(second, 0.95)
	if err != nil { t.Fatal(err) }
	if created2 { t.Fatal("similar experience created a duplicate neural unit") }
	if id1 != id2 { t.Fatalf("expected reuse of neural unit %d, got %d", id1, id2) }
	if score2 < 0.95 { t.Fatalf("expected similarity >= threshold, got %v", score2) }
}

func TestProjectVectorCreatesDistinctUnitForDissimilarExperience(t *testing.T) {
	brain := NewBrain()
	first := NewNeuralVector([]float64{1, 0, 0})
	second := NewNeuralVector([]float64{-1, 0, 0})

	id1, _, _, err := brain.ProjectVector(first, 0.95)
	if err != nil { t.Fatal(err) }
	id2, _, created, err := brain.ProjectVector(second, 0.95)
	if err != nil { t.Fatal(err) }
	if !created { t.Fatal("dissimilar experience unexpectedly reused the first unit") }
	if id1 == id2 { t.Fatal("distinct experiences received the same neural unit") }
}
