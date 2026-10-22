package runtime

import (
	"bytes"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestInspectionWireFixtureMatchesProductionEncoder(t *testing.T) {
	at := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	brain := knowledge.NewBrain()
	for _, values := range [][]float64{{1, math.Copysign(0, -1), 1e-9}, {0, 1, 1.2345678901234567}} {
		if _, _, _, err := brain.Registry.GetOrCreateRepresentationAt(knowledge.NewNeuralVector(values), 1, at); err != nil {
			t.Fatal(err)
		}
	}
	brain.BrainState.MemoryState["escaped<>&\u2028"] = 1e-9
	r := NewBrainRuntime(brain)
	r.SetClock(NewFixedClock(at))
	r.inspectionEpoch = "wire-fixture-v1"
	page, err := r.MonitorInspectionPage(InspectionPageRequest{Section: "neural_units", Order: "activity", Limit: 1}, "wire-fixture")
	if err != nil {
		t.Fatal(err)
	}
	var wire bytes.Buffer
	if err := json.NewEncoder(&wire).Encode(page); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join("..", "..", "..", "web", "monitor", "testdata", "inspection_page_v1.json")
	if os.Getenv("HORIZON_WRITE_INSPECTION_FIXTURE") == "1" {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, wire.Bytes(), 0644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(wire.Bytes(), want) {
		t.Fatal("browser fixture differs from actual Go wire encoder")
	}
}
