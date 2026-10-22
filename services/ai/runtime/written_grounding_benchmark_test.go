package runtime

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

// Isolate grounding cost from setup, complete serialization and cognition.
// These synthetic spellings test routing/memory, never language understanding.
func BenchmarkWrittenGroundingContracts(b *testing.B) {
	for _, size := range []int{100, 1000} {
		for _, neural := range []bool{false, true} {
			b.Run(fmt.Sprintf("units=%d/numeric=%t", size, neural), func(b *testing.B) {
				b.StopTimer()
				setup := time.Now()
				r := NewBrainRuntime(knowledge.NewBrain())
				r.SetClock(NewFixedClock(numericWrittenAt))
				for i := 0; i < size; i++ {
					if _, err := r.LearnExperience(learning.Experience{Sequence: []string{fmt.Sprintf("codec-%06d", i)}}, numericWrittenAt); err != nil {
						b.Fatal(err)
					}
				}
				if neural {
					plan, err := r.PlanNeuralWrittenGrounding()
					if err != nil {
						b.Fatal(err)
					}
					if err := r.ApplyNeuralWrittenGrounding(plan, filepath.Join(b.TempDir(), "backup.json"), numericWrittenAt); err != nil {
						b.Fatal(err)
					}
				}
				probe := fmt.Sprintf("codec-%06d", size/2)
				if _, err := r.GroundObservationAt(probe, "diagnostic", "text", numericWrittenAt); err != nil {
					b.Fatal(err)
				}
				setupSeconds := time.Since(setup).Seconds()
				path := filepath.Join(b.TempDir(), "checkpoint.json")
				if err := r.Checkpoint(path); err != nil {
					b.Fatal(err)
				}
				stat, err := os.Stat(path)
				if err != nil {
					b.Fatal(err)
				}
				b.ReportAllocs()
				b.ResetTimer()
				b.StartTimer()
				for i := 0; i < b.N; i++ {
					if _, err := r.GroundObservationAt(probe, "changed-source", "text", numericWrittenAt); err != nil {
						b.Fatal(err)
					}
				}
				b.StopTimer()
				b.ReportMetric(setupSeconds, "setup_s")
				b.ReportMetric(float64(stat.Size()), "checkpoint_bytes")
				b.ReportMetric(float64(len(r.brain.Registry.Nodes())), "final_units")
				b.ReportMetric(float64(len(r.brain.ProjectionPopulations)), "populations")
			})
		}
	}
}

// Optional external rollback evidence: the separately compiled H18 reader must
// restore backup.json, and its canonical bytes must match expected_hash below.
func TestWrittenGroundingRollbackProbe(t *testing.T) {
	dir := os.Getenv("HORIZON_WRITTEN_ROLLBACK_PROBE")
	if dir == "" {
		t.Skip("explicit old-binary rollback probe only")
	}
	r, _, _ := numericWrittenFixture(t, 7, true)
	if _, err := r.CognitiveThink(1); err != nil {
		t.Fatal(err)
	}
	hash, err := r.brain.CanonicalHash()
	if err != nil {
		t.Fatal(err)
	}
	plan, err := r.PlanNeuralWrittenGrounding()
	if err != nil {
		t.Fatal(err)
	}
	if err := r.ApplyNeuralWrittenGrounding(plan, filepath.Join(dir, "backup.json"), numericWrittenAt); err != nil {
		t.Fatal(err)
	}
	if err := r.Checkpoint(filepath.Join(dir, "after-migration.json")); err != nil {
		t.Fatal(err)
	}
	report, _ := json.Marshal(map[string]any{"expected_hash": hash, "parent": "8a4f9ced05d715b338bab4d324011e8c4f6c8596"})
	if err := os.WriteFile(filepath.Join(dir, "reference.json"), report, 0644); err != nil {
		t.Fatal(err)
	}
}
