package runtime

import (
	"fmt"
	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
	"math"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
	"time"
)

func BenchmarkNativeRelationalPolicy(b *testing.B) {
	for _, n := range []int{1000, 4000, 40000} {
		for _, density := range []float64{.001, .1, 1} {
			b.Run(fmt.Sprintf("nodes=%d/density=%g", n, density), func(b *testing.B) {
				rt := telemetryBenchmarkRuntime(b, n, 4)
				cfg := nativeControlConfig([]k.NodeID{1})
				cfg.Version = k.NativeControlRelationalVersion
				cfg.Encoder = "local-relational-v1"
				cfg.Epsilon = .1
				cfg.MaxActiveNodes = 100000
				cfg.MaxSynapseReads = 1000000
				if err := rt.brain.InitializeNativeControl(cfg, time.Date(2026, 11, 7, 0, 0, 0, 0, time.UTC)); err != nil {
					b.Fatal(err)
				}
				active := int(math.Max(1, float64(n)*density))
				state := map[k.NodeID]float64{}
				for i := 1; i <= active; i++ {
					state[k.NodeID(i)] = .8
				}
				samples := make([]float64, b.N)
				var before, after runtime.MemStats
				var out k.NativeReadout
				runtime.ReadMemStats(&before)
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					start := time.Now()
					var err error
					out, err = rt.brain.NativePolicy(state, true, 0)
					if err != nil {
						b.Fatal(err)
					}
					samples[i] = float64(time.Since(start)) / 1e6
				}
				b.StopTimer()
				runtime.ReadMemStats(&after)
				sort.Float64s(samples)
				for _, p := range []float64{.5, .95, .99} {
					b.ReportMetric(samples[int(math.Ceil(p*float64(b.N)))-1], fmt.Sprintf("p%d_ms", int(p*100)))
				}
				b.ReportMetric(float64(active), "active_nodes")
				b.ReportMetric(float64(out.ParameterReads), "parameter_reads")
				b.ReportMetric(float64(out.StructuralReads), "structural_reads")
				b.ReportMetric(float64(after.NumGC-before.NumGC), "GCs")
				b.ReportMetric(float64(after.PauseTotalNs-before.PauseTotalNs)/1e6, "GC_pause_ms")
			})
		}
	}
}

func BenchmarkNativeRelationalPersistence(b *testing.B) {
	for _, n := range []int{1000, 4000, 40000} {
		for _, mode := range []string{"checkpoint", "restore-pending", "replay-pair"} {
			b.Run(fmt.Sprintf("nodes=%d/mode=%s", n, mode), func(b *testing.B) {
				rt := telemetryBenchmarkRuntime(b, n, 4)
				dir := b.TempDir()
				wal, err := OpenEventLog(filepath.Join(dir, "events.jsonl"))
				if err != nil {
					b.Fatal(err)
				}
				defer wal.Close()
				rt.SetEventLog(wal)
				ids := []k.NodeID{}
				for i := 1; i <= 64; i++ {
					ids = append(ids, k.NodeID(i))
				}
				cfg := nativeControlConfig(ids)
				cfg.Version = k.NativeControlRelationalVersion
				cfg.Encoder = "local-relational-v1"
				cfg.ContrastMean = true
				at := time.Date(2026, 11, 7, 0, 0, 0, 0, time.UTC)
				apply := func(c NativeControlCommand) NativeControlResult {
					out, err := rt.ApplyNativeControl(c, at.Add(time.Duration(rt.LastSequence()+1)*time.Millisecond))
					if err != nil {
						b.Fatal(err)
					}
					return out
				}
				apply(NativeControlCommand{ID: "init", Initialize: &cfg})
				trial := func(id string, v float64) {
					state := map[k.NodeID]float64{}
					for _, node := range ids {
						state[node] = v
					}
					apply(NativeControlCommand{ID: id, Begin: &NativeBegin{EpisodeID: id, Training: true, Intervention: true, Observation: &NativeObservation{Source: "synthetic-benchmark", SampleID: id, State: state}}})
					var out NativeControlResult
					for j, a := range []int{0, 3} {
						action := a
						out = apply(NativeControlCommand{ID: fmt.Sprintf("%s-%d", id, j), Step: &NativeStep{EpisodeID: id, ExperimentAction: &action}})
					}
					apply(NativeControlCommand{ID: id + "-out", Outcome: &NativeOutcome{EpisodeID: id, TraceDigest: out.TraceDigest, Source: "synthetic-benchmark", OutcomeID: id, Provenance: "observed-environment", Reward: v, Learn: true, ContrastID: "pair"}})
				}
				trial("first", .8)
				cp := filepath.Join(dir, "pending.json")
				if err = rt.Checkpoint(cp); err != nil {
					b.Fatal(err)
				}
				trial("second", .4)
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					switch mode {
					case "checkpoint":
						err = rt.Checkpoint(filepath.Join(dir, "after.json"))
					case "restore-pending":
						restored := NewBrainRuntime(k.NewBrain())
						err = restored.RestoreCheckpoint(cp)
					case "replay-pair":
						var rebuilt *BrainRuntime
						rebuilt, err = RebuildRuntimeFromCheckpointAndJournal(wal, cp)
						if err == nil && rebuilt.brain.NativeControlSnapshot().Updates != 1 {
							b.Fatal("credit not recovered")
						}
					}
					if err != nil {
						b.Fatal(err)
					}
				}
				b.StopTimer()
				b.ReportMetric(float64(n), "world_nodes")
			})
		}
	}
}
