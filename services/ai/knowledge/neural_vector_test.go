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
