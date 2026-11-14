package runtime

import (
	"fmt"
	"github.com/project-horizon/horizon-core/services/ai/activation"
	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
	"math"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
	"time"
)

func BenchmarkNativeSubstrate(b *testing.B) {
	for _, n := range []int{1000, 4000, 40000} {
		for _, density := range []float64{.001, .1, 1} {
			for _, mode := range []string{"eager", "lazy", "operation"} {
				b.Run(fmt.Sprintf("nodes=%d/density=%g/mode=%s", n, density, mode), func(b *testing.B) {
					r := telemetryBenchmarkRuntime(b, n, 4)
					state := map[k.NodeID]float64{}
					active := int(math.Max(1, float64(n)*density))
					for i := 1; i <= active; i++ {
						state[k.NodeID(i)] = .8
					}
					samples := make([]float64, b.N)
					at := time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC)
					var before, after runtime.MemStats
					runtime.ReadMemStats(&before)
					b.ReportAllocs()
					b.ResetTimer()
					mutated := 0
					reads := 0
					for i := 0; i < b.N; i++ {
						now := at.Add(time.Duration(i) * time.Millisecond)
						if mode == "operation" {
							r.activation.RestoreState(activation.StateSnapshot{InternalState: state, InternalConfidence: state})
						}
						start := time.Now()
						switch mode {
						case "eager":
							r.activation.ApplyActivityPlasticityWithDynamics(state, state, now, k.CurrentDynamics)
							reads = n * 4
							mutated = n * 4
						case "lazy":
							mutated = r.activation.ApplySparseObservedPlasticity(state, state, now)
							reads = active * 4
						case "operation":
							w, e := r.activation.ExecuteNativeOperation(k.NativeOperation{Kind: "propagate", Gain: 1}, now)
							if e != nil {
								b.Fatal(e)
							}
							reads = w.ReadSynapses
							mutated = w.MutatedNodes
						}
						samples[i] = float64(time.Since(start)) / 1e6
					}
					b.StopTimer()
					runtime.ReadMemStats(&after)
					sort.Float64s(samples)
					for _, p := range []float64{.5, .95, .99} {
						b.ReportMetric(samples[int(math.Ceil(p*float64(len(samples))))-1], fmt.Sprintf("p%d_ms", int(p*100)))
					}
					b.ReportMetric(float64(active), "active_nodes")
					b.ReportMetric(float64(reads), "synapses_read")
					b.ReportMetric(float64(mutated), "mutated")
					b.ReportMetric(float64(after.NumGC-before.NumGC), "GCs")
					b.ReportMetric(float64(after.PauseTotalNs-before.PauseTotalNs)/1e6, "GC_pause_ms")
				})
			}
		}
	}
}

// Complete command-boundary cost including provenance, policy readout,
// recurrent transitions, delayed learning, dedup and optional fsync WAL.
// The sampled trajectory is controlled for timing, not a learning-quality arm.
func BenchmarkNativeController(b *testing.B) {
	for _, n := range []int{1000, 4000, 40000} {
		for _, journal := range []bool{false, true} {
			b.Run(fmt.Sprintf("nodes=%d/journal=%t", n, journal), func(b *testing.B) {
				r := telemetryBenchmarkRuntime(b, n, 4)
				if journal {
					wal, err := OpenEventLog(filepath.Join(b.TempDir(), "events.jsonl"))
					if err != nil {
						b.Fatal(err)
					}
					defer wal.Close()
					r.SetEventLog(wal)
				}
				ids := make([]k.NodeID, 64)
				state := map[k.NodeID]float64{}
				for i := range ids {
					ids[i] = k.NodeID(i + 1)
					state[ids[i]] = .8
				}
				cfg := nativeControlConfig(ids)
				at := time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC)
				apply := func(c NativeControlCommand) NativeControlResult {
					out, err := r.ApplyNativeControl(c, at.Add(time.Duration(r.LastSequence()+1)*time.Microsecond))
					if err != nil {
						b.Fatal(err)
					}
					return out
				}
				apply(NativeControlCommand{ID: "initialize", Initialize: &cfg})
				round := func(i int) int {
					id := fmt.Sprintf("episode-%d", i)
					apply(NativeControlCommand{ID: id, Begin: &NativeBegin{EpisodeID: id, Training: true, Observation: &NativeObservation{Source: "benchmark", SampleID: id, State: state}}})
					reads := 0
					var out NativeControlResult
					for step := 0; step < 3; step++ {
						u := 0.0
						if step == 2 {
							u = math.Nextafter(1, 0)
						}
						out = apply(NativeControlCommand{ID: fmt.Sprintf("%s-%d", id, step), Step: &NativeStep{EpisodeID: id, Sample: u}})
						reads += out.Work.ReadSynapses
						if out.Stopped {
							break
						}
					}
					if !out.Stopped {
						b.Fatal("timing trajectory did not stop")
					}
					apply(NativeControlCommand{ID: id + "-out", Outcome: &NativeOutcome{EpisodeID: id, Source: "benchmark", OutcomeID: id, TraceDigest: out.TraceDigest, Provenance: "observed-environment", Reward: .5, Learn: true}})
					return reads
				}
				for i := 0; i < 3; i++ {
					round(i)
				}
				samples := make([]float64, b.N)
				var before, after runtime.MemStats
				runtime.ReadMemStats(&before)
				b.ReportAllocs()
				b.ResetTimer()
				reads := 0
				for i := 0; i < b.N; i++ {
					start := time.Now()
					reads = round(i + 3)
					samples[i] = float64(time.Since(start)) / 1e6
				}
				b.StopTimer()
				runtime.ReadMemStats(&after)
				sort.Float64s(samples)
				for _, p := range []float64{.5, .95, .99} {
					b.ReportMetric(samples[int(math.Ceil(p*float64(len(samples))))-1], fmt.Sprintf("episode_p%d_ms", int(p*100)))
				}
				b.ReportMetric(float64(reads), "transition_reads_per_episode")
				b.ReportMetric(64, "input_active_nodes")
				b.ReportMetric(float64(after.NumGC-before.NumGC), "GCs")
				b.ReportMetric(float64(after.PauseTotalNs-before.PauseTotalNs)/1e6, "GC_pause_ms")
			})
		}
	}
}
