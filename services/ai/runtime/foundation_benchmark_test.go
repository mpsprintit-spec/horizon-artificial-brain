package runtime

import (
	"fmt"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"math"
	"path/filepath"
	goruntime "runtime"
	"sort"
	"testing"
	"time"
)

// Run this identical harness on baseline and candidate, sequentially. The
// substrate is synthetic; the sensory population is recruited by the real
// runtime. Percentiles exclude setup and include the complete command boundary.
func BenchmarkFoundationTransitions(b *testing.B) {
	for _, n := range []int{1000, 4000, 40000} {
		for _, journal := range []bool{false, true} {
			b.Run(fmt.Sprintf("nodes=%d/journal=%t", n, journal), func(b *testing.B) {
				r := telemetryBenchmarkRuntime(b, n, 4)
				if journal {
					log, e := OpenEventLog(filepath.Join(b.TempDir(), "events.jsonl"))
					if e != nil {
						b.Fatal(e)
					}
					defer log.Close()
					r.SetEventLog(log)
				}
				sensor := factorizedSensor()
				sensor.Modality = "motion"
				sensor.Source = "foundation-benchmark"
				sensor.Fields = []SensorField{{FieldID: "x", GroupID: "imu", Kind: "acceleration.mps2-v1", Values: []float64{.3}}}
				round := func(i int) (float64, float64) {
					sensor.SampleID = fmt.Sprintf("sample-%d", i)
					sensor.Timestamp = time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC).Add(time.Duration(i) * time.Second)
					if i > 0 {
						sensor.Provenance.ParentSampleIDs = []string{fmt.Sprintf("sample-%d", i-1)}
					}
					start := time.Now()
					if _, e := r.ProcessSensorObservation(sensor); e != nil {
						b.Fatal(e)
					}
					s := float64(time.Since(start)) / 1e6
					r.SetClock(transactionClock{sensor.Timestamp.Add(time.Millisecond)})
					start = time.Now()
					if _, e := r.CognitiveThink(1); e != nil {
						b.Fatal(e)
					}
					return s, float64(time.Since(start)) / 1e6
				}
				for i := 0; i < 3; i++ {
					round(i)
				}
				samples, thought := make([]float64, b.N), make([]float64, b.N)
				var before, after goruntime.MemStats
				goruntime.ReadMemStats(&before)
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					samples[i], thought[i] = round(i + 3)
				}
				b.StopTimer()
				goruntime.ReadMemStats(&after)
				for name, values := range map[string][]float64{"sensor": samples, "think": thought} {
					sort.Float64s(values)
					for _, p := range []float64{.5, .95, .99} {
						b.ReportMetric(values[int(math.Ceil(p*float64(len(values))))-1], fmt.Sprintf("%s_p%d_ms", name, int(p*100)))
					}
				}
				b.ReportMetric(float64(after.NumGC-before.NumGC), "GCs")
				b.ReportMetric(float64(after.PauseTotalNs-before.PauseTotalNs)/1e6, "GC_pause_ms")
				b.ReportMetric(float64(len(r.activation.CurrentState())), "active_nodes")
				b.ReportMetric(float64(n*4), "fixture_synapses")
			})
		}
	}
}

func BenchmarkFoundationPersistence(b *testing.B) {
	for _, n := range []int{1000, 4000, 40000} {
		for _, op := range []string{"checkpoint", "restore", "replay100"} {
			b.Run(fmt.Sprintf("nodes=%d/op=%s", n, op), func(b *testing.B) {
				r := telemetryBenchmarkRuntime(b, n, 4)
				dir := b.TempDir()
				cp := filepath.Join(dir, "checkpoint.json")
				wal, err := OpenEventLog(filepath.Join(dir, "events.jsonl"))
				if err != nil {
					b.Fatal(err)
				}
				defer wal.Close()
				r.SetEventLog(wal)
				if err = r.Checkpoint(cp); err != nil {
					b.Fatal(err)
				}
				if op == "replay100" {
					for i := 0; i < 100; i++ {
						r.SetClock(transactionClock{time.Date(2026, 11, 6, 0, 0, i, 0, time.UTC)})
						if _, err = r.CognitiveThink(1); err != nil {
							b.Fatal(err)
						}
					}
				}
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					switch op {
					case "checkpoint":
						err = r.Checkpoint(cp)
					case "restore":
						err = NewBrainRuntime(knowledge.NewBrain()).RestoreCheckpoint(cp)
					case "replay100":
						_, err = RebuildRuntimeFromCheckpointAndJournal(wal, cp)
					}
					if err != nil {
						b.Fatal(err)
					}
				}
				b.StopTimer()
			})
		}
	}
}
