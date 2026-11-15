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

// One op is two actually executed, matched trials (propagate->stop and stop),
// including command freezing, provenance, learning and optional durable WAL.
// Rewards here are workload inputs, not evidence for a cognitive capability.
func BenchmarkNativeActionPair(b *testing.B) {
	for _, n := range []int{1000, 4000, 40000} {
		for _, density := range []float64{.001, .1, 1} {
			for _, journal := range []bool{false, true} {
				b.Run(fmt.Sprintf("nodes=%d/density=%g/journal=%t", n, density, journal), func(b *testing.B) {
					rt := telemetryBenchmarkRuntime(b, n, 4)
					if journal {
						wal, err := OpenEventLog(filepath.Join(b.TempDir(), "events.jsonl"))
						if err != nil {
							b.Fatal(err)
						}
						defer wal.Close()
						rt.SetEventLog(wal)
					}
					cfg := nativeControlConfig([]k.NodeID{1})
					cfg.Version = k.NativeControlCausalVersion
					cfg.Encoder = "local-relational-v1"
					cfg.MaxActiveNodes = 100000
					cfg.MaxSynapseReads = 1000000
					at := time.Date(2026, 11, 8, 0, 0, 0, 0, time.UTC)
					apply := func(c NativeControlCommand) NativeControlResult {
						out, err := rt.ApplyNativeControl(c, at.Add(time.Duration(rt.LastSequence()+1)*time.Microsecond))
						if err != nil {
							b.Fatal(err)
						}
						return out
					}
					apply(NativeControlCommand{ID: "init", Initialize: &cfg})
					active := max(1, int(float64(n)*density))
					state := map[k.NodeID]float64{}
					for i := 1; i <= active; i++ {
						state[k.NodeID(i)] = .8
					}
					round := func(i int) (int, int) {
						pair := fmt.Sprintf("pair%d", i)
						reads, updates := 0, 0
						for trial := 0; trial < 2; trial++ {
							id := fmt.Sprintf("%s-%d", pair, trial)
							apply(NativeControlCommand{ID: id, Begin: &NativeBegin{EpisodeID: id, Training: true, Intervention: true, MatchKey: pair, Observation: &NativeObservation{Source: "pair-benchmark", SampleID: id, State: state}}})
							actions := []int{0, 3}
							reward := .5
							if trial == 1 {
								actions = []int{3}
								reward = .2
							}
							var out NativeControlResult
							for j, a := range actions {
								action := a
								out = apply(NativeControlCommand{ID: fmt.Sprintf("%s-%d", id, j), Step: &NativeStep{EpisodeID: id, ExperimentAction: &action}})
								reads += out.Work.ReadSynapses
							}
							learned := apply(NativeControlCommand{ID: id + "-out", Outcome: &NativeOutcome{EpisodeID: id, TraceDigest: out.TraceDigest, Source: "pair-benchmark", OutcomeID: id, Provenance: "observed-environment", Reward: reward, Learn: true, ContrastID: pair, ContrastMode: "action"}})
							updates += learned.Learning.UpdatedSynapses
						}
						return reads, updates
					}
					for i := 0; i < 3; i++ {
						round(i)
					}
					samples := make([]float64, b.N)
					var before, after runtime.MemStats
					runtime.ReadMemStats(&before)
					b.ReportAllocs()
					b.ResetTimer()
					reads, updates := 0, 0
					for i := 0; i < b.N; i++ {
						start := time.Now()
						reads, updates = round(i + 3)
						samples[i] = float64(time.Since(start)) / 1e6
					}
					b.StopTimer()
					runtime.ReadMemStats(&after)
					sort.Float64s(samples)
					for _, p := range []float64{.5, .95, .99} {
						b.ReportMetric(samples[int(math.Ceil(p*float64(b.N)))-1], fmt.Sprintf("pair_p%d_ms", int(p*100)))
					}
					b.ReportMetric(float64(active), "input_active_nodes")
					b.ReportMetric(float64(reads), "transition_reads_per_pair")
					b.ReportMetric(float64(updates), "last_pair_synapse_updates")
					b.ReportMetric(float64(after.NumGC-before.NumGC), "GCs")
					b.ReportMetric(float64(after.PauseTotalNs-before.PauseTotalNs)/1e6, "GC_pause_ms")
				})
			}
		}
	}
}
