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


func TestGroundObservationBindsCrossModalExperienceToCanonicalConcept(t *testing.T) {
	kb := NewKnowledgeBase()
	canonical := kb.Store("air")
	if canonical == nil {
		t.Fatal("expected canonical air representation")
	}

	grounded, err := kb.GroundObservation("air", "camera-1", "vision", 0.25)
	if err != nil {
		t.Fatalf("GroundObservation failed: %v", err)
	}
	if grounded.Status != GroundingExisting {
		t.Fatalf("expected existing canonical grounding, got %s", grounded.Status)
	}
	if grounded.NodeID != canonical.ID {
		t.Fatalf("cross-modal observation fragmented canonical concept: canonical=%d grounded=%d", canonical.ID, grounded.NodeID)
	}
	if len(grounded.Population) != 1 || grounded.Population[0] != canonical.ID {
		t.Fatalf("expected canonical unit as the only binding, got %v", grounded.Population)
	}
	if got := len(kb.Registry.Nodes()); got != 1 {
		t.Fatalf("cross-modal grounding created extra substrate nodes: got %d", got)
	}
	if got := len(kb.ProjectionPopulations); got != 0 {
		t.Fatalf("cross-modal grounding created a distributed population for a canonical concept: got %d", got)
	}
}

func TestUnknownCrossModalExperienceReusesItsDistributedPopulation(t *testing.T) {
	kb := NewKnowledgeBase()

	first, err := kb.GroundObservation("unknown-object", "camera-1", "vision", 0.25)
	if err != nil {
		t.Fatalf("first grounding failed: %v", err)
	}
	if first.Status != GroundingCandidate {
		t.Fatalf("expected first observation to be a candidate, got %s", first.Status)
	}
	if len(first.Population) != defaultProjectionPopulation {
		t.Fatalf("expected distributed population of %d, got %d", defaultProjectionPopulation, len(first.Population))
	}

	second, err := kb.GroundObservation("unknown-object", "microphone-1", "audio", 0.25)
	if err != nil {
		t.Fatalf("second grounding failed: %v", err)
	}
	if second.Status != GroundingExisting {
		t.Fatalf("expected repeated numeric experience to reuse its population, got %s", second.Status)
	}
	if second.NodeID != first.NodeID {
		t.Fatalf("repeated experience changed anchor: first=%d second=%d", first.NodeID, second.NodeID)
	}
	if len(second.Population) != len(first.Population) {
		t.Fatalf("repeated experience changed population size: first=%d second=%d", len(first.Population), len(second.Population))
	}
	if got := len(kb.Registry.Nodes()); got != defaultProjectionPopulation {
		t.Fatalf("repeated experience grew topology: got %d nodes", got)
	}
	if got := len(kb.ProjectionPopulations); got != 1 {
		t.Fatalf("expected one reusable distributed population, got %d", got)
	}
}
