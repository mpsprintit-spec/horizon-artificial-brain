package knowledge

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRetiredCognitiveBrainRefusedWithoutMutation(t *testing.T) {
	cases := map[string]string{
		"top":               `{"schema_version":2,"cognitive_model":{"optimizer_step":7}}`,
		"escaped-key":       `{"schema_version":2,"\u0063ognitive_model":{"weights":[1,2]}}`,
		"nested":            `{"brain_state":{"cognitive_model":{"weights":[1,2]}}}`,
		"population":        `{"schema_version":2,"populations":[{"projection_scope":{"modality":"internal-numeric","field":"internal-shared-gru-muse-bmps-v1"}}]}`,
		"legacy-population": `{"projection_populations":[{"projection_scope":{"modality":"internal-numeric","field":"internal-shared-gru-muse-bmps-v1"}}]}`,
	}
	for version := 4; version <= 6; version++ {
		cases[fmt.Sprintf("schema%d", version)] = fmt.Sprintf(`{"schema_version":%d}`, version)
	}
	for name, input := range cases {
		for _, standalone := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/standalone=%v", name, standalone), func(t *testing.T) {
				brain := NewBrain()
				brain.Store("retained-neuron")
				before, err := brain.CanonicalJSON()
				if err != nil {
					t.Fatal(err)
				}
				path := filepath.Join(t.TempDir(), "historical.json")
				original := []byte(input)
				if err := os.WriteFile(path, original, 0600); err != nil {
					t.Fatal(err)
				}
				if standalone {
					err = brain.LoadStandalone(path)
				} else {
					err = brain.Load(path)
				}
				if err == nil || !strings.Contains(err.Error(), "retired cognitive") {
					t.Fatalf("expected explicit retired-state refusal, got %v", err)
				}
				after, err := brain.CanonicalJSON()
				if err != nil {
					t.Fatal(err)
				}
				file, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(before, after) || !bytes.Equal(original, file) {
					t.Fatal("refused restore changed the live Brain or historical file")
				}
			})
		}
	}
}

func TestRetiredCognitiveNullDoesNotRejectSupportedBrain(t *testing.T) {
	if err := ValidateMemoryDecayState([]byte(`{"schema_version":2,"cognitive_model":null,"brain_state":{"cognitive_model":null}}`), true); err != nil {
		t.Fatal(err)
	}
}
