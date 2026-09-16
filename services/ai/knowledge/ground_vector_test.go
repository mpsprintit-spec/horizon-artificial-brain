package knowledge

import "testing"

func TestGroundVectorReusesExistingRepresentation(t *testing.T) {
	brain := NewBrain()
	vector := NewNeuralVector([]float64{0.2, -0.1, 0.7})

	first, err := brain.GroundVector(vector)
	if err != nil {
		t.Fatalf("first grounding failed: %v", err)
	}
	second, err := brain.GroundVector(vector)
	if err != nil {
		t.Fatalf("second grounding failed: %v", err)
	}
	if first.NodeID == "" || second.NodeID == "" {
		t.Fatal("expected node ids")
	}
	if first.NodeID != second.NodeID {
		t.Fatalf("expected representation reuse: %q != %q", first.NodeID, second.NodeID)
	}
	if !second.Existing {
		t.Fatal("expected second grounding to identify an existing representation")
	}
}
