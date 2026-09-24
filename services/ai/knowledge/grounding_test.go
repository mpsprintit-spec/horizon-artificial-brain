package knowledge

import "testing"

func TestGroundObservationCreatesCandidateThenReusesRepresentation(t *testing.T) {
	brain := NewBrain()

	first, err := brain.GroundObservation("unknown-object-42", "test", "vision", 0.95)
	if err != nil {
		t.Fatal(err)
	}
	if first.Status != GroundingCandidate {
		t.Fatalf("first observation status = %q, want %q", first.Status, GroundingCandidate)
	}
	if first.NodeID == 0 {
		t.Fatal("candidate did not receive a neural node ID")
	}

	second, err := brain.GroundObservation("unknown-object-42", "test", "vision", 0.95)
	if err != nil {
		t.Fatal(err)
	}
	if second.Status != GroundingExisting {
		t.Fatalf("second observation status = %q, want %q", second.Status, GroundingExisting)
	}
	if second.NodeID != first.NodeID {
		t.Fatalf("representation was not reused: first=%d second=%d", first.NodeID, second.NodeID)
	}
	if second.Similarity < 0.95 {
		t.Fatalf("similarity = %f, want >= 0.95", second.Similarity)
	}
}

func TestGroundObservationDoesNotCreateTokenIdentity(t *testing.T) {
	brain := NewBrain()
	grounded, err := brain.GroundObservation("new-sensor-value", "sensor", "tactile", 0.99)
	if err != nil {
		t.Fatal(err)
	}
	node := brain.Registry.GetByID(grounded.NodeID)
	if node == nil {
		t.Fatal("grounded node not found")
	}
	if len(grounded.Population) < 2 {
		t.Fatalf("grounded observation is not distributed: population size = %d", len(grounded.Population))
	}
	if len(node.Representation) == 0 {
		t.Fatal("grounded node has no numeric representation")
	}
}
