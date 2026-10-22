package knowledge

import (
	"reflect"
	"testing"
	"time"
)

func TestWrittenSurfaceAliasesReusePopulationAndKeepSensorsSeparate(t *testing.T) {
	brain := NewBrain()
	at := time.Unix(100, 0).UTC()
	var written GroundedRepresentation
	for _, modality := range []string{"text", "language", " TEXT ", "web-search"} {
		g, err := brain.GroundObservationAt(" AiR ", "source-"+modality, modality, .95, at)
		if err != nil {
			t.Fatal(err)
		}
		if written.NodeID == 0 {
			written = g
		}
		if !reflect.DeepEqual(g.Population, written.Population) || g.Token != "air" || g.Modality != modality {
			t.Fatalf("written alias changed population or transport provenance: %+v", g)
		}
		if !reflect.DeepEqual(EncodeObservation(" AIR ", modality), EncodeObservation("air", "language")) {
			t.Fatal("written encoder aliases disagree")
		}
	}
	for _, modality := range []string{"audio", "vision", "imu"} {
		g, err := brain.GroundObservationAt("air", "physical-sensor", modality, .95, at)
		if err != nil {
			t.Fatal(err)
		}
		if g.NodeID == written.NodeID || reflect.DeepEqual(EncodeObservation("air", modality), EncodeObservation("air", "text")) {
			t.Fatalf("%s collapsed into written text", modality)
		}
	}
	for _, modality := range []string{"text", "language"} {
		if label := brain.SurfaceForNode(written.NodeID, modality); label != "air" {
			t.Fatalf("live %s label=%q", modality, label)
		}
		snapshot, err := brain.InspectReadOnly()
		if err != nil {
			t.Fatal(err)
		}
		if label := snapshot.SurfaceForNode(written.NodeID, modality); label != "air" {
			t.Fatalf("detached %s label=%q", modality, label)
		}
	}
}

func TestKnowledgeCompilerUsesWrittenContractAndRetainsSourceMetadata(t *testing.T) {
	doc := KnowledgeDocument{ID: "written", Domain: "test", Source: "teacher", SourceType: "document", Experiences: []KnowledgeExperienceInput{{Before: []KnowledgeDatum{{Value: " NAKU ", Modality: "text"}}, After: []KnowledgeDatum{{Value: "naku", Modality: "language"}}}}}
	compiled, err := CompileKnowledgeDocument(doc, time.Unix(100, 0))
	if err != nil {
		t.Fatal(err)
	}
	exposures := compiled.Experiences[0].SymbolExposures
	if !reflect.DeepEqual(exposures[0].Representation, exposures[1].Representation) || exposures[0].Modality != "text" || exposures[1].Modality != "language" {
		t.Fatal("compiler encoder or source modality is inconsistent")
	}
	legacy, err := CompileKnowledgeDocumentWithEncoding(doc, time.Unix(100, 0), LegacySurfaceEncoding)
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(legacy.Experiences[0].SymbolExposures[0].Representation, legacy.Experiences[0].SymbolExposures[1].Representation) {
		t.Fatal("historical compiler contract was rewritten")
	}
}
