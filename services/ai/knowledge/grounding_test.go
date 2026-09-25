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



func TestGroundObservationKeepsCrossModalSurfacePopulationsDistinct(t *testing.T) {
	brain := NewBrain()
	language, err := brain.GroundObservation("air", "text", "language", 0.95)
	if err != nil {
		t.Fatal(err)
	}
	vision, err := brain.GroundObservation("air", "camera", "vision", 0.95)
	if err != nil {
		t.Fatal(err)
	}
	if language.Status != GroundingCandidate || vision.Status != GroundingCandidate {
		t.Fatalf("first observations should remain surface candidates: language=%s vision=%s", language.Status, vision.Status)
	}
	if language.NodeID == vision.NodeID {
		t.Fatalf("cross-modal surface observations collapsed into one neural unit: %d", language.NodeID)
	}
	if len(language.Population) != defaultProjectionPopulation || len(vision.Population) != defaultProjectionPopulation {
		t.Fatalf("expected two distributed surface populations of %d: language=%d vision=%d", defaultProjectionPopulation, len(language.Population), len(vision.Population))
	}
	if got := len(brain.ProjectionPopulations); got != 2 {
		t.Fatalf("expected separate surface populations, got %d", got)
	}
}

func TestGroundObservationReusesPopulationWithinSameModality(t *testing.T) {
	brain := NewBrain()
	first, err := brain.GroundObservation("air", "camera-1", "vision", 0.95)
	if err != nil {
		t.Fatal(err)
	}
	second, err := brain.GroundObservation("air", "camera-2", "vision", 0.95)
	if err != nil {
		t.Fatal(err)
	}
	if second.Status != GroundingExisting {
		t.Fatalf("same-modality repeat should reuse its population, got %s", second.Status)
	}
	if second.NodeID != first.NodeID {
		t.Fatalf("same-modality repeat changed anchor: first=%d second=%d", first.NodeID, second.NodeID)
	}
	if len(second.Population) != len(first.Population) {
		t.Fatalf("same-modality repeat changed population size: first=%d second=%d", len(first.Population), len(second.Population))
	}
	if got := len(brain.Registry.Nodes()); got != defaultProjectionPopulation {
		t.Fatalf("same-modality repeat grew topology: got %d nodes", got)
	}
	if got := len(brain.ProjectionPopulations); got != 1 {
		t.Fatalf("same-modality repeat created another population: got %d", got)
	}
}


func TestBindGroundedRepresentationsKeepsModalitiesDistinctAndLearnsSharedPattern(t *testing.T) {
	brain := NewBrain()
	written, err := brain.GroundObservation("api", "text", "language", 0.95)
	if err != nil {
		t.Fatal(err)
	}
	visual, err := brain.GroundObservation("api", "camera", "vision", 0.95)
	if err != nil {
		t.Fatal(err)
	}
	pattern := brain.BindGroundedRepresentations(written, visual, time.Unix(10, 0).UTC())
	if pattern == nil {
		t.Fatal("expected cross-modal grounding pattern")
	}
	if len(pattern.Members) != 2 || pattern.Members[0] != written.NodeID || pattern.Members[1] != visual.NodeID {
		t.Fatalf("unexpected shared pattern members: %v", pattern.Members)
	}
	if written.NodeID == visual.NodeID {
		t.Fatal("shared grounding must not collapse surface populations")
	}

	repeated := brain.BindGroundedRepresentations(written, visual, time.Unix(11, 0).UTC())
	if repeated == nil || repeated.ID != pattern.ID {
		t.Fatal("repeated cross-modal experience should consolidate the existing pattern")
	}
	if repeated.Frequency != 2 {
		t.Fatalf("expected shared grounding frequency 2, got %d", repeated.Frequency)
	}
	if len(brain.Registry.Nodes()) != defaultProjectionPopulation*2 {
		t.Fatalf("cross-modal binding changed neural population topology: got %d nodes", len(brain.Registry.Nodes()))
	}
}
