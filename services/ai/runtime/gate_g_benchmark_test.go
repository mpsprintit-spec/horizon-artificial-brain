package runtime

import (
	"fmt"
	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
	"math"
	"path/filepath"
	gr "runtime"
	"sort"
	"syscall"
	"testing"
	"time"
)

// This harness is copied unchanged to the authentic baseline. Seeding is a
// REAL journaled Process transaction outside the thought timing interval.
func BenchmarkGateGThought(b *testing.B)                     { benchmarkGateGThought(b, false, false) }
func BenchmarkGateGThoughtNativeIdle(b *testing.B)           { benchmarkGateGThought(b, true, false) }
func BenchmarkGateGContextContinuity(b *testing.B)           { benchmarkGateGThought(b, false, true) }
func BenchmarkGateGContextContinuityNativeIdle(b *testing.B) { benchmarkGateGThought(b, true, true) }
func benchmarkGateGThought(b *testing.B, nativeIdle, contextOnly bool) {
	for _, n := range []int{1000, 4000, 40000} {
		for _, dense := range []bool{false, true} {
			for _, journal := range []bool{false, true} {
				b.Run(fmt.Sprintf("nodes=%d/dense=%t/journal=%t", n, dense, journal), func(b *testing.B) {
					rt := telemetryBenchmarkRuntime(b, n, 4)
					if journal {
						w, e := OpenEventLog(filepath.Join(b.TempDir(), "events.jsonl"))
						if e != nil {
							b.Fatal(e)
						}
						defer w.Close()
						rt.SetEventLog(w)
					}
					if nativeIdle {
						cfg := nativeControlConfig([]k.NodeID{1})
						cfg.Version = k.NativeControlCausalVersion
						cfg.Encoder = "local-relational-v1"
						if _, e := rt.ApplyNativeControl(NativeControlCommand{ID: "initialize", Initialize: &cfg}, time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC)); e != nil {
							b.Fatal(e)
						}
					}
					count := 9
					if dense {
						count = n
					}
					context := map[k.NodeID]float64{}
					inputs := make([]k.NodeID, 0, count)
					for i := 1; i <= count; i++ {
						context[k.NodeID(i)] = .8
						inputs = append(inputs, k.NodeID(i))
					}
					samples := make([]float64, b.N)
					var before, after gr.MemStats
					gr.ReadMemStats(&before)
					b.ReportAllocs()
					b.ResetTimer()
					b.StopTimer()
					for i := 0; i < b.N; i++ {
						at := time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC).Add(time.Duration(i) * time.Second)
						event := Event{ID: fmt.Sprintf("numeric-context-%d", i), Timestamp: at, Context: context, Cycles: 1}
						if !contextOnly {
							event.StimulusNodeIDs = inputs
							event.ResetContext = true
							event.EpisodeID = event.ID
						}
						if _, _, e := rt.Process(event); e != nil {
							b.Fatal(e)
						}
						rt.SetClock(NewFixedClock(at.Add(time.Millisecond)))
						b.StartTimer()
						start := time.Now()
						if _, e := rt.CognitiveThink(1); e != nil {
							b.Fatal(e)
						}
						samples[i] = float64(time.Since(start)) / 1e6
						b.StopTimer()
					}
					gr.ReadMemStats(&after)
					gateGBenchMetrics(b, "thought", samples)
					gateGBenchResources(b, before, after)
					b.ReportMetric(float64(len(rt.activation.CurrentState())), "result_active_nodes")
				})
			}
		}
	}
}

func BenchmarkGateGNativeStages(b *testing.B) {
	for _, n := range []int{1000, 4000, 40000} {
		for _, dense := range []bool{false, true} {
			b.Run(fmt.Sprintf("nodes=%d/dense=%t", n, dense), func(b *testing.B) {
				rt := telemetryBenchmarkRuntime(b, n, 4)
				w, e := OpenEventLog(filepath.Join(b.TempDir(), "events.jsonl"))
				if e != nil {
					b.Fatal(e)
				}
				defer w.Close()
				rt.SetEventLog(w)
				cfg := nativeControlConfig([]k.NodeID{1})
				cfg.Version = k.NativeControlCausalVersion
				cfg.Encoder = "local-relational-v1"
				cfg.MaxActiveNodes = 100000
				cfg.MaxSynapseReads = 1000000
				at := time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC)
				apply := func(c NativeControlCommand) NativeControlResult {
					o, e := rt.ApplyNativeControl(c, at.Add(time.Duration(rt.LastSequence()+1)*time.Microsecond))
					if e != nil {
						b.Fatal(e)
					}
					return o
				}
				apply(NativeControlCommand{ID: "init", Initialize: &cfg})
				count := 9
				if dense {
					count = n
				}
				state := map[k.NodeID]float64{}
				for i := 1; i <= count; i++ {
					state[k.NodeID(i)] = .8
				}
				samples, begin, operation, learning := make([]float64, b.N), make([]float64, b.N), make([]float64, b.N), make([]float64, b.N)
				var before, after gr.MemStats
				gr.ReadMemStats(&before)
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					start := time.Now()
					pair := fmt.Sprintf("pair%d", i)
					for trial := 0; trial < 2; trial++ {
						id := fmt.Sprintf("%s-%d", pair, trial)
						ts := time.Now()
						apply(NativeControlCommand{ID: id, Begin: &NativeBegin{EpisodeID: id, Training: true, Intervention: true, MatchKey: pair, Observation: &NativeObservation{Source: "gate-g-benchmark", SampleID: id, State: state}}})
						if trial == 0 {
							begin[i] = float64(time.Since(ts)) / 1e6
						}
						var out NativeControlResult
						actions := []int{0, 3}
						if trial == 1 {
							actions = []int{3}
						}
						for j, a := range actions {
							a := a
							ts = time.Now()
							out = apply(NativeControlCommand{ID: fmt.Sprintf("%s-%d", id, j), Step: &NativeStep{EpisodeID: id, ExperimentAction: &a}})
							if trial == 0 && j == 0 {
								operation[i] = float64(time.Since(ts)) / 1e6
							}
						}
						reward := .8
						if trial == 1 {
							reward = .2
						}
						ts = time.Now()
						apply(NativeControlCommand{ID: id + "-out", Outcome: &NativeOutcome{EpisodeID: id, TraceDigest: out.TraceDigest, Source: "gate-g-benchmark", OutcomeID: id, Provenance: "observed-environment", Reward: reward, Learn: true, ContrastID: pair, ContrastMode: "action"}})
						if trial == 1 {
							learning[i] = float64(time.Since(ts)) / 1e6
						}
					}
					samples[i] = float64(time.Since(start)) / 1e6
				}
				b.StopTimer()
				gr.ReadMemStats(&after)
				for name, values := range map[string][]float64{"pair": samples, "begin": begin, "operation": operation, "learning": learning} {
					gateGBenchMetrics(b, name, values)
				}
				gateGBenchResources(b, before, after)
			})
		}
	}
}

func BenchmarkGateGMixedHandoff(b *testing.B) {
	for _, n := range []int{1000, 4000, 40000} {
		b.Run(fmt.Sprintf("nodes=%d", n), func(b *testing.B) {
			rt := telemetryBenchmarkRuntime(b, n, 4)
			w, e := OpenEventLog(filepath.Join(b.TempDir(), "events.jsonl"))
			if e != nil {
				b.Fatal(e)
			}
			defer w.Close()
			rt.SetEventLog(w)
			cfg := nativeControlConfig([]k.NodeID{1})
			cfg.Version = k.NativeControlCausalVersion
			cfg.Encoder = "local-relational-v1"
			at := time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC)
			apply := func(c NativeControlCommand) NativeControlResult {
				o, e := rt.ApplyNativeControl(c, at.Add(time.Duration(rt.LastSequence()+1)*time.Microsecond))
				if e != nil {
					b.Fatal(e)
				}
				return o
			}
			apply(NativeControlCommand{ID: "init", Initialize: &cfg})
			samples, begins := make([]float64, b.N), make([]float64, b.N)
			var before, after gr.MemStats
			gr.ReadMemStats(&before)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				id := fmt.Sprintf("sample%d", i)
				s := testSensorObservation()
				s.SampleID = id
				s.Timestamp = at.Add(time.Duration(rt.LastSequence()+1) * time.Microsecond)
				start := time.Now()
				if _, e := rt.ProcessSensorObservation(s); e != nil {
					b.Fatal(e)
				}
				ts := time.Now()
				apply(NativeControlCommand{ID: id, Begin: &NativeBegin{EpisodeID: id}})
				begins[i] = float64(time.Since(ts)) / 1e6
				samples[i] = float64(time.Since(start)) / 1e6
				a := 3
				o := apply(NativeControlCommand{ID: id + "-stop", Step: &NativeStep{EpisodeID: id, ActionOverride: &a}})
				apply(NativeControlCommand{ID: id + "-out", Outcome: &NativeOutcome{EpisodeID: id, TraceDigest: o.TraceDigest, Source: "benchmark-sensor", OutcomeID: id, Provenance: "observed-sensor", Reward: 0}})
			}
			b.StopTimer()
			gr.ReadMemStats(&after)
			gateGBenchMetrics(b, "handoff", samples)
			gateGBenchMetrics(b, "begin", begins)
			gateGBenchResources(b, before, after)
		})
	}
}

func BenchmarkGateGJournalAppend(b *testing.B) {
	w, e := OpenEventLog(filepath.Join(b.TempDir(), "events.jsonl"))
	if e != nil {
		b.Fatal(e)
	}
	defer w.Close()
	samples := make([]float64, b.N)
	at := time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ev := LoggedEvent{SchemaVersion: EventLogSchemaVersion, BrainIdentity: BrainIdentity, Sequence: uint64(i + 1), Type: EventTypeThink, Timestamp: at, Event: &Event{Timestamp: at, Cycles: 1}}
		start := time.Now()
		if e = w.Append(ev); e != nil {
			b.Fatal(e)
		}
		samples[i] = float64(time.Since(start)) / 1e6
	}
	b.StopTimer()
	gateGBenchMetrics(b, "append", samples)
}
func gateGBenchMetrics(b *testing.B, name string, samples []float64) {
	sort.Float64s(samples)
	for _, p := range []float64{.5, .95, .99} {
		b.ReportMetric(samples[int(math.Ceil(float64(len(samples))*p))-1], fmt.Sprintf("%s_p%d_ms", name, int(p*100)))
	}
}
func gateGBenchResources(b *testing.B, before, after gr.MemStats) {
	b.ReportMetric(float64(after.NumGC-before.NumGC), "cell_GCs")
	b.ReportMetric(float64(after.PauseTotalNs-before.PauseTotalNs)/1e6, "cell_GC_pause_ms")
	var r syscall.Rusage
	if e := syscall.Getrusage(syscall.RUSAGE_SELF, &r); e == nil {
		b.ReportMetric(float64(r.Maxrss)/1024, "process_peak_RSS_MB")
	}
}
